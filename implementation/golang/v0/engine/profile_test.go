package engine

import (
	"testing"

	"cardhash/golang/v0/standard"
)

// TestConfigurableProfileSupportsNonMTGIdentity proves another game can define
// its own namespace and hash-significant treatment fields without an MTG type.
// Reference: 20261117_01_tcg-decklist-graph-standard.md, lines 51-59 and 87-93.
func TestConfigurableProfileSupportsNonMTGIdentity(t *testing.T) {
	// A Pokemon-like profile uses a game namespace and a small printing vocabulary.
	profile, err := NewConfigurableProfile("pokemon/tcg", "pokemon/treatment/1", []string{"set", "rarity"})
	if err != nil {
		t.Fatal(err)
	}
	first := standard.Deck{Lists: []standard.CardList{{Role: "MAIN", Items: []standard.CardItem{{Quantity: 1, CardRef: "Pikachu", Treatment: standard.Treatment{"set": "base", "rarity": "common"}}}}}}
	second := standard.Deck{Lists: []standard.CardList{{Role: " main ", Items: []standard.CardItem{{Quantity: 1, CardRef: "pikachu", Treatment: standard.Treatment{"set": "base", "rarity": "common", "catalogId": "external-only"}}}}}}
	// Catalog-only metadata must not alter profile-selected treatment identity.
	firstStructure, firstTreatment, err := DeckHashes(first, profile)
	if err != nil {
		t.Fatal(err)
	}
	secondStructure, secondTreatment, err := DeckHashes(second, profile)
	if err != nil {
		t.Fatal(err)
	}
	if firstStructure != secondStructure || firstTreatment != secondTreatment {
		t.Fatal("non-MTG profile did not preserve configured identity boundaries")
	}
}

// TestGenericRuleEngineWorksWithoutMTG proves rule handlers can validate a
// non-MTG deck using only the generic GameProfile contract.
func TestGenericRuleEngineWorksWithoutMTG(t *testing.T) {
	// This profile models a game with a 40-card main list and a four-copy limit.
	profile, err := NewConfigurableProfile("example/tcg", "example/treatment/1", []string{"set"})
	if err != nil {
		t.Fatal(err)
	}
	items := make([]standard.CardItem, 0, 10)
	for index := 0; index < 10; index++ {
		items = append(items, standard.CardItem{Quantity: 4, CardRef: "Card " + string(rune('A'+index))})
	}
	deck := standard.Deck{Lists: []standard.CardList{{Role: standard.RoleMain, Items: items}}}
	// Compose reusable handlers; no MTG validator or MTG-specific role is used.
	validator := NewValidator(profile,
		RequiredRoleRule{Role: standard.RoleMain},
		CardCountRule{Role: standard.RoleMain, Min: 40},
		CopyLimitRule{Roles: []string{standard.RoleMain}, Max: 4},
	)
	if diagnostics := validator.Validate(deck); len(diagnostics) != 0 {
		t.Fatalf("unexpected non-MTG diagnostics: %#v", diagnostics)
	}
}

// TestGenericRuleEngineReportsCopyLimit proves a reusable handler reports a
// non-happy-path violation with normalized CardRef context.
func TestGenericRuleEngineReportsCopyLimit(t *testing.T) {
	// The deck is structurally valid but exceeds the configured game rule.
	profile, err := NewConfigurableProfile("example/tcg", "example/treatment/1", nil)
	if err != nil {
		t.Fatal(err)
	}
	deck := standard.Deck{Lists: []standard.CardList{{Role: standard.RoleMain, Items: []standard.CardItem{{Quantity: 5, CardRef: "alpha"}}}}}
	validator := NewValidator(profile, CopyLimitRule{Roles: []string{standard.RoleMain}, Max: 4, Code: "example-copy-limit"})
	diagnostics := validator.Validate(deck)
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "example-copy-limit" && diagnostic.CardRef == "alpha" {
			return
		}
	}
	t.Fatal("expected generic copy-limit diagnostic")
}
