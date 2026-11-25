package persona.core.broken.domain

import java.util.UUID
import kotlinx.serialization.Serializable
import persona.core.domainkit.DomainBoundary
import persona.core.domainkit.Instant
import persona.core.domainkit.Now
import persona.core.planted.domain.ItemId

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

    public fun id(): String = "broken"

    public fun wakes(now: Now): Instant = at

    public operator fun get(item: ItemId): Instant = at

    @IgnorableReturnValue
    public fun touched(): Broken = this

    public fun whenDue(effect: (Broken) -> Unit): Unit = effect(this)

    public fun due(): Boolean = persona.core.domainkit.adapt.SystemClock().now().reached(at)

    public fun pick(options: List<Instant>): Broken = Broken(options.shuffled().first())

    public fun <T> carry(value: T): Broken = this

    @DomainBoundary
    public fun raw(): Long = 0

    public companion object {
        public fun fromWire(at: Instant): Broken = Broken(at)

        public fun blank(at: Instant): Broken = Broken(at)

        public fun new(a: Instant, b: Instant, c: Instant, d: Instant, e: Instant): Broken = Broken(a)
    }
}

public interface BrokenStore {
    public fun save(broken: Broken)
}

public abstract class Shape

public interface BrokenVisitor {
    public fun read(broken: Broken)
}

public object Registry {
    private val seen: MutableSet<Instant> = HashSet()
}

private val cache = mutableMapOf<Instant, Broken>()
