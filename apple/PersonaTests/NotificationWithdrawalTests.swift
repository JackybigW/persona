import Foundation
import Testing
@testable import Persona

/// Which pushes the app takes back from the notification center once their interaction is settled (#3513).
struct NotificationWithdrawalTests {
    private let t0 = Date(timeIntervalSince1970: 1_000_000)

    private func delivered(_ identifier: String, _ id: String, group: String = "", at seconds: TimeInterval = 0,
                           occurrence: String? = nil) -> NotificationWithdrawal.Delivered {
        .init(identifier: identifier, group: group, interactionID: id, date: t0.addingTimeInterval(seconds),
              interactionCreatedAt: occurrence)
    }

    private func interaction(_ id: String, occurrence: String) throws -> Interaction {
        try JSONDecoder().decode(Interaction.self, from: Data(
            #"{"id":"\#(id)","machine_id":"m","session_id":"s","kind":"permission","created_at":"\#(occurrence)"}"#.utf8))
    }

    @Test func anAnsweredInteractionTakesItsPushAlong() {
        // interaction_resolved (answered on the computer) or answered in the app: ix1 left the list.
        let out = NotificationWithdrawal.settled(gone: ["ix1"], pending: ["ix2"],
                                                 delivered: [delivered("n1", "ix1"), delivered("n2", "ix2"), delivered("n3", "ix3")])
        #expect(out == ["n1"])
    }

    @Test func anInteractionBackInTheListKeepsItsPush() {
        // The answer did not go through and the card came back before the notification center answered.
        #expect(NotificationWithdrawal.settled(gone: ["ix1"], pending: ["ix1"], delivered: [delivered("n1", "ix1")]).isEmpty)
    }

    @Test func theSameIDInAnotherGroupIsAnotherInteraction() {
        let pushes = [delivered("home", "ix1"), delivered("work", "ix1", group: "g2")]
        #expect(NotificationWithdrawal.settled(gone: ["g2|ix1"], pending: ["ix1"], delivered: pushes) == ["work"])
        #expect(NotificationWithdrawal.settled(gone: ["ix1"], pending: ["g2|ix1"], delivered: pushes) == ["home"])
        // A group forgotten: all of its interactions left at once.
        #expect(NotificationWithdrawal.settled(gone: ["g2|ix1", "g2|ix9"], pending: ["ix1"], delivered: pushes) == ["work"])
    }

    @Test func modelRemovalLeavesAReplacementAlertBeforeItsModelEvent() throws {
        let firstAt = "2026-01-02T03:04:05.000000001Z"
        let secondAt = "2026-01-02T03:04:05.000000002Z"
        let first = try interaction("ix1", occurrence: firstAt)
        let second = try interaction("ix1", occurrence: secondAt)
        let pushes = [delivered("old", "ix1", occurrence: firstAt), delivered("new", "ix1", occurrence: secondAt)]
        let gone = [first.uid: first]
        #expect(NotificationWithdrawal.settledOccurrences(gone: gone, pending: [:], delivered: pushes) == ["old"])
        #expect(NotificationWithdrawal.settledOccurrences(gone: gone, pending: [second.uid: second], delivered: pushes) == ["old"])
        #expect(NotificationWithdrawal.settledOccurrences(gone: gone, pending: [first.uid: first], delivered: [pushes[0]]).isEmpty)
    }

    @Test func aGroupsWholeListTakesBackWhatItNoLongerHas() {
        // Reconnected: the list says ix2 is still pending; ix1 was answered while the socket was down.
        let out = NotificationWithdrawal.stale(listedAt: ["": t0.addingTimeInterval(60)], pending: ["ix2"],
                                               delivered: [delivered("n1", "ix1"), delivered("n2", "ix2")])
        #expect(out == ["n1"])
    }

    @Test func aPushNewerThanTheListStays() {
        // Delivered after the list was asked for: its interaction can be ahead of its event.
        let out = NotificationWithdrawal.stale(listedAt: ["": t0], pending: [], delivered: [delivered("n1", "ix1", at: 5)])
        #expect(out.isEmpty)
    }

    @Test func onlyGroupsWithAWholeListAreChecked() {
        // g2 has not answered yet: its list is the cache of the last run, which says nothing about its pushes.
        let pushes = [delivered("home", "ix1"), delivered("work", "ix7", group: "g2")]
        let out = NotificationWithdrawal.stale(listedAt: ["": t0.addingTimeInterval(60)], pending: [], delivered: pushes)
        #expect(out == ["home"])
        let both = NotificationWithdrawal.stale(listedAt: ["": t0.addingTimeInterval(60), "g2": t0.addingTimeInterval(60)],
                                                pending: ["g2|ix7"], delivered: pushes)
        #expect(both == ["home"])
    }

    @Test func uidMatchesTheInteractions() {
        #expect(delivered("n", "ix1").uid == "ix1")
        #expect(delivered("n", "ix1", group: "g2").uid == "g2|ix1")
    }

    @Test func aWithdrawalPushTakesBackWhatItNames() {
        // The hub says ix1 and ix3 are over (#4349); the suspended app still lists ix1 as pending.
        let pushes = [delivered("n1", "ix1"), delivered("n2", "ix2"), delivered("n3", "ix3", group: "g2")]
        #expect(NotificationWithdrawal.resolved(["ix1", "ix3"], group: "", delivered: pushes) == ["n1"])
        #expect(NotificationWithdrawal.resolved(["ix1", "ix3"], group: "g2", delivered: pushes) == ["n3"])
        // A push that did not say whose: the ids in any group.
        #expect(NotificationWithdrawal.resolved(["ix1", "ix3"], group: nil, delivered: pushes) == ["n1", "n3"])
        #expect(NotificationWithdrawal.resolved([], group: nil, delivered: pushes).isEmpty)
    }

    @Test func aDelayedWithdrawalDoesNotRemoveItsSameIDReplacement() {
        let first = "2026-01-02T03:04:05.000000001Z"
        let second = "2026-01-02T03:04:05.000000002Z"
        let pushes = [delivered("old", "ix1", occurrence: first), delivered("new", "ix1", occurrence: second)]
        let over = [NotificationWithdrawal.Resolved(interactionID: "ix1", interactionCreatedAt: first)]
        #expect(NotificationWithdrawal.resolvedOccurrences(over, group: "", delivered: pushes) == ["old"])
        // A pre-occurrence alert can still be collected after an upgrade; a known replacement cannot.
        let legacy = delivered("legacy", "ix1")
        #expect(NotificationWithdrawal.resolvedOccurrences(over, group: "", delivered: [legacy]) == ["legacy"])
    }

    @Test func anAssistantCardsPushWaitsForItsWithdrawal() {
        // A decision's push (card_id): never in the list of pending interactions, so neither a list without it nor a
        // settled interaction takes it back; the hub's withdrawal push naming it does.
        var card = delivered("n1", "todo-ask:t1")
        card.ownerCard = true
        let pushes = [card, delivered("n2", "ix2")]
        #expect(NotificationWithdrawal.stale(listedAt: ["": t0.addingTimeInterval(60)], pending: [], delivered: pushes) == ["n2"])
        #expect(NotificationWithdrawal.settled(gone: ["todo-ask:t1"], pending: [], delivered: pushes).isEmpty)
        #expect(NotificationWithdrawal.resolved(["todo-ask:t1"], group: "", delivered: pushes) == ["n1"])
    }
}
