package(default_visibility = ["//visibility:private"])

licenses(["notice"])

load("@io_bazel_rules_go//go:def.bzl", "go_test")

go_test(
    name = "go_default_test",
    srcs = ["primitives_test.go"],
    deps = [
        "//go/jsonfhir:go_default_library",
        "//proto/stu3:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)
