import Foundation

/// What a module's client of the shared audio session uses it for (`ModuleSlot.audioLeases`), ranked as the core's
/// `AudioSessionBroker` ranks its clients: a system call over a voice conversation over a recording over playback. The
/// raw values are the ranks, the same as the app's `AudioSessionUse`.
public enum AudioLeaseUse: Int, CaseIterable, Comparable, Sendable {
    case playback = 0
    case recording = 1
    case conversation = 2
    case call = 3

    public static func < (a: AudioLeaseUse, b: AudioLeaseUse) -> Bool { a.rawValue < b.rawValue }
}

/// A client of the shared audio session a module declared: the module, the name its audio code gives the broker, and
/// what it uses the session for.
public struct AudioLease: Hashable, Sendable {
    public let module: ModuleID
    public let name: String
    public let use: AudioLeaseUse

    public init(module: ModuleID, name: String, use: AudioLeaseUse) {
        self.module = module
        self.name = name
        self.use = use
    }
}
