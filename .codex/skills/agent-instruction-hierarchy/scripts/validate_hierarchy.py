#!/usr/bin/env python3
"""Validate repository-owned AGENTS.md coverage and instruction links."""

from __future__ import annotations

import argparse
import os
import re
from pathlib import Path


IGNORED_DIRECTORIES = {
    ".git",
    ".terraform",
    ".cache",
    ".local",
    ".pytest_cache",
    "__pycache__",
    "coverage",
    "node_modules",
    "tmp",
    "vendor",
}
REQUIRED_SECTIONS = (
    "## Purpose",
    "## Scope",
    "## Local rules",
    "## Usage",
    "## Validation",
    "## Elements",
    "## Instruction hierarchy",
)


def authored_directories(root: Path) -> list[Path]:
    result: list[Path] = []
    for current, directories, _ in os.walk(root):
        directories[:] = sorted(
            directory for directory in directories if directory not in IGNORED_DIRECTORIES
        )
        result.append(Path(current))
    return result


def relative_link(source: Path, target: Path) -> str:
    return os.path.relpath(target, source.parent).replace(os.sep, "/")


def validate_directory(root: Path, directory: Path, all_directories: set[Path]) -> list[str]:
    errors: list[str] = []
    instruction_file = directory / "AGENTS.md"
    display = directory.relative_to(root) if directory != root else Path(".")

    if not instruction_file.is_file():
        return [f"{display}: missing AGENTS.md"]

    content = instruction_file.read_text(encoding="utf-8")
    for section in REQUIRED_SECTIONS:
        if section not in content:
            errors.append(f"{display}: missing required section {section!r}")

    child_entries = sorted(path.name for path in directory.iterdir() if path.name != "AGENTS.md")
    for entry in child_entries:
        if entry in IGNORED_DIRECTORIES:
            continue
        row = re.compile(rf"^\|\s*`{re.escape(entry)}`\s*\|\s*[^|\s][^|]*\|\s*$", re.MULTILINE)
        if not row.search(content):
            errors.append(f"{display}: Elements table does not describe {entry!r}")

    parent = directory.parent
    if directory == root:
        if "Parent: none" not in content:
            errors.append(f"{display}: root must state 'Parent: none'")
    elif parent in all_directories:
        expected_parent_link = relative_link(instruction_file, parent / "AGENTS.md")
        if expected_parent_link not in content:
            errors.append(f"{display}: missing parent link {expected_parent_link!r}")

    for child in sorted(path for path in directory.iterdir() if path.is_dir()):
        if child.name in IGNORED_DIRECTORIES or child not in all_directories:
            continue
        expected_child_link = relative_link(instruction_file, child / "AGENTS.md")
        if expected_child_link not in content:
            errors.append(f"{display}: missing child link {expected_child_link!r}")

    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", default=".", help="repository root to validate")
    arguments = parser.parse_args()

    root = Path(arguments.root).resolve()
    directories = authored_directories(root)
    directory_set = set(directories)
    errors = [
        error
        for directory in directories
        for error in validate_directory(root, directory, directory_set)
    ]

    if errors:
        print("AGENTS.md hierarchy validation failed:")
        print("\n".join(f"- {error}" for error in errors))
        return 1

    print(f"AGENTS.md hierarchy validation passed for {len(directories)} directories.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
