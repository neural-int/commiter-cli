// swift-tools-version: 6.2

import PackageDescription

let package = Package(
    name: "CommiterMLXHelper",
    platforms: [
        .macOS(.v14),
    ],
    dependencies: [
        // MLXGuidedGeneration is currently published from mlx-swift-lm's
        // main branch. Keep the revision explicit until the product is in a
        // stable release.
        .package(
            url: "https://github.com/ml-explore/mlx-swift-lm.git",
            revision: "c6446cf7bfb7cea76408013b614d4b2c530eaa03"
        ),
        .package(
            url: "https://github.com/huggingface/swift-huggingface.git",
            exact: "0.11.0"
        ),
        .package(
            url: "https://github.com/huggingface/swift-transformers.git",
            exact: "1.3.4"
        ),
    ],
    targets: [
        .target(
            name: "CommiterMLXHelperProtocol"
        ),
        .executableTarget(
            name: "commiter-mlx-helper",
            dependencies: [
                "CommiterMLXHelperProtocol",
                .product(name: "MLXGuidedGeneration", package: "mlx-swift-lm"),
                .product(name: "MLXHuggingFace", package: "mlx-swift-lm"),
                .product(name: "MLXLLM", package: "mlx-swift-lm"),
                .product(name: "MLXLMCommon", package: "mlx-swift-lm"),
                .product(name: "HuggingFace", package: "swift-huggingface"),
                .product(name: "Transformers", package: "swift-transformers"),
            ]
        ),
        .testTarget(
            name: "CommiterMLXHelperProtocolTests",
            dependencies: ["CommiterMLXHelperProtocol"]
        ),
    ]
)
