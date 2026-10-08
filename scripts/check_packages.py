#!/usr/bin/env python3
"""Validate generated release packages without publishing them."""

import argparse
import email
import json
import os
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile
import tomllib
import zipfile


def verify(actual, expected, source):
    if actual != expected:
        raise SystemExit(f"{source}: expected {expected}, got {actual}")
    print(f"{source}: {actual[0]} {actual[1]}", flush=True)


def distribution_metadata(path):
    if path.suffix == ".whl":
        with zipfile.ZipFile(path) as archive:
            name = next(n for n in archive.namelist() if n.endswith(".dist-info/METADATA"))
            metadata = archive.read(name)
    else:
        with tarfile.open(path) as archive:
            member = next(m for m in archive.getmembers() if m.name.endswith("/PKG-INFO"))
            metadata = archive.extractfile(member).read()
    message = email.message_from_bytes(metadata)
    return message["Name"], message["Version"]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--allow-local-version", action="store_true")
    parser.add_argument("version")
    parser.add_argument("node_archive", type=Path)
    parser.add_argument("python_archive", type=Path)
    args = parser.parse_args()
    if "+" in args.version and not args.allow_local_version:
        parser.error("local versions are for branch checks and cannot be published to PyPI")

    env = os.environ.copy()
    for name in ("NODE_AUTH_TOKEN", "NPM_TOKEN", "PYPI_USERNAME", "PYPI_PASSWORD"):
        env.pop(name, None)
    env.update(NPM_CONFIG_USERCONFIG="/dev/null", NPM_CONFIG_REGISTRY="https://registry.npmjs.org")

    def version_for(language):
        return subprocess.check_output(
            ["pulumictl", "convert-version", "--version", args.version, "--language", language],
            env=env, text=True,
        ).strip()

    node_version = version_for("javascript").removeprefix("v")
    python_version = version_for("python")
    if "+" in args.version:
        python_version += "+" + args.version.split("+", 1)[1]

    with tempfile.TemporaryDirectory() as temp:
        root = Path(temp)
        for language, archive in (("nodejs", args.node_archive), ("python", args.python_archive)):
            destination = root / language
            destination.mkdir()
            subprocess.run(["tar", "-zxf", str(archive.resolve()), "-C", str(destination)], check=True)

        node_bin = root / "nodejs/bin"
        node = json.loads((node_bin / "package.json").read_text())
        verify((node["name"], node["version"]), ("@pydantic/pulumi-logfire", node_version), "npm package")

        python_bin = root / "python/bin"
        project = tomllib.loads((python_bin / "pyproject.toml").read_text())["project"]
        expected = ("pydantic-pulumi-logfire", python_version)
        verify((project["name"], project["version"]), expected, "Python project")
        distributions = [p for p in (python_bin / "dist").iterdir() if not p.name.startswith(".")]
        wheels = [p for p in distributions if p.suffix == ".whl"]
        sdists = [p for p in distributions if p.name.endswith(".tar.gz")]
        if len(distributions) != 2 or len(wheels) != 1 or len(sdists) != 1:
            raise SystemExit("expected exactly one wheel and one source distribution")
        for path in wheels + sdists:
            verify(distribution_metadata(path), expected, path.name)

        subprocess.run(["npm", "pack", "--dry-run", "--json"], cwd=node_bin, env=env, check=True)
        subprocess.run([sys.executable, "-m", "venv", str(root / "twine")], env=env, check=True)
        python = str(root / "twine/bin/python")
        subprocess.run(
            [python, "-m", "pip", "install", "--quiet", "--disable-pip-version-check", "twine"],
            env=env, check=True,
        )
        subprocess.run([python, "-m", "twine", "check", *map(str, distributions)], env=env, check=True)


if __name__ == "__main__":
    main()
