package engine

// These tests exercise comparison and replay semantics from the proposal's
// parallel StructureChangeSet/TreatmentChangeSet model.
//
// Reference: 20261117_01_tcg-decklist-graph-standard.md, lines 153-175 and
// 193-218.

import (
	"testing"

	"cardhash/golang/standard"
)

// TestCompareProducesParallelScopes proves that treatment-only evolution emits
// an empty structural result and a non-empty treatment result for one pair.
func TestCompareProducesParallelScopes(t *testing.T) {
	// All source copies have one treatment; the target redistributes two copies
	// and leaves one printing unspecified without changing total CardRef quantity.
	profile, err := NewConfigurableProfile("test/tcg", "test/treatment/1", []string{"set"})
	if err != nil {
		t.Fatal(err)
	}
	source := standard.Deck{Lists: []standard.CardList{{Role: standard.RoleMain, Items: []standard.CardItem{{Quantity: 3, CardRef: "Lightning Bolt", Treatment: standard.Treatment{"set": "1ED"}}}}}}
	target := standard.Deck{Lists: []standard.CardList{{Role: standard.RoleMain, Items: []standard.CardItem{{Quantity: 1, CardRef: "Lightning Bolt", Treatment: standard.Treatment{"set": "1ED"}}, {Quantity: 1, CardRef: "Lightning Bolt", Treatment: standard.Treatment{"set": "2ED"}}, {Quantity: 1, CardRef: "Lightning Bolt"}}}}}
	// Compare must materialize both scopes even when one has no changes.
	structure, treatment, err := Compare(source, target, profile)
	if err != nil {
		t.Fatal(err)
	}
	// Structural identity sees three Lightning Bolts on both sides.
	if len(structure.Items) != 0 {
		t.Fatalf("expected empty structural diff, got %#v", structure.Items)
	}
	// Treatment identity sees removal of two 1ED copies, one untreated addition,
	// and one 2ED addition.
	if len(treatment.Items) != 3 {
		t.Fatalf("expected three treatment changes, got %#v", treatment.Items)
	}
	// The scope labels are part of the result contract, not inferred by callers.
	if structure.Scope != standard.StructureScope || treatment.Scope != standard.TreatmentScope {
		t.Fatal("comparison scopes were not declared")
	}
}

// TestApplyTreatmentChangesRejectsOverRemoval protects replay safety when a
// negative operation exceeds the source bucket quantity.
func TestApplyTreatmentChangesRejectsOverRemoval(t *testing.T) {
	// The source contains one untreated copy but the operation removes two.
	source := standard.Deck{Lists: []standard.CardList{{Role: standard.RoleMain, Items: []standard.CardItem{{Quantity: 1, CardRef: "Lightning Bolt"}}}}}
	changes := standard.ChangeSet{Scope: standard.TreatmentScope, Items: []standard.ChangeItem{{Role: standard.RoleMain, Quantity: -2, CardRef: "Lightning Bolt"}}}
	// Application must reject underflow instead of producing a negative item.
	if _, err := ApplyChangeSet(source, changes, GenericProfile{}, ClearTreatment); err == nil {
		t.Fatal("expected over-removal to fail")
	}
}

// TestApplyTreatmentChangesReachesExpectedTarget verifies that exact treatment
// buckets can replay the example transition from the proposal.
func TestApplyTreatmentChangesReachesExpectedTarget(t *testing.T) {
	// The source starts with three specifically treated copies.
	profile, err := NewConfigurableProfile("test/tcg", "test/treatment/1", []string{"set"})
	if err != nil {
		t.Fatal(err)
	}
	source := standard.Deck{Lists: []standard.CardList{{Role: standard.RoleMain, Items: []standard.CardItem{{Quantity: 3, CardRef: "Lightning Bolt", Treatment: standard.Treatment{"set": "1ED"}}}}}}
	changes := standard.ChangeSet{Scope: standard.TreatmentScope, Items: []standard.ChangeItem{{Role: standard.RoleMain, Quantity: -2, CardRef: "Lightning Bolt", Treatment: standard.Treatment{"set": "1ED"}}, {Role: standard.RoleMain, Quantity: 1, CardRef: "Lightning Bolt"}, {Role: standard.RoleMain, Quantity: 1, CardRef: "Lightning Bolt", Treatment: standard.Treatment{"set": "2ED"}}}}
	// Apply the signed treatment operations, including an untreated addition.
	target, err := ApplyChangeSet(source, changes, profile, ClearTreatment)
	if err != nil {
		t.Fatal(err)
	}
	// Compare resulting treatment identity rather than presentation order.
	_, targetTreatment, err := DeckHashes(target, profile)
	if err != nil {
		t.Fatal(err)
	}
	// Construct the expected semantic target independently from the application.
	wanted := standard.Deck{Lists: []standard.CardList{{Role: standard.RoleMain, Items: []standard.CardItem{{Quantity: 1, CardRef: "Lightning Bolt", Treatment: standard.Treatment{"set": "1ED"}}, {Quantity: 1, CardRef: "Lightning Bolt", Treatment: standard.Treatment{"set": "2ED"}}, {Quantity: 1, CardRef: "Lightning Bolt"}}}}}
	_, wantedTreatment, err := DeckHashes(wanted, profile)
	if err != nil {
		t.Fatal(err)
	}
	if targetTreatment != wantedTreatment {
		t.Fatalf("applied hash %s differs from wanted %s", targetTreatment, wantedTreatment)
	}
}
