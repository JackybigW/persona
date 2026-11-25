import PersonaDomainKit
import PersonaPlantedDomain

/// App code using the planted domain: it keeps the next item a snooze returns, and switches on the Outcome at the edge.
struct PlantedRow {
    func snooze(_ item: Item, until: Instant, now: Now) -> Item {
        switch item.snoozed(until: until, at: now) {
        case .ok(let next): return next
        case .failed: return item
        }
    }
}
