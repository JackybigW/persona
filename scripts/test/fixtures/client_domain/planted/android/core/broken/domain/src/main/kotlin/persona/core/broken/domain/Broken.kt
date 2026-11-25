package persona.core.broken.domain

import java.util.UUID
import kotlinx.serialization.Serializable
import persona.core.domainkit.DomainBoundary
import persona.core.domainkit.Instant

@Serializable
public data class Record(val doneAt: Long)

public sealed interface State

public enum class Kind { ONE, TWO }

public class Open(public val name: String)

public class Broken private constructor(private val at: Instant) {
    public var count: Int = 0

    public fun doneAt(): Instant = at

    public fun rename(name: String): Broken = this

    public fun stamp(): Broken {
        println(UUID.randomUUID())
        if (System.currentTimeMillis() < 0) throw IllegalStateException()
        return this
    }

    public fun evolve(state: State): Broken = this

    @OptIn(DomainBoundary::class)
    public fun leak(): Long = at.wire()

    public companion object {
        public fun fromWire(at: Instant): Broken = Broken(at)

        public fun new(a: Instant, b: Instant, c: Instant, d: Instant, e: Instant): Broken = Broken(a)
    }
}
