// Package engine implements reusable algorithms over the shared standard
// ontology. It owns hashing, changeset comparison, changeset replay, generic
// profiles, and composable rule handlers, but it MUST NOT encode MTG-specific
// roles, card names, or legality assumptions.
//
// Reference: 20261117_01_tcg-decklist-graph-standard.md, sections Decklist
// Identity and Hashing, Changesets and Diffs, and Compatibility and Versioning
// (lines 250-272).
package engine
