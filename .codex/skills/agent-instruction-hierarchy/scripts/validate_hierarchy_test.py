#!/usr/bin/env python3
from pathlib import Path
from tempfile import TemporaryDirectory
import unittest

import validate_hierarchy

SECTIONS = """# Instructions
## Purpose
x
## Scope
x
## Local rules
x
## Usage
x
## Validation
x
## Elements
| Element | Behavior |
| --- | --- |
{elements}
## Instruction hierarchy
{hierarchy}
"""


class HierarchyValidatorTest(unittest.TestCase):
    def test_rejects_missing_file_and_required_section(self):
        with TemporaryDirectory() as raw:
            root = Path(raw)
            self.assertIn("missing AGENTS.md", validate_hierarchy.validate_directory(root, root, {root})[0])
            (root / "AGENTS.md").write_text("Parent: none", encoding="utf-8")
            self.assertTrue(any("missing required section" in error for error in validate_hierarchy.validate_directory(root, root, {root})))

    def test_rejects_malformed_element_row(self):
        with TemporaryDirectory() as raw:
            root = Path(raw)
            (root / "owned.txt").write_text("x", encoding="utf-8")
            (root / "AGENTS.md").write_text(SECTIONS.format(elements="`owned.txt`", hierarchy="- Parent: none."), encoding="utf-8")
            self.assertTrue(any("Elements table" in error for error in validate_hierarchy.validate_directory(root, root, {root})))

    def test_accepts_reciprocal_links_and_rejects_broken_child_link(self):
        with TemporaryDirectory() as raw:
            root = Path(raw)
            child = root / "child"
            child.mkdir()
            (root / "AGENTS.md").write_text(SECTIONS.format(elements="| `child` | Child rules. |", hierarchy="- Parent: none.\n- Child: [child](child/AGENTS.md)."), encoding="utf-8")
            (child / "AGENTS.md").write_text(SECTIONS.format(elements="", hierarchy="- Parent: [root](../AGENTS.md).\n- Children: none."), encoding="utf-8")
            directories = {root, child}
            self.assertEqual(validate_hierarchy.validate_directory(root, root, directories), [])
            self.assertEqual(validate_hierarchy.validate_directory(root, child, directories), [])
            (root / "AGENTS.md").write_text(SECTIONS.format(elements="| `child` | Child rules. |", hierarchy="- Parent: none.\n- Child: [child](missing/AGENTS.md)."), encoding="utf-8")
            self.assertTrue(any("missing child link" in error for error in validate_hierarchy.validate_directory(root, root, directories)))


if __name__ == "__main__":
    unittest.main()
