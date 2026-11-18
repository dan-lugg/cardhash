# TCG Decklist Graph Baseline

This module is a deliberately small Go reference implementation of the current decklist proposal. It is intended to make the model executable and provide future design guidance, not to claim that unresolved standard questions are settled.

The module is intentionally divided into three namespaces: `standard` contains shared ontology and contracts, `engine` contains reusable Go algorithms and validation machinery, and `mtg` contains Magic-specific profile and rule composition. Dependencies flow from game-specific code toward generic engine code and from engine code toward standard contracts.

## Included

- Explicit `Deck`, `CardList`, `CardItem`, `CardMeta`, `Treatment`, and `ChangeSet` values.
- `standard` ontology and interfaces separated from implementation algorithms.
- `engine` hashing, changeset, replay, configurable-profile, and rule-handler implementations.
- `mtg` profile, treatment vocabulary, format presets, and MTG rule composition.
- Profile-driven card-reference and treatment canonicalization.
- `GameProfile` and `RoleProfile` interfaces for non-MTG identity namespaces.
- Composable `RuleHandler` validation with reusable role-count, total-count, and copy-limit handlers.
- Deterministic structural and treatment hashes using SHA-256 over canonical JSON inputs.
- Parallel structural and treatment comparisons with normalized signed changes.
- Deck comparison currently flattens role-scoped card changes; a first-class `CardList` comparison contract and canonical nested-versus-flattened deck result shape remain provisional.
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
- Role-level operations, ordered-list operations, merge provenance, signatures, transport schemas, and a profile registry remain future work.
- Unknown treatment fields are preserved by callers but excluded from the MTG treatment hash, matching the current proposal's provisional policy.
- Hash strings are lowercase hexadecimal SHA-256 values; the wire format and hash-agility policy are not standardized here.
