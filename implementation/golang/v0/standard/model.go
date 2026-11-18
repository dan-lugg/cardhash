// Package standard contains the shared ontology and contracts of the TCG
// Decklist Graph Standard. It MUST remain independent of rule engines, game
// catalogs, and implementation-specific legality policy.
//
// The package intentionally keeps the core representation independent of a
// catalog, database, transport encoding, or legality engine. That separation
// follows the proposal's distinction between card identity, optional metadata,
// treatment data, and externally evaluated legality.
//
// Reference: 20261117_01_tcg-decklist-graph-standard.md, "Objective", "Deck",
// "CardList", "CardItem", and "CardMeta" (lines 9-17, 19-35, 37-49, and 67-77).
package standard

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Treatment stores printing or physical-realization attributes for a card item.
//
// The map is deliberately open-ended because the proposal allows game-specific
// printing fields and unknown extension data. A HashProfile decides which keys
// are known and hash-significant; this type itself does not make that decision.
// Reference: standard sections Treatment and Treatment Profiles (lines 79-93).
type Treatment map[string]string

// CardMeta stores optional, non-identity information about a card reference.
//
// CardMeta is kept beside CardRef rather than folded into it so catalog IDs,
// localized names, and display data cannot silently change structural identity.
// Reference: standard section CardMeta (lines 67-77).
type CardMeta struct {
	DisplayName string            `json:"displayName,omitempty"`
	Values      map[string]string `json:"values,omitempty"`
}

// CardItem represents a positive quantity of one card reference.
//
// Quantity belongs to the item rather than the list, and zero is intentionally
// not representable as a valid canonical item. Treatment remains optional so a
// list can mix specified printings with copies whose printing is unspecified.
// Reference: standard sections CardItem and CardList Hashes (lines 43-49 and 118-126).
type CardItem struct {
	Quantity  int       `json:"quantity"`
	CardRef   string    `json:"cardRef"`
	Treatment Treatment `json:"treatment,omitempty"`
	Meta      *CardMeta `json:"meta,omitempty"`
}

// CardList is a collection of card items with one semantic role.
//
// Role is stored explicitly because the same card content may occur in several
// deck zones while retaining the same list-level card hash. The role therefore
// participates in the containing deck hash, not in the list card-content hash.
// Reference: standard sections Deck and Deck and CardList Structure (lines 19-33 and
// 95-112).
type CardList struct {
	Role  string     `json:"role"`
	Items []CardItem `json:"items"`
}

// Deck is an explicit container of one or more role-bearing card lists.
//
// Name and source hashes are metadata. They are retained for display and direct
// provenance, but the hash implementation below intentionally excludes them
// from content identity.
// Reference: standard sections Deck (lines 19-35) and Deck Hashes (lines 29-31).
type Deck struct {
	Name                string     `json:"name,omitempty"`
	Lists               []CardList `json:"lists"`
	SourceStructureHash string     `json:"sourceStructureHash,omitempty"`
	SourceTreatmentHash string     `json:"sourceTreatmentHash,omitempty"`
}

// HashProfile defines game-specific identity and treatment normalization rules.
//
// The proposal leaves normalization and treatment-field selection to a
// versionable game profile. Making that profile an interface keeps the core
// model usable for games other than MTG while forcing every hash operation to
// make its profile dependency explicit.
// Reference: standard sections CardRef Naming and Normalization and Treatment Profiles
// (lines 61-65 and 87-93).
type HashProfile interface {
	NormalizeCardRef(string) (string, error)
	CanonicalTreatment(Treatment) (map[string]string, bool, error)
	TreatmentProfileID() string
}

// RoleProfile adds game-specific role normalization to hash identity.
//
// HashProfile remains small for compatibility with profiles that only define
// card identity and treatment. Profiles that also implement RoleProfile allow
// the core to canonicalize roles according to the game's namespace rules.
// Reference: standard sections Deck and CardList Structure and Compatibility
// and Versioning (lines 95-112 and 266-272).
type RoleProfile interface {
	NormalizeRole(string) (string, error)
}

// GameProfile is the complete profile contract used by generic validation.
//
// GameID provides a namespace for diagnostics and future capability negotiation;
// the existing HashProfile methods retain the proposal's identity boundary.
// Reference: standard sections CardRef, Treatment Profiles, and Compatibility
// and Versioning (lines 51-65, 87-93, and 266-272).
type GameProfile interface {
	HashProfile
	RoleProfile
	GameID() string
}

// Validate checks the structural invariants that apply before legality evaluation.
//
// This method deliberately does not check format legality. It verifies only the
// representational contract needed by hashing and comparison: at least one
// list, unique canonical roles, positive quantities, resolvable profile-level
// references, and profile-acceptable treatments.
// Reference: standard sections Deck, CardItem, and legality separation (lines 19-27
// and 43-49).
func (d Deck) Validate(profile HashProfile) error {
	// The ontology package does not choose an implementation profile; callers must
	// supply one so profile selection remains explicit at the engine boundary.
	if profile == nil {
		return errors.New("hash profile must not be nil")
	}
	if len(d.Lists) == 0 {
		return errors.New("deck must contain at least one card list")
	}
	roles := make(map[string]struct{}, len(d.Lists))
	for index, list := range d.Lists {
		// Canonical role comparison prevents cosmetic casing/spacing from creating
		// duplicate semantic zones or different deck identities.
		role, roleErr := NormalizeRole(profile, list.Role)
		if roleErr != nil {
			return fmt.Errorf("list %d: %w", index, roleErr)
		}
		if _, exists := roles[role]; exists {
			return fmt.Errorf("duplicate deck role %q", role)
		}
		roles[role] = struct{}{}
		for itemIndex, item := range list.Items {
			// Zero quantities are intentionally rejected instead of normalized into
			// the canonical representation, matching the proposal's invalid-item rule.
			if item.Quantity <= 0 {
				return fmt.Errorf("list %q item %d has non-positive quantity", role, itemIndex)
			}
			// Profile normalization validates identity syntax without requiring catalog
			// resolution, so unresolved names remain diffable.
			if _, err := profile.NormalizeCardRef(item.CardRef); err != nil {
				return fmt.Errorf("list %q item %d: %w", role, itemIndex, err)
			}
			// Treatment validation is profile-specific and does not make legality part
			// of the content identity.
			if _, _, err := profile.CanonicalTreatment(item.Treatment); err != nil {
				return fmt.Errorf("list %q item %d treatment: %w", role, itemIndex, err)
			}
		}
	}
	return nil
}

// CanonicalRole applies the baseline role spelling policy used when a profile
// does not provide a stronger role namespace. Role normalization is an
// implementation policy for this baseline because the proposal requires
// deterministic role/hash pairs but leaves role namespaces to game profiles.
// Future profiles may need a richer role namespace.
// Reference: standard sections Deck and Deck and CardList Structure (lines 21-29
// and 95-112).
func CanonicalRole(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

// NormalizeRole delegates role identity to a profile when available, while
// retaining a deterministic fallback for legacy HashProfile implementations.
func NormalizeRole(profile HashProfile, value string) (string, error) {
	if roleProfile, ok := profile.(RoleProfile); ok {
		return roleProfile.NormalizeRole(value)
	}
	role := CanonicalRole(value)
	if role == "" {
		return "", errors.New("role must not be empty")
	}
	return role, nil
}

// LookupRole locates a list using the same role normalization used by hashing
// and structural validation. It is shared by generic rules and changeset replay
// so a non-MTG profile cannot validate one role spelling but apply another.
func LookupRole(deck Deck, profile HashProfile, role string) *CardList {
	canonical, err := NormalizeRole(profile, role)
	if err != nil {
		return nil
	}
	for index := range deck.Lists {
		candidate, candidateErr := NormalizeRole(profile, deck.Lists[index].Role)
		if candidateErr == nil && candidate == canonical {
			return &deck.Lists[index]
		}
	}
	return nil
}

// SortedRoles returns a copy of lists ordered by canonical role name.
//
// The copy protects the caller's presentation order while offering a stable
// view for diagnostics and future encoders. Hashing performs its own explicit
// canonical ordering rather than relying on this convenience method.
// Reference: standard sections CardList Hashes and Canonical Input (lines 128-134).
func (d Deck) SortedRoles() []CardList {
	// Copy first so sorting does not mutate the source deck's user-facing order.
	result := append([]CardList(nil), d.Lists...)
	// Stable role ordering makes inspection and future serialization predictable.
	sort.Slice(result, func(i, j int) bool { return result[i].Role < result[j].Role })
	return result
}
