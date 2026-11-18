# TCG Decklist Graph v0 Baseline

This module is the v0 Go reference implementation of the decklist proposal. It is intended to make the model executable and provide future design guidance, not to claim that unresolved standard questions are settled. Future implementation iterations MUST use a separate namespace rather than silently changing v0 behavior.

The v0 implementation is intentionally divided into three namespaces under `v0`: `v0/standard` contains shared ontology and contracts, `v0/engine` contains reusable Go algorithms and validation machinery, and `v0/mtg` contains Magic-specific profile and rule composition. Dependencies flow from game-specific code toward generic engine code and from engine code toward standard contracts.

## Included

- Explicit `Deck`, `CardList`, `CardItem`, `CardMeta`, `Treatment`, and `ChangeSet` values.
- `v0/standard` ontology and interfaces separated from implementation algorithms.
- `v0/engine` hashing, changeset, replay, configurable-profile, and rule-handler implementations.
- `v0/mtg` profile, treatment vocabulary, format presets, and MTG rule composition.
- Profile-driven card-reference and treatment canonicalization.
- `GameProfile` and `RoleProfile` interfaces for non-MTG identity namespaces.
- Composable `RuleHandler` validation with reusable role-count, total-count, and copy-limit handlers.
- Deterministic v0 structural and treatment hashes using SHA-256 over Go-canonical JSON inputs.
- Parallel v0 structural and treatment comparisons with normalized signed changes.
- A provisional Deck-first comparison API; the settled standard direction is profile-scoped `CardList` comparison composed by role.
- Treatment-inclusive changeset application and structural changeset application with clear/preserve policies.
- A catalog-independent MTG profile using NFC name normalization and baseline constructed and Commander checks.
- `ConfigurableProfile` and generic rule-engine examples demonstrating non-MTG use.
- Validation diagnostics that remain outside content identity.

## Run

```sh
go test ./...
go vet ./...
```

## Deliberate Baseline Limits

- The MTG validator does not resolve Oracle data, set legality, color identity, banned lists, card types, or commander eligibility.
- Role-level operations, Deck header operations, merge provenance, signatures, transport schemas, coordinate registries, and parameterized profile hashing remain future work.
- All CardLists are treated as unordered by the standard direction; v0 retains source-independent canonical sorting behavior.
- Hash strings are lowercase hexadecimal SHA-256 values; v0 does not implement RFC 8785 JCS or the settled hash-agility envelope.
