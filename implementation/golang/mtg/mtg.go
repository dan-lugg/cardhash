package mtg

// This file is the baseline MTG game profile and validator. It intentionally
// implements only rules that can be evaluated from deck contents alone; it does
// not fetch Oracle data, set legality, color identity, bans, or card type data.
// That keeps legality an external profile evaluation while still giving the
// reference implementation useful in-place validation behavior.
//
// Reference: 20261117_01_tcg-decklist-graph-standard.md, "CardRef", "CardRef
// Naming and Normalization", "CardMeta", and legality separation (lines 51-77
// and 23-27).

import (
	"fmt"
	"strings"

	"cardhash/golang/engine"
	"cardhash/golang/standard"
	"golang.org/x/text/unicode/norm"
)

// MTGProfile implements the baseline Magic card-name and treatment rules.
//
// The profile treats canonical English names as identity, applies NFC and
// case-insensitive comparison, and recognizes a compact set of printing fields
// for treatment hashing. Catalog IDs remain optional metadata by design.
// Reference: standard lines 53-59, 63-65, and 69-75.
type MTGProfile struct{}

// GameID identifies this profile as the Magic: The Gathering namespace.
func (MTGProfile) GameID() string { return GameID }

// NormalizeRole applies the shared role policy while allowing MTG to satisfy
// the generic GameProfile contract.
func (MTGProfile) NormalizeRole(value string) (string, error) {
	role := standard.CanonicalRole(value)
	if role == "" {
		return "", fmt.Errorf("role must not be empty")
	}
	return role, nil
}

// NormalizeCardRef applies NFC, whitespace, delimiter, and case normalization.
//
// NFC is applied before whitespace and delimiter handling, matching the stated
// MTG normalization sequence. The result is lower case because the proposal
// requires canonical lowercase comparison/hash form while allowing readable
// display spelling to be retained elsewhere.
// Reference: standard lines 57 and 61-65.
func (MTGProfile) NormalizeCardRef(value string) (string, error) {
	// Normalize canonically equivalent Unicode sequences before any identity work.
	value = norm.NFC.String(strings.TrimSpace(value))
	// Collapse source formatting whitespace into one identity separator.
	value = strings.Join(strings.Fields(value), " ")
	// Normalize the delimiter before restoring exactly one readable separator.
	value = strings.ReplaceAll(value, " // ", "//")
	value = strings.ReplaceAll(value, "//", " // ")
	value = strings.Join(strings.Fields(value), " ")
	// An empty name cannot identify a card, even when catalog lookup is unavailable.
	if value == "" {
		return "", fmt.Errorf("card reference must not be empty")
	}
	return strings.ToLower(value), nil
}

// CanonicalTreatment recognizes the baseline MTG printing fields and ignores extensions.
//
// Unknown keys are deliberately excluded from the treatment hash but remain
// available to the caller's original Treatment map for round-tripping. This is
// the proposal's provisional forward-compatibility policy; it must not be read
// as proof that unknown data is semantically irrelevant.
// Reference: standard lines 81-93 and adversarial review "Treatment Semantics".
func (MTGProfile) CanonicalTreatment(value standard.Treatment) (map[string]string, bool, error) {
	// Nil means no treatment was supplied and therefore cannot activate treatment
	// scope on its own.
	if value == nil {
		return nil, false, nil
	}
	// This list is the versioned baseline treatment vocabulary. It intentionally
	// excludes Oracle IDs because the proposal makes them CardMeta metadata.
	known := map[string]struct{}{
		TreatmentSet: {}, TreatmentCollectorNumber: {}, TreatmentLanguage: {}, TreatmentFinish: {}, TreatmentFoil: {},
		TreatmentBorder: {}, TreatmentFrame: {}, TreatmentEdition: {}, TreatmentArtwork: {}, TreatmentReskin: {},
	}
	result := make(map[string]string)
	for key, field := range value {
		// Reject malformed field names even when the field would otherwise be
		// unknown, so malformed treatment cannot silently cross a profile boundary.
		if strings.TrimSpace(key) == "" {
			return nil, false, fmt.Errorf("treatment field name must not be empty")
		}
		if _, exists := known[key]; !exists {
			// Preserve unknown fields outside the hash layer by simply not projecting
			// them into the canonical treatment map.
			continue
		}
		// Lowercase textual values for stable comparison in this baseline profile.
		result[key] = strings.ToLower(strings.TrimSpace(field))
	}
	return result, len(result) > 0, nil
}

// TreatmentProfileID identifies the baseline MTG treatment rules.
// The immutable namespaced ID is included in treatment hashes when recognized
// treatment data exists, preventing silent profile-version collisions.
// Reference: standard line 91.
func (MTGProfile) TreatmentProfileID() string { return TreatmentProfileID }

// MTGFormat identifies the built-in validation rule set.
//
// These format identifiers select baseline validators; they are not embedded in
// content hashes and therefore cannot change deck identity when rule profiles
// evolve.
type MTGFormat string

const (
	// MTGConstructed uses a sixty-card main deck and four-copy limit.
	MTGConstructed MTGFormat = FormatConstructed
	// MTGCommander uses a 100-card singleton deck with a commander role.
	MTGCommander MTGFormat = FormatCommander
)

// MTGRules describes a deliberately small, catalog-independent legality profile.
//
// Fields express only quantity, role, and count constraints available from the
// deck model. A future richer MTG profile can add catalog-backed diagnostics
// without changing the core Deck or hash contracts.
// Reference: standard lines 23-27 and 258-264.
type MTGRules struct {
	Format            MTGFormat
	MainMinimum       int
	MainMaximum       int
	MaxCopies         int
	SideboardMaximum  int
	CommanderRequired bool
	CommanderMinimum  int
	CommanderMaximum  int
	Singleton         bool
}

// ConstructedRules returns baseline non-rotating constructed constraints.
//
// "Constructed" here is a useful count-rule baseline, not a claim to implement
// any official rotating format's set, ban, or card-pool rules.
func ConstructedRules() MTGRules {
	return MTGRules{Format: MTGConstructed, MainMinimum: 60, MaxCopies: 4, SideboardMaximum: 15}
}

// CommanderRules returns baseline Commander count constraints without color identity.
//
// The 98-99 main-card range accounts for one or two commander cards. Basic lands
// are the only singleton exception recognized without external card metadata.
func CommanderRules() MTGRules {
	return MTGRules{Format: MTGCommander, MainMinimum: 98, MainMaximum: 99, MaxCopies: 1, CommanderRequired: true, CommanderMinimum: 1, CommanderMaximum: 2, Singleton: true}
}

// ValidateMTG applies the selected in-place MTG legality rules to a deck.
//
// It reports all count/role violations it can observe instead of returning the
// first failure. Unknown card names are allowed because the proposal requires
// comparison and hashing to remain possible without catalog resolution.
// Reference: standard lines 25-27 and 240-242.
func ValidateMTG(deck standard.Deck, rules MTGRules) []standard.Diagnostic {
	// Use the same profile for structural validation, card-count normalization,
	// and rule dispatch so identity and legality agree on names and roles.
	profile := MTGProfile{}
	handlers := []engine.RuleHandler{
		// Every MTG format in this baseline has a primary main list.
		engine.RequiredRoleRule{Role: standard.RoleMain},
		engine.CardCountRule{Role: standard.RoleMain, Min: rules.MainMinimum, Max: rules.MainMaximum, Code: CodeMainCardCount},
	}
	// Sideboard is optional, but its upper bound applies whenever the role exists.
	if rules.SideboardMaximum > 0 {
		handlers = append(handlers, engine.CardCountRule{Role: RoleSideboard, Max: rules.SideboardMaximum, Code: CodeSideboardTooLarge})
	}
	// Commander count is expressed as a normal reusable role-count rule.
	if rules.CommanderRequired || rules.CommanderMinimum > 0 || rules.CommanderMaximum > 0 {
		handlers = append(handlers, engine.CardCountRule{Role: RoleCommanders, Min: rules.CommanderMinimum, Max: rules.CommanderMaximum, Code: CodeInvalidCommanderCount})
	}
	if rules.Format == MTGCommander {
		// Commander total is a cross-role rule rather than an MTG-specific special
		// case, so other games can reuse the same handler for their zones.
		handlers = append(handlers, engine.CombinedCardCountRule{Roles: []string{standard.RoleMain, RoleCommanders}, Exact: 100, Code: CodeCommanderTotal})
	}
	if rules.MaxCopies > 0 {
		roles := []string{standard.RoleMain}
		if rules.Format == MTGCommander {
			roles = append(roles, RoleCommanders)
		}
		copyRule := engine.CopyLimitRule{Roles: roles, Max: rules.MaxCopies}
		if rules.Singleton {
			copyRule.Code = CodeSingletonViolation
			copyRule.Exempt = isBasicLand
		}
		handlers = append(handlers, copyRule)
	}
	return engine.NewValidator(profile, handlers...).Validate(deck)
}

// isBasicLand identifies the small built-in Commander singleton exception.
// Without an external catalog this list is intentionally conservative and does
// not attempt to recognize every card with basic-land rules text.
func isBasicLand(ref string) bool {
	switch ref {
	case "plains", "island", "swamp", "mountain", "forest", "wastes":
		return true
	default:
		return false
	}
}
