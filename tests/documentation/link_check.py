#!/usr/bin/env python3
"""Reject broken repository-relative Markdown links."""

from __future__ import annotations

import re
import sys
from pathlib import Path

IGNORED = {".git", ".local", ".terraform", "node_modules", "tmp", "vendor"}
LINK = re.compile(r"(?<!!)\[[^]]*\]\(([^)]+)\)")
FENCED_BLOCK = re.compile(r"```.*?```", re.DOTALL)


def markdown_files(root: Path):
    for path in root.rglob("*.md"):
        if not any(part in IGNORED for part in path.parts):
            yield path


def target_path(source: Path, raw_target: str) -> Path | None:
    target = raw_target.strip().strip("<>").split("#", 1)[0]
    if not target or re.match(r"^[a-z][a-z0-9+.-]*:", target, re.IGNORECASE):
        return None
    return (source.parent / target).resolve()


def main() -> int:
    root = Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
    errors: list[str] = []
    for source in markdown_files(root):
        content = FENCED_BLOCK.sub("", source.read_text(encoding="utf-8"))
        for match in LINK.finditer(content):
            target = target_path(source, match.group(1))
            if target is not None and not target.exists():
                errors.append(f"{source.relative_to(root)}: missing link target {match.group(1)!r}")
    if errors:
        print("Markdown link validation failed:")
        print("\n".join(f"- {error}" for error in errors))
        return 1
    print("Markdown link validation passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
