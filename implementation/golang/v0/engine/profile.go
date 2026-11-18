package engine

import (
	"errors"
	"fmt"
	"strings"

	"cardhash/golang/v0/standard"
)

// GenericProfile is the engine's explicit fallback profile for non-game-specific
// work. It belongs to engine rather than standard because it is an implementation
// policy, not a core ontology contract.
type GenericProfile struct{}

// GameID identifies the generic fallback namespace.
func (GenericProfile) GameID() string { return GenericGameID }

// NormalizeRole applies the baseline role spelling policy.
func (GenericProfile) NormalizeRole(value string) (string, error) {
	role := standard.CanonicalRole(value)
	if role == "" {
		return "", errors.New("role must not be empty")
	}
	return role, nil
}

// NormalizeCardRef normalizes a generic human-readable card reference.
func (GenericProfile) NormalizeCardRef(value string) (string, error) {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" {
		return "", errors.New("card reference must not be empty")
	}
	return strings.ToLower(value), nil
}

// CanonicalTreatment preserves all non-empty generic treatment fields.
func (GenericProfile) CanonicalTreatment(value standard.Treatment) (map[string]string, bool, error) {
	if value == nil {
		return nil, false, nil
	}
	result := make(map[string]string, len(value))
	for key, field := range value {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, false, errors.New("treatment field name must not be empty")
		}
		result[key] = strings.TrimSpace(field)
	}
	return result, len(result) > 0, nil
}

// TreatmentProfileID identifies the generic treatment normalization policy.
func (GenericProfile) TreatmentProfileID() string { return GenericTreatmentProfileID }

// ConfigurableProfile is a small non-MTG profile implementation for games that
// can describe identity and treatment vocabulary without writing Go code.
//
// It is intentionally not a catalog: CardRef values remain authoritative input
// strings, and the profile only defines their namespace-level normalization.
// This makes it useful for prototypes, games with human-readable stable names,
// and future adapters that can later add catalog-backed validation.
// Reference: 20261117_01_tcg-decklist-graph-standard.md, lines 51-59, 61-77,
// and 87-93.
type ConfigurableProfile struct {
	// Namespace identifies the game/profile family, such as "pokemon/tcg".
	Namespace string
	// TreatmentNamespace identifies the versioned treatment vocabulary.
	TreatmentNamespace string
	// TreatmentFields lists the fields that contribute to treatment identity.
	TreatmentFields map[string]struct{}
}

// NewConfigurableProfile constructs a profile for a non-MTG card game.
//
// Empty treatment-field input is valid and means that treatment is preserved by
// callers but does not activate treatment hashing for this profile.
func NewConfigurableProfile(namespace, treatmentNamespace string, treatmentFields []string) (ConfigurableProfile, error) {
	// Namespaces are required because equal card names from different games must
	// not be accidentally treated as one profile's identity space.
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return ConfigurableProfile{}, fmt.Errorf("profile namespace must not be empty")
	}
	if strings.TrimSpace(treatmentNamespace) == "" {
		return ConfigurableProfile{}, fmt.Errorf("treatment namespace must not be empty")
	}
	fields := make(map[string]struct{}, len(treatmentFields))
	for _, field := range treatmentFields {
		field = strings.TrimSpace(field)
		if field == "" {
			return ConfigurableProfile{}, fmt.Errorf("treatment field name must not be empty")
		}
		fields[field] = struct{}{}
	}
	return ConfigurableProfile{Namespace: namespace, TreatmentNamespace: strings.TrimSpace(treatmentNamespace), TreatmentFields: fields}, nil
}

// GameID returns the configured game namespace for diagnostics and negotiation.
func (p ConfigurableProfile) GameID() string { return p.Namespace }

// NormalizeRole applies the core role spelling policy for a configurable game.
func (p ConfigurableProfile) NormalizeRole(value string) (string, error) {
	role := standard.CanonicalRole(value)
	if role == "" {
		return "", fmt.Errorf("role must not be empty")
	}
	return role, nil
}

// NormalizeCardRef applies deterministic human-readable-name normalization.
func (p ConfigurableProfile) NormalizeCardRef(value string) (string, error) {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" {
		return "", fmt.Errorf("card reference must not be empty")
	}
	return strings.ToLower(value), nil
}

// CanonicalTreatment projects only fields declared by this game's profile.
func (p ConfigurableProfile) CanonicalTreatment(value standard.Treatment) (map[string]string, bool, error) {
	if value == nil {
		return nil, false, nil
	}
	result := make(map[string]string)
	for key, field := range value {
		if strings.TrimSpace(key) == "" {
			return nil, false, fmt.Errorf("treatment field name must not be empty")
		}
		if _, known := p.TreatmentFields[key]; !known {
			continue
		}
		result[key] = strings.ToLower(strings.TrimSpace(field))
	}
	return result, len(result) > 0, nil
}

// TreatmentProfileID returns the immutable treatment vocabulary identifier.
func (p ConfigurableProfile) TreatmentProfileID() string { return p.TreatmentNamespace }
