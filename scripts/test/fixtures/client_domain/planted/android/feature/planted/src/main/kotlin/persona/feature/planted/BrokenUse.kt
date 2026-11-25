package persona.feature.planted

import persona.core.domainkit.DomainBoundary
import persona.core.domainkit.Instant
import persona.core.planted.domain.ItemId
import persona.core.planted.domain.ItemVisitor
import persona.core.planted.domain.Urgency

// App code breaking the rules outside the domain modules. expected.txt lists the findings.
@OptIn(DomainBoundary::class)
internal fun leak(id: ItemId): String = id.wire()

internal class RowReader : ItemVisitor {
    override fun visitItem(id: ItemId, urgency: Urgency, wakes: Instant) = Unit
}
