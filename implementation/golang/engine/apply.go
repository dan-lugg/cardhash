package engine

// This file applies the proposal's materialized changes to a source deck. It
// keeps application separate from comparison because a changeset can be stored,
// transmitted, or explained independently of provenance, and because treatment
// omission requires an explicit caller policy.
//
// Reference: 20261117_01_tcg-decklist-graph-standard.md, "Deck Evolution and
// Relations" (lines 159-165), especially the treatment-omission rule at line
// 161.

import (
	"fmt"
	"sort"

	"cardhash/golang/standard"
)

// TreatmentPolicy controls omitted treatment while applying a structural changeset.
//
// Structural ChangeItems do not carry treatment, so application must not invent
// a printing for additions. The policy only controls whether existing treatment
// is cleared from the resulting structural representation or retained where a
// source bucket supplies it.
// Reference: standard line 161.
type TreatmentPolicy string

const (
	// ClearTreatment produces an untreated result for structural application.
	ClearTreatment TreatmentPolicy = "clear"
	// PreserveTreatment retains matching source treatment while applying removals;
	// newly added cards remain untreated because no source match exists.
	PreserveTreatment TreatmentPolicy = "preserve"
)

// ApplyChangeSet applies a normalized changeset to a source deck.
//
// Treatment-scope changes are exact bucket operations. Structure-scope changes
// operate on canonical CardRefs and use the supplied omission policy. In both
// cases the caller receives a newly materialized deck whose hashes can be
// recalculated from actual resulting data.
// Reference: standard lines 161-165 and 169-175.
func ApplyChangeSet(source standard.Deck, changes standard.ChangeSet, profile standard.HashProfile, policy TreatmentPolicy) (standard.Deck, error) {
	// Normalize a nil profile exactly as hashing and validation do.
	if profile == nil {
		profile = GenericProfile{}
	}
	// Never apply against an invalid source; otherwise an apparently valid delta
	// could produce a target that no conforming consumer can hash.
	if err := source.Validate(profile); err != nil {
		return standard.Deck{}, fmt.Errorf("source: %w", err)
	}
	if changes.Scope != standard.StructureScope && changes.Scope != standard.TreatmentScope {
		return standard.Deck{}, fmt.Errorf("unknown changeset scope %q", changes.Scope)
	}
	// Treatment scope has exact buckets, while structure scope needs the omission
	// policy and may redistribute removals across source treatment buckets.
	if changes.Scope == standard.TreatmentScope {
		return applyTreatmentChanges(source, changes, profile)
	}
	return applyStructureChanges(source, changes, profile, policy)
}

// applyTreatmentChanges applies signed operations to exact treatment buckets.
// A negative operation cannot remove more copies from a bucket than the source
// provides; this is the application-time validation required for replay safety.
func applyTreatmentChanges(source standard.Deck, changes standard.ChangeSet, profile standard.HashProfile) (standard.Deck, error) {
	// Flatten source data using the same canonical treatment keys used by Compare.
	buckets := flatten(source, profile, standard.TreatmentScope)
	items, err := normalizeChangeItems(changes.Items, profile, standard.TreatmentScope)
	if err != nil {
		return standard.Deck{}, err
	}
	for _, item := range items {
		// Apply each normalized signed quantity to its exact bucket.
		key := changeKey(item.Role, item.CardRef, item.Treatment)
		current := buckets[key]
		current.Role, current.CardRef, current.Treatment = item.Role, item.CardRef, item.Treatment
		current.Quantity += item.Quantity
		// Negative quantities are valid only when the source has enough matching
		// cards; allowing an underflow would make target identity ambiguous.
		if current.Quantity < 0 {
			return standard.Deck{}, fmt.Errorf("change removes more %q from role %q than source contains", item.CardRef, item.Role)
		}
		// Remove empty buckets so canonical output never contains zero quantities.
		if current.Quantity == 0 {
			delete(buckets, key)
		} else {
			buckets[key] = current
		}
	}
	return deckFromBuckets(source, buckets, profile), nil
}

// applyStructureChanges applies CardRef-level operations while treating
// treatment as omitted. Removal is deterministic: source treatment buckets are
// visited by their canonical treatment key, and additions are always untreated.
// Reference: standard lines 161 and 169-173.
func applyStructureChanges(source standard.Deck, changes standard.ChangeSet, profile standard.HashProfile, policy TreatmentPolicy) (standard.Deck, error) {
	// Reject an unspecified policy rather than silently choosing a treatment
	// interpretation that may differ between implementations.
	if policy != ClearTreatment && policy != PreserveTreatment {
		return standard.Deck{}, fmt.Errorf("unknown treatment policy %q", policy)
	}
	// Start from source so metadata and roles survive application; only card items
	// are changed below.
	result := source
	items, err := normalizeChangeItems(changes.Items, profile, standard.StructureScope)
	if err != nil {
		return standard.Deck{}, err
	}
	for _, item := range items {
		// Locate the target role using canonical role semantics.
		list := standard.LookupRole(result, profile, item.Role)
		if list == nil {
			// A positive addition can create a role; a negative operation cannot
			// remove cards from a role that is absent from the source.
			if item.Quantity < 0 {
				return standard.Deck{}, fmt.Errorf("role %q does not exist", item.Role)
			}
			result.Lists = append(result.Lists, standard.CardList{Role: item.Role})
			list = standard.LookupRole(result, profile, item.Role)
		}
		if item.Quantity > 0 {
			// Structural additions have no matching source item, so they must not
			// receive invented treatment data.
			list.Items = append(list.Items, standard.CardItem{Quantity: item.Quantity, CardRef: item.CardRef})
			continue
		}
		remaining := -item.Quantity
		// Copy and sort buckets so the source deck's presentation order cannot
		// determine which treated copies are consumed by a structural removal.
		buckets := append([]standard.CardItem(nil), list.Items...)
		sort.SliceStable(buckets, func(i, j int) bool { return treatmentKey(buckets[i].Treatment) < treatmentKey(buckets[j].Treatment) })
		for index := range buckets {
			// Only matching CardRefs can satisfy the structural removal; treatment is
			// intentionally ignored for this scope.
			ref, _ := profile.NormalizeCardRef(buckets[index].CardRef)
			if ref != item.CardRef || remaining == 0 {
				continue
			}
			removed := buckets[index].Quantity
			if removed > remaining {
				removed = remaining
			}
			buckets[index].Quantity -= removed
			remaining -= removed
		}
		if remaining > 0 {
			return standard.Deck{}, fmt.Errorf("change removes more %q from role %q than source contains", item.CardRef, item.Role)
		}
		// Drop depleted buckets so the resulting deck satisfies positive-quantity
		// invariants rather than carrying zero entries internally.
		list.Items = buckets[:0]
		for _, bucket := range buckets {
			if bucket.Quantity > 0 {
				list.Items = append(list.Items, bucket)
			}
		}
	}
	if policy == ClearTreatment {
		// Clear mode makes all resulting cards untreated. Preserve mode leaves
		// surviving source buckets untouched and additions already untreated.
		for listIndex := range result.Lists {
			for itemIndex := range result.Lists[listIndex].Items {
				result.Lists[listIndex].Items[itemIndex].Treatment = nil
			}
		}
	}
	return result, nil
}

// deckFromBuckets rebuilds a deck from canonical buckets after treatment-scope
// application. Map order is discarded and each role's items are sorted for a
// stable human-facing result, even though hashes do not depend on that order.
func deckFromBuckets(source standard.Deck, buckets map[string]standard.ChangeItem, profile standard.HashProfile) standard.Deck {
	// Preserve deck metadata and role containers while replacing their items.
	result := source
	for index := range result.Lists {
		result.Lists[index].Items = nil
	}
	for _, item := range buckets {
		// Reuse an existing role when possible; otherwise create the role required
		// by a positive change item.
		list := standard.LookupRole(result, profile, item.Role)
		if list == nil {
			result.Lists = append(result.Lists, standard.CardList{Role: item.Role})
			list = standard.LookupRole(result, profile, item.Role)
		}
		list.Items = append(list.Items, standard.CardItem{Quantity: item.Quantity, CardRef: item.CardRef, Treatment: item.Treatment})
	}
	for index := range result.Lists {
		// Stable output ordering makes serialized results easier to inspect and
		// prevents callers from observing random map iteration order.
		sort.Slice(result.Lists[index].Items, func(i, j int) bool {
			if result.Lists[index].Items[i].CardRef != result.Lists[index].Items[j].CardRef {
				return result.Lists[index].Items[i].CardRef < result.Lists[index].Items[j].CardRef
			}
			return treatmentKey(result.Lists[index].Items[i].Treatment) < treatmentKey(result.Lists[index].Items[j].Treatment)
		})
	}
	return result
}
