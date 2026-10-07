#!/usr/bin/env python3
"""Check Make's cache invalidation in temporary workspaces without SDK toolchains."""

import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[1]


class BuildDependenciesTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.repo = Path(self.temp.name)
        shutil.copy2(ROOT / "Makefile", self.repo / "Makefile")
        for path in (ROOT / "scripts").glob("*"):
            if path.suffix in (".mk", ".sh"):
                dest = self.repo / path.relative_to(ROOT)
                dest.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(path, dest)
        inputs = ["README.md", "mise.toml", ".config/mise.toml",
                  "provider/cmd/pulumi-resource-logfire/schema.json",
                  "provider/cmd/pulumi-resource-logfire/bridge-metadata.json"]
        inputs += [str(p.relative_to(ROOT)) for p in (ROOT / "provider").rglob("*")
                   if p.suffix == ".go" or p.name in ("go.mod", "go.sum")]
        for name in inputs:
            dest = self.repo / name
            dest.parent.mkdir(parents=True, exist_ok=True)
            dest.write_text("original\n")

    def configure(self, target, recipe):
        # Substitute the expensive compilers, retaining the real dependency graph.
        targets = ["bin/pulumi-tfgen-logfire", "bin/pulumi-resource-logfire",
                   ".make/mise_install", ".make/schema", "bin/jsign-6.0.jar",
                   ".make/generate_go", ".make/generate_nodejs", ".make/generate_python"]
        overlay = "mise_env:\n\t@true\n"
        for name in targets:
            overlay += f"{name}:\n\t@mkdir -p $(dir $@)\n\t@touch $@\n"
        overlay += f"{target}:\n\t@mkdir -p $(dir $@)\n\t@{recipe}\n\t@touch $@\n"
        (self.repo / "check.mk").write_text(overlay)

    def make(self, target, *settings):
        result = subprocess.run(
            ["make", "-s", "-f", "Makefile", "-f", "check.mk", target, *settings],
            cwd=self.repo, text=True, capture_output=True,
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def age_workspace(self):
        for path in self.repo.rglob("*"):
            if path.is_file():
                os.utime(path, (1_000_000_000, 1_000_000_000))

    def assert_rebuild(self, changed, target):
        source = self.repo / changed
        source.write_text("original\n")
        (self.repo / target).unlink(missing_ok=True)
        self.configure(target, f"cat {changed} > result.txt")
        self.make(target)
        self.assertEqual((self.repo / "result.txt").read_text(), "original\n")
        self.age_workspace()
        source.write_text("updated\n")
        os.utime(source, (1_000_000_010, 1_000_000_010))
        self.make(target)
        self.assertEqual((self.repo / "result.txt").read_text(), "updated\n")

    def test_provider_inputs_regenerate_schema(self):
        for changed in ("provider/shim/provider.go", "provider/shim/go.mod",
                        "provider/cmd/pulumi-tfgen-logfire/main.go",
                        "provider/pkg/version/version.go"):
            with self.subTest(changed=changed):
                self.assert_rebuild(changed, ".make/schema")

    def test_dashboard_helper_regenerates_schema(self):
        self.assert_rebuild("scripts/fix_dashboard_go_examples.sh", ".make/schema")

    def test_dashboard_helper_regenerates_go_sdk(self):
        self.assert_rebuild("scripts/fix_dashboard_go_examples.sh", ".make/generate_go")

    def test_readme_regenerates_python_sdk(self):
        self.assert_rebuild("README.md", ".make/generate_python")

    def test_tool_configuration_reinstalls_tools(self):
        for changed in ("mise.toml", ".config/mise.toml", "provider/go.mod"):
            with self.subTest(changed=changed):
                self.assert_rebuild(changed, ".make/mise_install")

    def test_provider_entry_point_rebuilds_binary(self):
        self.assert_rebuild("provider/cmd/pulumi-resource-logfire/main.go",
                            "bin/pulumi-resource-logfire")

    def test_embedded_schema_rebuilds_binary(self):
        self.assert_rebuild("provider/cmd/pulumi-resource-logfire/schema.json",
                            "bin/pulumi-resource-logfire")

    def test_provider_version_changes_rebuild_binary(self):
        self.configure("bin/pulumi-resource-logfire", "printf '%s\\n' '$(PROVIDER_VERSION)' > result.txt")
        self.make("provider", "PROVIDER_VERSION=0.1.0")
        self.assertEqual((self.repo / "result.txt").read_text(), "0.1.0\n")
        self.age_workspace()
        self.make("provider", "PROVIDER_VERSION=0.2.0")
        self.assertEqual((self.repo / "result.txt").read_text(), "0.2.0\n")
        self.make("provider", "PROVIDER_VERSION=0.1.0")
        self.assertEqual((self.repo / "result.txt").read_text(), "0.1.0\n")

    def test_conversion_setting_changes_regenerate_schema(self):
        self.configure(".make/schema", "printf '%s\\n' '$(PULUMI_CONVERT)' > result.txt")
        self.make("schema", "PULUMI_CONVERT=0")
        self.age_workspace()
        self.make("schema", "PULUMI_CONVERT=1")
        self.assertEqual((self.repo / "result.txt").read_text(), "1\n")

    def test_unchanged_inputs_keep_cached_artifacts(self):
        self.configure(".make/schema", "printf 'generated\\n' >> result.txt")
        self.make("schema")
        self.make("schema")
        self.assertEqual((self.repo / "result.txt").read_text(), "generated\n")

    def test_ci_restore_preserves_cache_until_settings_change(self):
        self.configure(".make/schema", "printf 'generated\\n' >> result.txt")
        self.make("schema")
        self.make("prepare_local_workspace")
        self.make("schema", "--touch")
        self.make("schema")
        self.assertEqual((self.repo / "result.txt").read_text(), "generated\n")
        self.make("schema", "PROVIDER_VERSION=0.2.0")
        self.assertEqual((self.repo / "result.txt").read_text(), "generated\ngenerated\n")

    def test_provider_entry_point_rebuilds_cross_compiled_binary(self):
        self.assert_rebuild("provider/cmd/pulumi-resource-logfire/main.go",
                            "bin/linux-amd64/pulumi-resource-logfire")

    def test_clean_removes_all_sdk_directories(self):
        for language in ("go", "nodejs", "python"):
            (self.repo / "sdk" / language).mkdir(parents=True)
            (self.repo / "sdk" / language / "artifact").write_text("built\n")
        self.assertEqual(sorted(p.name for p in (self.repo / "sdk").iterdir()),
                         ["go", "nodejs", "python"])
        self.configure(".make/schema", "true")
        self.make("clean")
        self.assertEqual(list((self.repo / "sdk").iterdir()), [])


class GeneratedArtifactsHookTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.repo = Path(self.temp.name)
        (self.repo / "sdk").mkdir()
        (self.repo / "sdk/existing.py").write_text("original\n")
        (self.repo / "Makefile").write_text("PROVIDER_VERSION ?= 0.1.0\n")
        (self.repo / ".ci-mgmt.yaml").write_text("major-version: 0\n")
        subprocess.run(["git", "init", "-q", str(self.repo)], check=True)
        subprocess.run(["git", "add", "."], cwd=self.repo, check=True)
        fake_tools = self.repo / "fake-tools"
        fake_tools.mkdir()
        for name in ("mise", "make"):
            tool = fake_tools / name
            tool.write_text("#!/bin/sh\nexit 0\n")
            tool.chmod(0o755)
        self.env = dict(os.environ, PATH=f"{fake_tools}:{os.environ['PATH']}")

    def hook(self):
        return subprocess.run(
            ["bash", str(ROOT / "scripts/precommit-check-generated.sh")],
            cwd=self.repo, env=self.env, text=True, capture_output=True,
        )

    def test_staged_generated_files_are_accepted(self):
        result = self.hook()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_new_generated_files_require_staging(self):
        (self.repo / "sdk/new_resource.py").write_text("new resource\n")
        result = self.hook()
        self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
        self.assertIn("Generated artifacts changed", result.stdout)

    def test_regenerated_changes_require_staging(self):
        (self.repo / "sdk/existing.py").write_text("regenerated\n")
        result = self.hook()
        self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
        self.assertIn("Generated artifacts changed", result.stdout)

    def test_unrelated_untracked_files_are_accepted(self):
        (self.repo / "notes.txt").write_text("local notes\n")
        result = self.hook()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


class CrossBuildTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.repo = Path(self.temp.name)
        self.env = dict(os.environ)
        (self.repo / "bin").mkdir()
        (self.repo / "scripts").mkdir()
        shutil.copy2(ROOT / "scripts/crossbuild.mk", self.repo / "scripts/crossbuild.mk")
        (self.repo / "Makefile").write_text(
            "PACK := logfire\nPROVIDER := pulumi-resource-logfire\n"
            f"WORKING_DIR := {self.repo}\n"
            "build_provider_cmd = mkdir -p $(dir $(3)) && printf 'binary' > $(3)\n"
            "include scripts/crossbuild.mk\n"
            ".make/schema:\n\t@mkdir -p .make && touch $@\n"
        )

    def make(self, target, *settings):
        return subprocess.run(
            ["make", "-s", target, "CI=true", *settings],
            cwd=self.repo, env=self.env, text=True, capture_output=True,
        )

    def signing_tools(self):
        tools = self.repo / "tools"
        tools.mkdir()
        self.env.update(PATH=f"{tools}:{os.environ['PATH']}",
                        TRACE_FILE=str(self.repo / "signing.txt"))
        scripts = {
            "az": """#!/bin/sh
printf '%s\n' "$1" >> "$TRACE_FILE"
case "$1" in
  login) exit "${LOGIN_STATUS:-0}" ;;
  account) printf '{"accessToken":"test-token"}\n' ;;
  logout) exit 0 ;;
esac
""",
            "jq": "#!/bin/sh\ncat >/dev/null\nprintf 'test-token\\n'\n",
            "java": """#!/bin/sh
printf 'java\n' >> "$TRACE_FILE"
if [ "${SIGN_STATUS:-0}" != 0 ]; then exit "$SIGN_STATUS"; fi
for binary do :; done
printf ' signed' >> "$binary"
""",
        }
        for name, script in scripts.items():
            tool = tools / name
            tool.write_text(script)
            tool.chmod(0o755)
        (self.repo / "bin/jsign-6.0.jar").touch()
        return ("SKIP_SIGNING=false", "AZURE_SIGNING_CLIENT_ID=test-client",
                "AZURE_SIGNING_CLIENT_SECRET=test-secret",
                "AZURE_SIGNING_TENANT_ID=test-tenant",
                "AZURE_SIGNING_KEY_VAULT_URI=https://example.invalid")

    def test_failed_azure_login_does_not_publish_binary(self):
        settings = self.signing_tools()
        self.env["LOGIN_STATUS"] = "1"
        result = self.make("provider-windows-amd64", *settings)
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual((self.repo / "signing.txt").read_text(), "login\n")
        self.assertFalse((self.repo / "bin/windows-amd64/pulumi-resource-logfire.exe").exists())

    def test_failed_signing_tool_is_retried(self):
        settings = self.signing_tools()
        self.env["SIGN_STATUS"] = "1"
        result = self.make("provider-windows-amd64", *settings)
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        binary = self.repo / "bin/windows-amd64/pulumi-resource-logfire.exe"
        self.assertFalse(binary.exists())
        self.assertEqual((self.repo / "signing.txt").read_text(), "login\naccount\njava\n")
        self.env["SIGN_STATUS"] = "0"
        result = self.make("provider-windows-amd64", *settings)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(binary.read_text(), "binary signed")
        self.assertEqual((self.repo / "signing.txt").read_text(),
                         "login\naccount\njava\nlogin\naccount\njava\nlogout\n")

    def test_successful_signing_publishes_only_completed_binary(self):
        settings = self.signing_tools()
        (self.repo / "README.md").write_text("readme\n")
        (self.repo / "LICENSE").write_text("license\n")
        result = self.make("provider-windows-amd64", *settings)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        result = self.make("provider_dist-windows-amd64", "PROVIDER_VERSION=0.1.0", *settings)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        with tarfile.open(self.repo / "bin/pulumi-resource-logfire-v0.1.0-windows-amd64.tar.gz") as archive:
            self.assertEqual(archive.extractfile("./pulumi-resource-logfire.exe").read(), b"binary signed")
            self.assertNotIn("./pulumi-resource-logfire.exe.unsigned", archive.getnames())
        self.assertEqual((self.repo / "signing.txt").read_text(), "login\naccount\njava\nlogout\n")

    def test_windows_build_and_package_can_skip_signing(self):
        (self.repo / "README.md").write_text("readme\n")
        (self.repo / "LICENSE").write_text("license\n")
        for target in ("provider-windows-amd64", "provider_dist-windows-amd64"):
            result = self.make(target, "PROVIDER_VERSION=0.1.0", "SKIP_SIGNING=true")
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        with tarfile.open(self.repo / "bin/pulumi-resource-logfire-v0.1.0-windows-amd64.tar.gz") as archive:
            self.assertEqual(archive.extractfile("./pulumi-resource-logfire.exe").read(), b"binary")
        self.assertFalse((self.repo / "bin/jsign-6.0.jar").exists())

    def test_partial_signing_configuration_fails(self):
        (self.repo / "bin/jsign-6.0.jar").touch()
        result = self.make("provider-windows-amd64", "SKIP_SIGNING=false",
                           "AZURE_SIGNING_CLIENT_ID=test-client", "AZURE_SIGNING_CLIENT_SECRET=",
                           "AZURE_SIGNING_TENANT_ID=", "AZURE_SIGNING_KEY_VAULT_URI=")
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("Can't sign windows binaries", result.stdout)
        self.assertFalse((self.repo / "bin/windows-amd64/pulumi-resource-logfire.exe").exists())

    def test_failed_windows_signing_is_retried(self):
        (self.repo / "bin/jsign-6.0.jar").touch()
        for attempt in range(2):
            with self.subTest(attempt=attempt):
                result = self.make(
                    "provider-windows-amd64", "SKIP_SIGNING=false",
                    "AZURE_SIGNING_CLIENT_ID=", "AZURE_SIGNING_CLIENT_SECRET=",
                    "AZURE_SIGNING_TENANT_ID=", "AZURE_SIGNING_KEY_VAULT_URI=",
                )
                self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                self.assertIn("Can't sign windows binaries", result.stdout)
                self.assertFalse((self.repo / "bin/windows-amd64/pulumi-resource-logfire.exe").exists())

    def test_linux_build_does_not_fetch_windows_signing_tool(self):
        result = self.make("provider-linux-amd64")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual((self.repo / "bin/linux-amd64/pulumi-resource-logfire").read_text(), "binary")
        self.assertFalse((self.repo / "bin/jsign-6.0.jar").exists())

    def test_windows_build_can_explicitly_skip_signing(self):
        result = self.make("provider-windows-amd64", "SKIP_SIGNING=true")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual((self.repo / "bin/windows-amd64/pulumi-resource-logfire.exe").read_text(), "binary")
        self.assertFalse((self.repo / "bin/jsign-6.0.jar").exists())

    def test_archive_refreshes_readme_without_recompiling(self):
        (self.repo / "README.md").write_text("original\n")
        (self.repo / "LICENSE").write_text("license\n")
        result = self.make("provider_dist-linux-amd64", "PROVIDER_VERSION=0.1.0")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        archive = self.repo / "bin/pulumi-resource-logfire-v0.1.0-linux-amd64.tar.gz"
        with tarfile.open(archive) as package:
            self.assertEqual(package.extractfile("README.md").read(), b"original\n")
        for path in self.repo.rglob("*"):
            if path.is_file():
                os.utime(path, (1_000_000_000, 1_000_000_000))
        (self.repo / "README.md").write_text("updated\n")
        os.utime(self.repo / "README.md", (1_000_000_010, 1_000_000_010))
        result = self.make("provider_dist-linux-amd64", "PROVIDER_VERSION=0.1.0")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        with tarfile.open(archive) as package:
            self.assertEqual(package.extractfile("README.md").read(), b"updated\n")
        self.assertEqual((self.repo / "bin/linux-amd64/pulumi-resource-logfire").stat().st_mtime,
                         1_000_000_000)


if __name__ == "__main__":
    unittest.main()
