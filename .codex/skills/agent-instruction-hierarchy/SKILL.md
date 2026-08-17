---
name: agent-instruction-hierarchy
description: "Create, update, or validate linked AGENTS.md instructions across this repository. Use when adding or changing a repository-owned directory, creating AGENTS.md files, documenting directory behavior, or checking instruction coverage: require local rules, direct-element behavior, and accurate parent/child AGENTS.md references."
---

# Agent Instruction Hierarchy

Every repository-owned directory must contain `AGENTS.md`. Exclude only Git internals, dependency/vendor directories, generated caches, and test/runtime artifacts.

## Required file structure

Each `AGENTS.md` must include these sections:

```markdown
# <Directory Name>

## Purpose

## Scope

## Local rules

## Usage

## Validation

## Elements

| Element | Behavior |
| --- | --- |
| `child-or-file` | Concise behavioral responsibility. |

## Instruction hierarchy

- Parent: [AGENTS.md](../AGENTS.md)
- Children: [child AGENTS.md](child/AGENTS.md)
```

List every direct repository-owned child other than `AGENTS.md` in `## Elements`. Describe what it does, not merely its file type.

## Linking rules

- Root `AGENTS.md` must state `Parent: none`.
- Every non-root directory must link to its nearest parent `AGENTS.md`.
- Every directory with authored child directories must link to each child `AGENTS.md`.
- Keep links relative and reciprocal: a parent lists each child, and the child links back to the parent.
- Update the current directory, its parent, and affected child instructions in the same change whenever adding, moving, or removing a directory.

## Workflow

1. Read the nearest parent instructions before changing a directory.
2. Create or update the current directory's `AGENTS.md` before adding its implementation files.
3. Describe each new direct element and add its own `AGENTS.md` if it is a directory.
4. Update hierarchy links in both directions.
5. Run `scripts/validate_hierarchy.py --root .`.
6. Manually confirm that each element description matches current behavior; record the reviewer and date in the plan when the directory is complete.

Do not use an automated pass as proof that behavior descriptions are correct; it validates coverage and links only.
