package mtg

// These tests cover the catalog-independent MTG baseline: quantity rules,
// commander role/count rules, name normalization, and failure diagnostics.
//
// Reference: 20261117_01_tcg-decklist-graph-standard.md, lines 23-27,
// 51-65, and 244-264.

import (
	"testing"

	"cardhash/golang/engine"
	"cardhash/golang/standard"
)

// TestMTGConstructedValidation proves the built-in four-copy rule reports a
// violation while allowing basic structural validation to succeed first.
func TestMTGConstructedValidation(t *testing.T) {
	// Use repeated one-card items to prove validation aggregates quantities rather
	// than relying on source grouping or presentation format.
	items := make([]standard.CardItem, 0, 61)
	for i := 0; i < 60; i++ {
		items = append(items, standard.CardItem{Quantity: 1, CardRef: "Island"})
	}
	// One extra card makes the total valid but leaves Island above the copy limit.
	items = append(items, standard.CardItem{Quantity: 1, CardRef: "Lightning Bolt"})
	deck := standard.Deck{Lists: []standard.CardList{{Role: standard.RoleMain, Items: items}}}
	// Evaluate only the in-place constructed baseline; no catalog is consulted.
	diagnostics := ValidateMTG(deck, ConstructedRules())
	found := false
	// Search the complete diagnostic set because validators report multiple issues.
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == engine.DiagnosticCodeCopyLimit {
			found = true
		}
	}
	if !found {
		t.Fatal("expected constructed copy-limit diagnostic")
	}
}

// TestMTGCommanderValidation proves a 99-card main list plus one commander is
// accepted by the count-only Commander baseline.
func TestMTGCommanderValidation(t *testing.T) {
	// Basic lands are exempt from singleton counting in this catalog-free profile.
	deck := standard.Deck{Lists: []standard.CardList{
		{Role: standard.RoleMain, Items: []standard.CardItem{{Quantity: 99, CardRef: "Island"}}},
		{Role: RoleCommanders, Items: []standard.CardItem{{Quantity: 1, CardRef: "Alela, Artful Provocateur"}}},
	}}
	// A valid count profile should emit no error diagnostics.
	diagnostics := ValidateMTG(deck, CommanderRules())
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == engine.DiagnosticSeverityError {
			t.Fatalf("unexpected commander diagnostic: %#v", diagnostic)
		}
	}
}

// TestMTGNormalizationUsesNFCAndDelimiterRules pins the MTG CardRef spelling
// rules for split/faced names and presentation whitespace.
func TestMTGNormalizationUsesNFCAndDelimiterRules(t *testing.T) {
	// The two source forms differ only in delimiter spacing and case.
	profile := MTGProfile{}
	first, err := profile.NormalizeCardRef("Fire//Ice")
	if err != nil {
		t.Fatal(err)
	}
	second, err := profile.NormalizeCardRef(" fire // ice ")
	if err != nil {
		t.Fatal(err)
	}
	// Both must converge to the canonical lowercase readable delimiter form.
	if first != second || first != "fire // ice" {
		t.Fatalf("unexpected normalized refs: %q and %q", first, second)
	}
}

// TestMTGValidationReportsMissingCommander proves commander-role requirements
// are diagnostics rather than hash or parse failures.
func TestMTGValidationReportsMissingCommander(t *testing.T) {
	// The main list is large enough to make the missing-role diagnostic focused.
	deck := standard.Deck{Lists: []standard.CardList{{Role: standard.RoleMain, Items: []standard.CardItem{{Quantity: 99, CardRef: "Island"}}}}}
	// Validation must return a structured code that clients can act on.
	diagnostics := ValidateMTG(deck, CommanderRules())
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == CodeInvalidCommanderCount {
			return
		}
	}
	t.Fatal("expected missing commander diagnostic")
}
