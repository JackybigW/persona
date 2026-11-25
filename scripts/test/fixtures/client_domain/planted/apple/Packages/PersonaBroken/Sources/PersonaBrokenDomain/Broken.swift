import Foundation
@_spi(Clock) import PersonaDomainKit
import PersonaBrokenSupport

public enum State { case working }

public struct Broken: Codable, CustomStringConvertible {
    public let title: String
    public private(set) var count: Int
    public var doneAt: Date { Date(timeIntervalSince1970: 0) }
    public var description: String { title }

    public init(title: String, count: Int) {
        self.title = title
        self.count = count
    }

    public static func make(a: Instant, b: Instant, c: Instant, d: Instant, e: Instant) -> Broken {
        Broken(title: "", count: 0)
    }

    public mutating func bump() { count += 1 }

    public func save() throws { print(title) }

    public static func evolve(_ broken: Broken, _ state: State) -> Broken { broken }
}

public struct Box<Wrapped: Sendable>: Sendable {
    private let box: Wrapped
}

public extension Broken {
    enum Hidden { case hidden }
}
