import AVFoundation
import os

/// What a client of the shared audio session uses it for, ranked: a system call outranks a voice conversation, which
/// outranks a recording (a meeting, a note, dictation), which outranks playing something back (#13431, #13849 item 3).
/// The raw values are the ranks; `PersonaModuleAPI`'s `AudioLeaseUse` (the `audio_leases` slot) has the same cases and
/// values, so a module's lease maps onto one of these (`ModulesTests` checks it).
nonisolated enum AudioSessionUse: Int, Comparable, CaseIterable, Sendable, CustomStringConvertible {
    case playback = 0
    case recording = 1
    case conversation = 2
    case call = 3

    static func < (a: AudioSessionUse, b: AudioSessionUse) -> Bool { a.rawValue < b.rawValue }

    var description: String {
        switch self {
        case .playback: "playback"
        case .recording: "recording"
        case .conversation: "conversation"
        case .call: "call"
        }
    }
}

/// One client of the shared audio session: a name for the logs and the lease list, and what it uses the session for.
nonisolated struct AudioSessionClient: Hashable, Sendable, CustomStringConvertible {
    let name: String
    let use: AudioSessionUse

    init(_ name: String, use: AudioSessionUse) {
        self.name = name
        self.use = use
    }

    var description: String { "\(name) (\(use))" }
}

/// The calls the broker makes on the audio session: the app's shared `AVAudioSession`, or a test's stand-in that notes
/// what was asked.
nonisolated protocol AudioSessionControl: Sendable {
    func setCategory(_ category: AVAudioSession.Category, mode: AVAudioSession.Mode, options: AVAudioSession.CategoryOptions) throws
    func setActive(_ active: Bool, options: AVAudioSession.SetActiveOptions) throws
    var category: AVAudioSession.Category { get }
    var currentRoute: AVAudioSessionRouteDescription { get }
    var isOtherAudioPlaying: Bool { get }
    var isInputAvailable: Bool { get }
    var outputLatency: TimeInterval { get }
    var ioBufferDuration: TimeInterval { get }
    func requestRecordPermission(_ answer: @escaping @Sendable (Bool) -> Void)
    #if os(iOS) || os(visionOS)
    var availableInputs: [AVAudioSessionPortDescription] { get }
    func setPreferredInput(_ port: AVAudioSessionPortDescription?) throws
    func overrideOutputAudioPort(_ port: AVAudioSession.PortOverride) throws
    #endif
}

/// The system's shared session. `AVAudioSession` is thread-safe; the broker is called on the main actor and off it.
nonisolated struct SystemAudioSession: AudioSessionControl {
    private var session: AVAudioSession { AVAudioSession.sharedInstance() }

    func setCategory(_ category: AVAudioSession.Category, mode: AVAudioSession.Mode, options: AVAudioSession.CategoryOptions) throws {
        try session.setCategory(category, mode: mode, options: options)
    }

    func setActive(_ active: Bool, options: AVAudioSession.SetActiveOptions) throws {
        try session.setActive(active, options: options)
    }

    var category: AVAudioSession.Category { session.category }
    var currentRoute: AVAudioSessionRouteDescription { session.currentRoute }
    var isOtherAudioPlaying: Bool { session.isOtherAudioPlaying }
    var isInputAvailable: Bool { session.isInputAvailable }
    var outputLatency: TimeInterval { session.outputLatency }
    var ioBufferDuration: TimeInterval { session.ioBufferDuration }
    func requestRecordPermission(_ answer: @escaping @Sendable (Bool) -> Void) { session.requestRecordPermission(answer) }

    #if os(iOS) || os(visionOS)
    var availableInputs: [AVAudioSessionPortDescription] { session.availableInputs ?? [] }
    func setPreferredInput(_ port: AVAudioSessionPortDescription?) throws { try session.setPreferredInput(port) }
    func overrideOutputAudioPort(_ port: AVAudioSession.PortOverride) throws { try session.overrideOutputAudioPort(port) }
    #endif
}

/// The one place the app talks to the shared audio session (#13849 item 3, #13431): the voice conversations, system
/// calls, CarPlay, recordings, the composer's dictation, meeting playback and the screen's Picture in Picture all set the
/// session up and let it go through here, never through `AVAudioSession` themselves
/// (`scripts/test/test_apple_audio_session.py` keeps it that way). Modules declare their leases through the
/// `audio_leases` slot of the module registry.
///
/// Each client says who it is and what it uses the session for (`AudioSessionClient`), and the broker keeps a lease per
/// client from its activation to its deactivation, ranked call > conversation > recording > playback (`owner`). The
/// broker decides nothing yet: every call goes to the session exactly as the client asked, in the order it asked, and a
/// lower-ranked client that activates over a higher one is only logged. The gates that exist (`LiveCallGate`,
/// `RecordingMicrophone`) still decide; moving those decisions here is a later change.
///
/// The session's own queue (`queue`) is where dictation, meeting playback and a call's category take turns.
nonisolated final class AudioSessionBroker: Sendable {
    static let shared = AudioSessionBroker(session: SystemAudioSession())

    /// The audio session's own queue: dictation opens and lets go of its session here, meeting playback sets its category
    /// here, and the call sets its category here (`PersonaCalls.setCallCategory`), so one never runs in the middle of the
    /// other (`LiveCallGate.liveNow` is asked on it first).
    static let queue = DispatchQueue(label: "persona.audio-session")

    private let session: any AudioSessionControl
    private let leases = OSAllocatedUnfairLock<[AudioSessionClient]>(initialState: [])
    private static let log = Logger(subsystem: "persona", category: "audio-session")

    init(session: any AudioSessionControl) {
        self.session = session
    }

    // MARK: Leases

    /// The clients holding the session now, in the order they activated it.
    var holders: [AudioSessionClient] { leases.withLock { $0 } }

    /// The highest-ranked holder (the latest of equal rank), nil when nobody holds the session.
    var owner: AudioSessionClient? {
        leases.withLock { list in list.reversed().max { $0.use < $1.use } }
    }

    private func took(_ client: AudioSessionClient) {
        let above: AudioSessionClient? = leases.withLock { list in
            list.removeAll { $0 == client }
            let above = list.filter { $0.use > client.use }.max { $0.use < $1.use }
            list.append(client)
            return above
        }
        if let above {
            Self.log.notice("audio: \(client.description, privacy: .public) activates the session while \(above.description, privacy: .public) holds it")
        }
    }

    private func released(_ client: AudioSessionClient) {
        leases.withLock { $0.removeAll { $0 == client } }
    }

    /// The system activated the session for `client` (a call's session, handed over by the call service): a lease with no
    /// call to the session.
    func systemActivated(for client: AudioSessionClient) { took(client) }

    /// The system took the session back from `client`.
    func systemDeactivated(for client: AudioSessionClient) { released(client) }

    // MARK: Setting the session up

    func setCategory(_ category: AVAudioSession.Category, mode: AVAudioSession.Mode = .default,
                     options: AVAudioSession.CategoryOptions = [], for client: AudioSessionClient) throws {
        try session.setCategory(category, mode: mode, options: options)
    }

    /// Activates the session for `client` and records its lease.
    func activate(for client: AudioSessionClient, options: AVAudioSession.SetActiveOptions = []) throws {
        try session.setActive(true, options: options)
        took(client)
    }

    /// Deactivates the session for `client` (by default telling the others, music, they may go on) and ends its lease,
    /// whether or not the session would let go.
    func deactivate(for client: AudioSessionClient, notifyingOthers: Bool = true) throws {
        released(client)
        try session.setActive(false, options: notifyingOthers ? .notifyOthersOnDeactivation : [])
    }

    #if os(iOS) || os(visionOS)
    func setPreferredInput(_ port: AVAudioSessionPortDescription?, for client: AudioSessionClient) throws {
        try session.setPreferredInput(port)
    }

    func overrideOutput(_ port: AVAudioSession.PortOverride, for client: AudioSessionClient) throws {
        try session.overrideOutputAudioPort(port)
    }

    /// The inputs the session offers now (none when it lists none).
    var availableInputs: [AVAudioSessionPortDescription] { session.availableInputs }
    #endif

    // MARK: Reading the session

    var category: AVAudioSession.Category { session.category }
    var currentRoute: AVAudioSessionRouteDescription { session.currentRoute }
    /// Another app's audio plays (a VoIP call without CallKit, music).
    var isOtherAudioPlaying: Bool { session.isOtherAudioPlaying }
    var isInputAvailable: Bool { session.isInputAvailable }
    /// How long after a render the output is heard, and the I/O buffer's length (the voice's logs, #13963).
    var outputLatency: TimeInterval { session.outputLatency }
    var ioBufferDuration: TimeInterval { session.ioBufferDuration }

    /// The older systems' microphone permission prompt (`AVAudioApplication` replaces it from iOS 17 / watchOS 10).
    func requestRecordPermission(_ answer: @escaping @Sendable (Bool) -> Void) { session.requestRecordPermission(answer) }
}
