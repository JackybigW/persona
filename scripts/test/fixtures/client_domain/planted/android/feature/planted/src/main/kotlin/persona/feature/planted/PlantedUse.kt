package persona.feature.planted

import persona.core.domainkit.Instant
import persona.core.domainkit.Now
import persona.core.planted.domain.Item

/** App code using the planted domain: it keeps the next item a snooze returns, or the item when the snooze is refused. */
internal fun snooze(item: Item, until: Instant, now: Now): Item = item.snoozed(until, now).fold({ item }, { it })
