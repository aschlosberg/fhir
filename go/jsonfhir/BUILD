package(default_visibility = ["//visibility:public"])

licenses(["notice"])

load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["jsonfhir.go"],
    importpath = "github.com/google/fhir/go/jsonfhir",
    deps = [
        "//proto/stu3:go_default_library",
        "@com_github_golang_protobuf//descriptor:go_default_library_gen",
        "@com_github_golang_protobuf//proto:go_default_library",
        "@io_bazel_rules_go//proto/wkt:descriptor_go_proto",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "interfaces_test.go",
        "jsonfhir_test.go",
    ],
    data = [
        "//testdata/stu3:examples",
        "//testdata/stu3:ndjson",
    ],
    embed = [":go_default_library"],
    deps = [
        "//proto/stu3:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)
