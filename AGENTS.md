# Project Guidance

## Purpose

This repository defines a language-independent standard for trading card game decklist graphs, with Magic: The Gathering as the primary use case and extensibility for other games.

## Documentation

- Proposal documents MUST use RFC 2119 and RFC 8174 keywords for normative requirements.
- Documentation SHOULD prefer short, composable, human-readable type names.
- Relationship, hierarchy, flow, and lifecycle diagrams SHOULD use Mermaid fenced code blocks instead of ASCII diagrams whenever Mermaid can express the concept clearly.
- Mermaid diagrams MUST use stable, descriptive node identifiers and SHOULD keep display labels aligned with the terminology defined in the surrounding text.
- Diagrams SHOULD remain language-agnostic unless a section explicitly describes a game-specific profile.
- Markdown documents MUST use the repository's date-prefixed naming convention when they are newly created, except for repository convention files such as `README.md` and `AGENTS.md`.

## Standard Evolution

- Provisional terminology MUST be identified as provisional until adopted by the proposal.
- New terms SHOULD be evaluated for readability, composability, and consistent naming length before they are introduced.
- Magic: The Gathering examples SHOULD remain illustrative and MUST NOT silently become requirements for other game profiles.
- Open design questions SHOULD be recorded in the relevant proposal document for the next review cycle.
