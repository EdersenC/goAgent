const std = @import("std");
const Builder = std.build.Builder;

pub fn build(b: *Builder) void {
    const exe = b.addExecutable("my_project", "src/main.zig");
    exe.addModule("lib", b.addModule("lib", .{ .root_source_file = b.path("src/lib.zig") }));
    exe.addModule("utils", b.addModule("utils", .{ .root_source_file = b.path("src/utils.zig") }));

    exe.setBuildMode(.Debug);
    exe.install();

    const test_step = b.step("test", "Run unit tests");
    const tests = b.addTest("test/test_main.zig");
    test_step.dependOn(&tests.step);

    b.default_step.dependOn(&exe.step);
}

