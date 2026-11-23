import AVFoundation
import PersonaKit

/// The audio route as `VoiceRoute` (PersonaKit) sees it, read through the broker: the voice conversations follow it, and
/// a meeting's spoken answers ask it whether to move to the speaker (`MeetingSpeaker`). In the core so recording needs
/// nothing of voice for it (#13431).
nonisolated extension AudioSessionBroker {
    /// The session's outputs, inputs and the inputs it offers, as ports and ids.
    var routeSnapshot: VoiceRoute.Snapshot {
        let route = currentRoute
        return VoiceRoute.Snapshot(outputs: route.outputs.map(Self.device), inputs: route.inputs.map(Self.device),
                                   available: availableInputs.map(Self.device))
    }

    private static func device(_ description: AVAudioSessionPortDescription) -> VoiceRoute.Device {
        VoiceRoute.Device(port: port(description.portType), id: description.uid, name: description.portName)
    }

    static func port(_ port: AVAudioSession.Port) -> VoiceRoute.Port {
        switch port {
        case .builtInMic: .builtInMic
        case .builtInReceiver: .builtInReceiver
        case .builtInSpeaker: .builtInSpeaker
        case .headphones: .headphones
        case .headsetMic: .headsetMic
        case .bluetoothHFP: .bluetoothHFP
        case .bluetoothA2DP: .bluetoothA2DP
        case .bluetoothLE: .bluetoothLE
        case .usbAudio: .usbAudio
        case .carAudio: .carAudio
        default: .other
        }
    }
}
