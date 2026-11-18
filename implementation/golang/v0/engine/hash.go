package engine

// This file contains the executable form of the proposal's content-identity
// rules. The implementation uses Go's deterministic encoding/json behavior for
// the local baseline, while keeping the canonical input types private so callers
// cannot accidentally hash display metadata.
//
// Reference: 20261117_01_tcg-decklist-graph-standard.md, "CardList Hashes"
// (lines 114-151), and "Deck" hash rules (lines 29-31).

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"cardhash/golang/v0/standard"
)

// canonicalItem is the hash-visible projection of a CardItem.
//
// CardMeta is intentionally absent. A nil treatment is retained in treatment
// scope when any significant treatment exists elsewhere in the list, allowing
// untreated copies to remain distinct from treated buckets without changing
// structural identity.
// Reference: standard lines 118-126 and 171-175.
type canonicalItem struct {
	Quantity  int               `json:"quantity"`
	CardRef   string            `json:"cardRef"`
	Treatment map[string]string `json:"treatment,omitempty"`
}

// canonicalList is the ordered JSON payload used for list-level hashing.
// Items are sorted before this value is serialized.
type canonicalList struct {
	Items []canonicalItem `json:"items"`
}

// canonicalDeckEntry is one role/hash pair in the deck-level hash input.
// Role is included here because role is meaningful to a deck but not to a list's
// card-content hash.
// Reference: standard lines 29 and 110.
type canonicalDeckEntry struct {
	Role string `json:"role"`
	Hash string `json:"hash"`
}

// ListHashes returns structural and treatment hashes for a canonicalized list.
//
// The structural projection merges all quantities by canonical CardRef and
// therefore ignores treatment partitions. The treatment projection keeps one
// bucket per recognized treatment value. When no recognized treatment exists,
// this method returns the exact structural hash twice, as required by the
// proposal's fallback rule.
// Reference: standard sections Hash Scope and Canonical Input (lines 118-134).
func ListHashes(list standard.CardList, profile standard.HashProfile) (structure, treatment string, err error) {
	// A nil profile is a usable generic baseline rather than a panic condition.
	if profile == nil {
		profile = GenericProfile{}
	}
	// Normalize once so both hash scopes share the same identity decisions.
	canonical, hasTreatment, err := canonicalizeList(list, profile)
	if err != nil {
		return "", "", err
	}
	// Structural hashing intentionally collapses treatment buckets back into
	// quantities by CardRef.
	structureInput := canonicalList{Items: structureItems(canonical)}
	structure, err = hashJSON(HashDomainStructureList, structureInput)
	if err != nil {
		return "", "", err
	}
	// No significant treatment means treatment identity is exactly structural
	// identity, not a profile-specific empty treatment namespace.
	if !hasTreatment {
		return structure, structure, nil
	}
	// Include the immutable treatment-profile ID whenever treatment contributes
	// to identity, preventing profile-version ambiguity.
	treatment, err = hashJSON(HashDomainTreatmentListPrefix+profile.TreatmentProfileID(), canonical)
	return structure, treatment, err
}

// DeckHashes returns structural and treatment hashes for a deck's role/hash pairs.
//
// Deck metadata, including Name and source references, is excluded. Each list is
// hashed independently, then role/hash pairs are sorted by canonical role before
// the deck-level domain-separated hash is calculated.
// Reference: standard sections Deck (lines 29-35) and Deck and CardList Structure
// (lines 95-112).
func DeckHashes(deck standard.Deck, profile standard.HashProfile) (structure, treatment string, err error) {
	// Validate before hashing so malformed canonical inputs cannot become stable
	// identities that other implementations would be required to accept.
	if err := deck.Validate(profile); err != nil {
		return "", "", err
	}
	if profile == nil {
		profile = GenericProfile{}
	}
	entriesStructure := make([]canonicalDeckEntry, 0, len(deck.Lists))
	entriesTreatment := make([]canonicalDeckEntry, 0, len(deck.Lists))
	for _, list := range deck.Lists {
		// List hashes deliberately do not include role; the containing deck adds it
		// back through the role/hash pair below.
		listStructure, listTreatment, hashErr := ListHashes(list, profile)
		if hashErr != nil {
			return "", "", hashErr
		}
		// Canonicalize role spelling through the active profile before it participates
		// in either deck hash.
		role, roleErr := standard.NormalizeRole(profile, list.Role)
		if roleErr != nil {
			return "", "", roleErr
		}
		entriesStructure = append(entriesStructure, canonicalDeckEntry{Role: role, Hash: listStructure})
		entriesTreatment = append(entriesTreatment, canonicalDeckEntry{Role: role, Hash: listTreatment})
	}
	// Sort both scopes independently so source presentation order cannot affect
	// deck identity.
	sort.Slice(entriesStructure, func(i, j int) bool { return entriesStructure[i].Role < entriesStructure[j].Role })
	sort.Slice(entriesTreatment, func(i, j int) bool { return entriesTreatment[i].Role < entriesTreatment[j].Role })
	structure, err = hashJSON(HashDomainStructureDeck, entriesStructure)
	if err != nil {
		return "", "", err
	}
	treatment, err = hashJSON(HashDomainTreatmentDeck, entriesTreatment)
	return structure, treatment, err
}

// canonicalizeList creates the treatment-aware canonical multiset for a list.
//
// The structural map is populated alongside treatment buckets so every item
// contributes to both scopes. Treatment buckets are keyed by canonical JSON of
// reference plus canonical treatment, making repeated equivalent items merge
// while preserving distinct treatment values.
// Reference: standard lines 130-134 and 171-175.
func canonicalizeList(list standard.CardList, profile standard.HashProfile) (canonicalList, bool, error) {
	// structural is retained as an explicit accumulation guard and documents the
	// structural scope even though structureItems recomputes its projection from
	// the treatment-aware result.
	structural := make(map[string]int)
	// treatmentBuckets represent the treatment-inclusive multiset.
	treatmentBuckets := make(map[string]canonicalItem)
	hasTreatment := false
	for _, item := range list.Items {
		// Canonical list input never contains zero or negative quantities.
		if item.Quantity <= 0 {
			return canonicalList{}, false, fmt.Errorf("card %q has non-positive quantity", item.CardRef)
		}
		// CardRef normalization is profile-owned because names and namespaces vary
		// by game.
		ref, err := profile.NormalizeCardRef(item.CardRef)
		if err != nil {
			return canonicalList{}, false, err
		}
		structural[ref] += item.Quantity
		// The profile both filters unknown fields and reports whether any recognized
		// field should activate treatment hashing for this list.
		canonicalTreatment, significant, treatmentErr := profile.CanonicalTreatment(item.Treatment)
		if treatmentErr != nil {
			return canonicalList{}, false, treatmentErr
		}
		hasTreatment = hasTreatment || significant
		// JSON is used only as an internal deterministic composite key here; the
		// actual hash payload is produced by hashJSON below.
		keyBytes, marshalErr := json.Marshal(struct {
			Ref       string
			Treatment map[string]string
		}{ref, canonicalTreatment})
		if marshalErr != nil {
			return canonicalList{}, false, marshalErr
		}
		key := string(keyBytes)
		// Repeated equivalent items combine quantities before serialization.
		bucket := treatmentBuckets[key]
		bucket.CardRef = ref
		bucket.Treatment = canonicalTreatment
		bucket.Quantity += item.Quantity
		treatmentBuckets[key] = bucket
	}
	// Map iteration order is not identity-bearing, so materialize and sort it.
	result := canonicalList{Items: make([]canonicalItem, 0, len(treatmentBuckets))}
	for _, item := range treatmentBuckets {
		result.Items = append(result.Items, item)
	}
	// Sort first by canonical CardRef and then by canonical treatment JSON. This
	// gives equivalent representations the same serialized byte sequence.
	sort.Slice(result.Items, func(i, j int) bool {
		if result.Items[i].CardRef != result.Items[j].CardRef {
			return result.Items[i].CardRef < result.Items[j].CardRef
		}
		return treatmentKey(result.Items[i].Treatment) < treatmentKey(result.Items[j].Treatment)
	})
	return result, hasTreatment, nil
}

// structureItems projects treatment-aware buckets into structural quantities.
//
// This second aggregation is important: differently treated copies of one card
// must remain separate for TreatmentHash but combine for StructureHash.
// Reference: standard lines 118-124.
func structureItems(list canonicalList) []canonicalItem {
	quantities := make(map[string]int)
	for _, item := range list.Items {
		// Ignore treatment while summing structural card composition.
		quantities[item.CardRef] += item.Quantity
	}
	result := make([]canonicalItem, 0, len(quantities))
	for ref, quantity := range quantities {
		result = append(result, canonicalItem{Quantity: quantity, CardRef: ref})
	}
	// Sort by canonical reference before returning the structural payload.
	sort.Slice(result, func(i, j int) bool { return result[i].CardRef < result[j].CardRef })
	return result
}

// treatmentKey creates a deterministic comparison key for treatment buckets.
// Encoding/json sorts map keys, so equivalent field maps produce equivalent keys.
func treatmentKey(treatment map[string]string) string {
	data, _ := json.Marshal(treatment)
	return string(data)
}

// hashJSON applies the baseline domain-separated canonical JSON hash contract.
//
// The domain prevents a list payload from being confused with a deck or
// changeset payload even when their JSON values happen to look alike. SHA-256
// and lowercase hexadecimal are the current implementation choices; the
// proposal leaves display format and broader hash agility as future work.
// Reference: standard lines 128-135 and adversarial review "Hashing And
// Canonicalization".
func hashJSON(domain string, value any) (string, error) {
	// Keep domain and value in a fixed-field envelope so every hash scope has a
	// distinct, inspectable canonical input.
	data, err := json.Marshal(struct {
		Domain string `json:"domain"`
		Value  any    `json:"value"`
	}{domain, value})
	if err != nil {
		return "", err
	}
	// Hash the exact UTF-8 bytes emitted by encoding/json.
	digest := sha256.Sum256(data)
	// Hex is convenient for transport and debugging while remaining unambiguous.
	return hex.EncodeToString(digest[:]), nil
}
