package engine

// This file materializes the proposal's two parallel comparison scopes. A
// ChangeSet is an explanatory/result value, not embedded provenance: callers
// can compare any two representable decks without first asserting derivation.
//
// Reference: 20261117_01_tcg-decklist-graph-standard.md, "Deck Evolution and
// Relations" (lines 153-165 and 216-242).

import (
	"fmt"
	"sort"

	"cardhash/golang/v0/standard"
)

// Compare produces both required scope-specific changesets for two decks.
//
// The function validates representation but intentionally does not ask an MTG
// legality engine or external catalog to resolve either deck. This implements
// total comparison over the representable model and keeps provenance separate
// from similarity/diff computation.
// Reference: standard lines 155-163 and 240-242.
func Compare(source, target standard.Deck, profile standard.HashProfile) (structure, treatment standard.ChangeSet, err error) {
	// Validate each side independently so callers know which input is malformed.
	if err := source.Validate(profile); err != nil {
		return standard.ChangeSet{}, standard.ChangeSet{}, fmt.Errorf("source: %w", err)
	}
	if err := target.Validate(profile); err != nil {
		return standard.ChangeSet{}, standard.ChangeSet{}, fmt.Errorf("target: %w", err)
	}
	// Compute all four endpoint hashes once; the values are carried as references
	// on the corresponding scope-specific results.
	sourceStructure, sourceTreatment, err := DeckHashes(source, profile)
	if err != nil {
		return standard.ChangeSet{}, standard.ChangeSet{}, err
	}
	targetStructure, targetTreatment, err := DeckHashes(target, profile)
	if err != nil {
		return standard.ChangeSet{}, standard.ChangeSet{}, err
	}
	// Structural and treatment deltas are computed independently rather than
	// deriving one from the other, because treatment-only evolution is valid.
	structureItems, err := diff(source, target, profile, standard.StructureScope)
	if err != nil {
		return standard.ChangeSet{}, standard.ChangeSet{}, err
	}
	treatmentItems, err := diff(source, target, profile, standard.TreatmentScope)
	if err != nil {
		return standard.ChangeSet{}, standard.ChangeSet{}, err
	}
	structure = standard.ChangeSet{Scope: standard.StructureScope, SourceStructureHash: sourceStructure, TargetStructureHash: targetStructure, Items: structureItems}
	treatment = standard.ChangeSet{Scope: standard.TreatmentScope, SourceStructureHash: sourceStructure, TargetStructureHash: targetStructure, SourceTreatmentHash: sourceTreatment, TargetTreatmentHash: targetTreatment, Items: treatmentItems}
	return structure, treatment, nil
}

// Hash calculates the normalized changeset identity while excluding source/target references.
//
// Raw ordering, duplicate operations, and source/target hash references do not
// affect this identity. Malformed card references or treatment fields are
// rejected instead of silently disappearing from the hash input.
// Reference: standard lines 173-175.
func HashChangeSet(c standard.ChangeSet, profile standard.HashProfile) (string, error) {
	// Normalize at hash time so callers may construct denormalized changesets and
	// still receive the proposal's canonical semantic identity.
	if profile == nil {
		profile = GenericProfile{}
	}
	items, err := normalizeChangeItems(c.Items, profile, c.Scope)
	if err != nil {
		return "", err
	}
	domain := HashDomainStructureChangeSet
	if c.Scope == standard.TreatmentScope {
		domain = HashDomainTreatmentChangeSet
	}
	return hashJSON(domain, struct {
		Scope string                `json:"scope"`
		Items []standard.ChangeItem `json:"items"`
	}{string(c.Scope), items})
}

// diff computes the signed multiset difference for one comparison scope.
//
// The map representation makes the result independent of source presentation
// order. A positive target-minus-source quantity becomes an addition; a
// negative quantity retains the source bucket so treatment-removal identity is
// preserved.
// Reference: standard lines 169-175 and 193-214.
func diff(source, target standard.Deck, profile standard.HashProfile, scope standard.ChangeScope) ([]standard.ChangeItem, error) {
	// Flatten both decks into canonical role/reference/treatment buckets.
	left := flatten(source, profile, scope)
	right := flatten(target, profile, scope)
	keys := make(map[string]struct{}, len(left)+len(right))
	// Union the keys so additions and removals are both represented.
	for key := range left {
		keys[key] = struct{}{}
	}
	for key := range right {
		keys[key] = struct{}{}
	}
	items := make([]standard.ChangeItem, 0, len(keys))
	for key := range keys {
		// A zero net quantity is semantically unchanged and must not be serialized.
		quantity := right[key].Quantity - left[key].Quantity
		if quantity == 0 {
			continue
		}
		// Use the target bucket for additions and source bucket for removals so the
		// signed operation carries the correct treatment value.
		item := right[key]
		if quantity < 0 {
			item = left[key]
		}
		item.Quantity = quantity
		items = append(items, item)
	}
	return normalizeChangeItems(items, profile, scope)
}

// flatten converts a valid deck into the multiset used by a scope-specific diff.
//
// Structure scope deliberately sets treatment to nil. Treatment scope keeps
// profile-recognized fields and therefore distinguishes treated and untreated
// copies when required.
func flatten(deck standard.Deck, profile standard.HashProfile, scope standard.ChangeScope) map[string]standard.ChangeItem {
	result := make(map[string]standard.ChangeItem)
	for _, list := range deck.Lists {
		for _, sourceItem := range list.Items {
			// Inputs have already passed validation, so these profile operations are
			// expected to succeed; comparison remains catalog-independent.
			ref, _ := profile.NormalizeCardRef(sourceItem.CardRef)
			treatment, _, _ := profile.CanonicalTreatment(sourceItem.Treatment)
			if scope == standard.StructureScope {
				// Treatment is excluded entirely from structural identity.
				treatment = nil
			}
			role, roleErr := standard.NormalizeRole(profile, list.Role)
			if roleErr != nil {
				// Valid decks cannot reach this branch, but preserving the empty role
				// fallback keeps this internal helper total for future callers.
				role = standard.CanonicalRole(list.Role)
			}
			key := changeKey(role, ref, treatment)
			// Combine repeated equivalent source items before calculating the delta.
			item := result[key]
			item.Role, item.CardRef, item.Treatment = role, ref, treatment
			item.Quantity += sourceItem.Quantity
			result[key] = item
		}
	}
	return result
}

// normalizeChangeItems canonicalizes and combines signed operations.
//
// This is the changeset equivalent of list canonicalization: it removes zeros,
// canonicalizes identity, combines equal buckets, rejects malformed values,
// and sorts the result by role, CardRef, and treatment. It intentionally accepts
// denormalized input so callers can normalize external operation histories.
// Reference: standard lines 169-175.
func normalizeChangeItems(items []standard.ChangeItem, profile standard.HashProfile, scope standard.ChangeScope) ([]standard.ChangeItem, error) {
	combined := make(map[string]standard.ChangeItem)
	for _, item := range items {
		// Zero operations have no semantic effect and are prohibited in canonical
		// output, even if a permissive caller supplied one.
		if item.Quantity == 0 {
			continue
		}
		ref, err := profile.NormalizeCardRef(item.CardRef)
		if err != nil {
			return nil, err
		}
		treatment, _, err := profile.CanonicalTreatment(item.Treatment)
		if err != nil {
			return nil, err
		}
		if scope == standard.StructureScope {
			// Structural normalization intentionally discards treatment identity.
			treatment = nil
		}
		role, roleErr := standard.NormalizeRole(profile, item.Role)
		if roleErr != nil {
			return nil, roleErr
		}
		key := changeKey(role, ref, treatment)
		// Signed quantities combine algebraically; cancellation is removed below.
		combined[key] = standard.ChangeItem{Role: role, CardRef: ref, Treatment: treatment, Quantity: combined[key].Quantity + item.Quantity}
	}
	result := make([]standard.ChangeItem, 0, len(combined))
	for _, item := range combined {
		// A net-zero operation must disappear from the canonical changeset.
		if item.Quantity != 0 {
			result = append(result, item)
		}
	}
	// Deterministic ordering makes the changeset hash independent of map order.
	sort.Slice(result, func(i, j int) bool {
		if result[i].Role != result[j].Role {
			return result[i].Role < result[j].Role
		}
		if result[i].CardRef != result[j].CardRef {
			return result[i].CardRef < result[j].CardRef
		}
		return treatmentKey(result[i].Treatment) < treatmentKey(result[j].Treatment)
	})
	return result, nil
}

// changeKey creates an internal composite key for a role/reference/treatment
// bucket. NUL separators prevent ordinary field concatenation from creating
// accidental key collisions inside the in-memory normalization map.
func changeKey(role, ref string, treatment standard.Treatment) string {
	return role + "\x00" + ref + "\x00" + treatmentKey(treatment)
}
