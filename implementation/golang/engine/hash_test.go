package engine

// These tests pin the identity boundaries described by the proposal: display
// metadata and source presentation are non-identity, while recognized treatment
// changes affect only treatment scope.
//
// Reference: 20261117_01_tcg-decklist-graph-standard.md, lines 29-31,
// 118-149, and 274-280.

import (
	"testing"

	"cardhash/golang/standard"
)

// TestHashesIgnorePresentationOrderAndMetadata proves that canonical identity
// survives card reordering, whitespace/case normalization, and deck-name changes.
func TestHashesIgnorePresentationOrderAndMetadata(t *testing.T) {
	// The first deck uses ordinary readable source formatting.
	profile, err := NewConfigurableProfile("test/tcg", "test/treatment/1", []string{"set"})
	if err != nil {
		t.Fatal(err)
	}
	first := standard.Deck{Lists: []standard.CardList{{Role: standard.RoleMain, Items: []standard.CardItem{{Quantity: 1, CardRef: "Lightning Bolt"}, {Quantity: 2, CardRef: "Giant Growth"}}}}}
	second := standard.Deck{Name: "different", Lists: []standard.CardList{{Role: standard.RoleMain, Items: []standard.CardItem{{Quantity: 2, CardRef: "giant   growth"}, {Quantity: 1, CardRef: " LIGHTNING BOLT "}}}}}
	// Hash the baseline form before changing only non-identity representation.
	firstStructure, firstTreatment, err := DeckHashes(first, profile)
	if err != nil {
		t.Fatal(err)
	}
	// Hash the reordered, normalized, differently named form.
	secondStructure, secondTreatment, err := DeckHashes(second, profile)
	if err != nil {
		t.Fatal(err)
	}
	// Both scopes must remain equal because none of the changed fields are identity.
	if firstStructure != secondStructure || firstTreatment != secondTreatment {
		t.Fatalf("equivalent decks differ: %s/%s vs %s/%s", firstStructure, firstTreatment, secondStructure, secondTreatment)
	}
}

// TestTreatmentChangesTreatmentHashOnly proves that treatment is additive to
// structural identity rather than replacing CardRef/quantity identity.
func TestTreatmentChangesTreatmentHashOnly(t *testing.T) {
	// The structural list leaves printing unspecified.
	profile, err := NewConfigurableProfile("test/tcg", "test/treatment/1", []string{"set"})
	if err != nil {
		t.Fatal(err)
	}
	structural := standard.CardList{Role: standard.RoleMain, Items: []standard.CardItem{{Quantity: 3, CardRef: "Lightning Bolt"}}}
	treated := standard.CardList{Role: standard.RoleMain, Items: []standard.CardItem{{Quantity: 2, CardRef: "Lightning Bolt", Treatment: standard.Treatment{"set": "1ED"}}, {Quantity: 1, CardRef: "Lightning Bolt", Treatment: standard.Treatment{"set": "2ED"}}}}
	// Establish the untreated baseline hashes.
	firstStructure, firstTreatment, err := ListHashes(structural, profile)
	if err != nil {
		t.Fatal(err)
	}
	// Split the same quantity across two recognized set treatments.
	secondStructure, secondTreatment, err := ListHashes(treated, profile)
	if err != nil {
		t.Fatal(err)
	}
	// Card composition is unchanged even though treatment composition differs.
	if firstStructure != secondStructure {
		t.Fatal("treatment changed structural hash")
	}
	// Treatment identity must expose the printing distinction.
	if firstTreatment == secondTreatment {
		t.Fatal("treatment did not change treatment hash")
	}
}

// TestInvalidZeroQuantityIsRejected enforces the canonical positive-quantity
// rule instead of allowing zero entries to become hash-visible noise.
func TestInvalidZeroQuantityIsRejected(t *testing.T) {
	// Zero is invalid at the CardItem boundary and must fail before hashing.
	deck := standard.Deck{Lists: []standard.CardList{{Role: standard.RoleMain, Items: []standard.CardItem{{Quantity: 0, CardRef: "Lightning Bolt"}}}}}
	if _, _, err := DeckHashes(deck, GenericProfile{}); err == nil {
		t.Fatal("expected zero quantity to be rejected")
	}
}

// TestRoleNormalizationIsIdentityBearingButPresentationIndependent verifies
// that role names participate in deck identity after canonical role normalization.
func TestRoleNormalizationIsIdentityBearingButPresentationIndependent(t *testing.T) {
	// These role spellings are presentation variants of the same semantic role.
	profile := GenericProfile{}
	first := standard.Deck{Lists: []standard.CardList{{Role: "MAIN", Items: []standard.CardItem{{Quantity: 1, CardRef: "Island"}}}}}
	second := standard.Deck{Lists: []standard.CardList{{Role: " main ", Items: []standard.CardItem{{Quantity: 1, CardRef: "Island"}}}}}
	// Hash both variants through the public method API.
	firstStructure, _, err := DeckHashes(first, profile)
	if err != nil {
		t.Fatal(err)
	}
	secondStructure, _, err := DeckHashes(second, profile)
	if err != nil {
		t.Fatal(err)
	}
	// Canonical role handling must prevent cosmetic spelling from changing identity.
	if firstStructure != secondStructure {
		t.Fatal("role spelling changed structural identity")
	}
}
