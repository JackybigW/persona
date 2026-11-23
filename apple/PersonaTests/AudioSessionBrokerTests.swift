import AVFoundation
import os
import PersonaModuleAPI
import Testing
@testable import Persona

/// The core's audio session broker (#13849 item 3, #13431): every call reaches the session exactly as the client asked,
/// in its order, and the broker keeps one lease per client, ranked call > conversation > recording > playback.
@MainActor
struct AudioSessionBrokerTests {
    /// Notes every call the broker makes; a failure can be armed for one kind of call.
    private final class Recorder: AudioSessionControl, @unchecked Sendable {
        let calls = OSAllocatedUnfairLock<[String]>(initialState: [])
        let failActivation = OSAllocatedUnfairLock(initialState: false)
        struct Refused: Error {}

        func setCategory(_ category: AVAudioSession.Category, mode: AVAudioSession.Mode, options: AVAudioSession.CategoryOptions) throws {
            calls.withLock { $0.append("category \(category.rawValue) \(mode.rawValue) \(options.rawValue)") }
        }

        func setActive(_ active: Bool, options: AVAudioSession.SetActiveOptions) throws {
            if active, failActivation.withLock({ $0 }) { throw Refused() }
            calls.withLock { $0.append("active \(active) \(options.rawValue)") }
        }

        var category: AVAudioSession.Category { .soloAmbient }
        var currentRoute: AVAudioSessionRouteDescription { AVAudioSession.sharedInstance().currentRoute }
        var isOtherAudioPlaying: Bool { false }
        var isInputAvailable: Bool { true }
        var outputLatency: TimeInterval { 0 }
        var ioBufferDuration: TimeInterval { 0 }
        func requestRecordPermission(_ answer: @escaping @Sendable (Bool) -> Void) { answer(true) }
        var availableInputs: [AVAudioSessionPortDescription] { [] }
        func setPreferredInput(_ port: AVAudioSessionPortDescription?) throws {
            calls.withLock { $0.append("input \(port?.uid ?? "none")") }
        }
        func overrideOutputAudioPort(_ port: AVAudioSession.PortOverride) throws {
            calls.withLock { $0.append("override \(port.rawValue)") }
        }
    }

    private let voice = AudioSessionClient("voice", use: .conversation)
    private let dictation = AudioSessionClient("dictation", use: .recording)
    private let replay = AudioSessionClient("meeting-replay", use: .playback)

    @Test func callsReachTheSessionAsAskedAndInOrder() throws {
        let session = Recorder()
        let broker = AudioSessionBroker(session: session)
        try broker.setCategory(.playAndRecord, mode: .voiceChat, options: [.allowBluetoothHFP], for: voice)
        try broker.activate(for: voice)
        try broker.overrideOutput(.speaker, for: voice)
        try broker.setPreferredInput(nil, for: voice)
        try broker.deactivate(for: voice)
        try broker.deactivate(for: voice, notifyingOthers: false)
        let notify = AVAudioSession.SetActiveOptions.notifyOthersOnDeactivation.rawValue
        #expect(session.calls.withLock { $0 } == [
            "category \(AVAudioSession.Category.playAndRecord.rawValue) \(AVAudioSession.Mode.voiceChat.rawValue) \(AVAudioSession.CategoryOptions.allowBluetoothHFP.rawValue)",
            "active true 0",
            "override \(AVAudioSession.PortOverride.speaker.rawValue)",
            "input none",
            "active false \(notify)",
            "active false 0",
        ])
    }

    @Test func dictationsActivationKeepsItsOptions() throws {
        let session = Recorder()
        let broker = AudioSessionBroker(session: session)
        try broker.activate(for: dictation, options: .notifyOthersOnDeactivation)
        #expect(session.calls.withLock { $0 } == ["active true \(AVAudioSession.SetActiveOptions.notifyOthersOnDeactivation.rawValue)"])
    }

    @Test func aLeaseRunsFromActivationToDeactivation() throws {
        let broker = AudioSessionBroker(session: Recorder())
        #expect(broker.holders.isEmpty)
        #expect(broker.owner == nil)
        try broker.activate(for: replay)
        try broker.activate(for: voice)
        #expect(broker.holders == [replay, voice])
        #expect(broker.owner == voice)
        try broker.activate(for: replay) // again: one lease, moved to the end
        #expect(broker.holders == [voice, replay])
        #expect(broker.owner == voice, "playback does not outrank a conversation")
        try broker.deactivate(for: voice)
        #expect(broker.owner == replay)
        try broker.deactivate(for: replay)
        #expect(broker.owner == nil)
    }

    @Test func theRankIsCallConversationRecordingPlayback() throws {
        let broker = AudioSessionBroker(session: Recorder())
        try broker.activate(for: dictation)
        broker.systemActivated(for: .systemCall)
        try broker.activate(for: voice)
        #expect(broker.owner == .systemCall)
        broker.systemDeactivated(for: .systemCall)
        #expect(broker.owner == voice)
        #expect(AudioSessionUse.allCases.sorted() == [.playback, .recording, .conversation, .call])
    }

    @Test func aRefusedActivationTakesNoLease() {
        let session = Recorder()
        session.failActivation.withLock { $0 = true }
        let broker = AudioSessionBroker(session: session)
        #expect(throws: Recorder.Refused.self) { try broker.activate(for: voice) }
        #expect(broker.holders.isEmpty)
    }

    /// A module's lease (`audio_leases`) ranks as the core's client of the same use: the two enums keep the same ranks.
    @Test func aModulesLeaseMapsOntoTheBrokersRanks() {
        for use in AudioLeaseUse.allCases {
            let client = AudioSessionClient(AudioLease(module: "voice", name: "conversation", use: use))
            #expect(client.use.rawValue == use.rawValue)
            #expect(client.name == "voice.conversation")
        }
        #expect(AudioLeaseUse.allCases.count == AudioSessionUse.allCases.count)
    }
}
