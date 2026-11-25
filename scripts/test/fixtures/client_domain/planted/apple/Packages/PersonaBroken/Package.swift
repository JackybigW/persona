// swift-tools-version: 6.2
// A planted package for test_client_domain_modules.py: every Swift rule broken once. expected.txt lists the findings.
import PackageDescription

let package = Package(
    name: "PersonaBroken",
    platforms: [.macOS(.v15)],
    dependencies: [.package(path: "../PersonaDomainKit")],
    targets: [
        .target(name: "PersonaBrokenSupport"),
        .target(
            name: "PersonaBrokenDomain",
            dependencies: ["PersonaBrokenSupport", .product(name: "PersonaDomainKit", package: "PersonaDomainKit")],
            swiftSettings: [.swiftLanguageMode(.v6)]
        ),
    ]
)
