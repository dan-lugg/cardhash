package engine

import (
	"fmt"
	"sort"

	"cardhash/golang/standard"
)

// ValidationContext carries the deck and profile shared by all rule handlers.
//
// Handlers receive one context rather than game-specific globals so the same
// engine can validate MTG, another TCG, or a test profile with different role
// and card-reference semantics.
// Reference: standard lines 23-27 and 244-268.
type ValidationContext struct {
	Deck    standard.Deck
	Profile standard.GameProfile
}

// RuleHandler is one independently testable legality or construction rule.
//
// A handler reports diagnostics instead of returning a boolean so validators can
// expose all observed violations and attach role/card context to each result.
// Reference: standard lines 25-27 and 220-220.
type RuleHandler interface {
	Name() string
	Validate(ValidationContext) []standard.Diagnostic
}

// Validator composes a profile with ordered, reusable rule handlers.
type Validator struct {
	// Profile supplies the game namespace and normalization rules for handlers.
	Profile standard.GameProfile
	// Handlers are evaluated in this order so diagnostic ordering is intentional.
	Handlers []RuleHandler
}

// NewValidator creates a generic validator with the supplied rule pipeline.
func NewValidator(profile standard.GameProfile, handlers ...RuleHandler) Validator {
	return Validator{Profile: profile, Handlers: append([]RuleHandler(nil), handlers...)}
}

// Validate executes structural validation followed by every configured handler.
//
// Structural failure stops rule execution because handlers cannot safely reason
// about invalid quantities or roles. Rule failures do not short-circuit; the
// caller receives the complete diagnostic set in handler order.
func (v Validator) Validate(deck standard.Deck) []standard.Diagnostic {
	if v.Profile == nil {
		return []standard.Diagnostic{{Severity: DiagnosticSeverityError, Code: DiagnosticCodeMissingProfile, Message: "validator profile must not be nil"}}
	}
	if err := deck.Validate(v.Profile); err != nil {
		return []standard.Diagnostic{{Severity: DiagnosticSeverityError, Code: DiagnosticCodeInvalidDeck, Message: err.Error()}}
	}
	context := ValidationContext{Deck: deck, Profile: v.Profile}
	diagnostics := make([]standard.Diagnostic, 0)
	for _, handler := range v.Handlers {
		if handler == nil {
			diagnostics = append(diagnostics, standard.Diagnostic{Severity: DiagnosticSeverityError, Code: DiagnosticCodeNilRuleHandler, Message: "validator contains a nil rule handler"})
			continue
		}
		diagnostics = append(diagnostics, handler.Validate(context)...)
	}
	return diagnostics
}

// RequiredRoleRule requires a role-bearing CardList to exist.
type RequiredRoleRule struct {
	Role string
}

// Name identifies the required-role handler in diagnostics and instrumentation.
func (r RequiredRoleRule) Name() string { return RuleHandlerNameRequiredRole }

// Validate reports a missing role without asserting card-count legality.
func (r RequiredRoleRule) Validate(context ValidationContext) []standard.Diagnostic {
	if findRole(context.Deck, context.Profile, r.Role) != nil {
		return nil
	}
	return []standard.Diagnostic{{Severity: DiagnosticSeverityError, Code: DiagnosticCodeMissingRole, Message: fmt.Sprintf("required role %q is missing", r.Role), Role: r.Role}}
}

// CardCountRule constrains the total quantity in one role.
type CardCountRule struct {
	// Role identifies the semantic zone whose quantity is constrained.
	Role string
	// Min is the inclusive lower bound; zero means no lower bound.
	Min int
	// Max is the inclusive upper bound; zero means no upper bound.
	Max int
	// Code allows a game profile to retain a domain-specific diagnostic code.
	Code string
}

// Name identifies the per-role count handler.
func (r CardCountRule) Name() string { return RuleHandlerNameCardCount }

// Validate reports lower and upper bound violations for the configured role.
func (r CardCountRule) Validate(context ValidationContext) []standard.Diagnostic {
	list := findRole(context.Deck, context.Profile, r.Role)
	count := countCards(list)
	diagnostics := make([]standard.Diagnostic, 0, 2)
	code := r.Code
	if code == "" {
		code = DiagnosticCodeCardCountTooSmall
	}
	if r.Min > 0 && count < r.Min {
		diagnostics = append(diagnostics, standard.Diagnostic{Severity: DiagnosticSeverityError, Code: code, Message: fmt.Sprintf("role %q contains %d cards; minimum is %d", r.Role, count, r.Min), Role: r.Role})
	}
	code = r.Code
	if code == "" {
		code = DiagnosticCodeCardCountTooLarge
	}
	if r.Max > 0 && count > r.Max {
		diagnostics = append(diagnostics, standard.Diagnostic{Severity: DiagnosticSeverityError, Code: code, Message: fmt.Sprintf("role %q contains %d cards; maximum is %d", r.Role, count, r.Max), Role: r.Role})
	}
	return diagnostics
}

// CombinedCardCountRule constrains the combined quantity of several roles.
type CombinedCardCountRule struct {
	// Roles are summed after profile role normalization.
	Roles []string
	// Exact is the required combined quantity.
	Exact int
	// Code allows a profile to identify the cross-role failure in domain terms.
	Code string
}

// Name identifies the combined-total handler.
func (r CombinedCardCountRule) Name() string { return RuleHandlerNameCombinedCardCount }

// Validate reports a mismatch between the combined role total and Exact.
func (r CombinedCardCountRule) Validate(context ValidationContext) []standard.Diagnostic {
	total := 0
	for _, role := range r.Roles {
		total += countCards(findRole(context.Deck, context.Profile, role))
	}
	if total == r.Exact {
		return nil
	}
	code := r.Code
	if code == "" {
		code = DiagnosticCodeCombinedCardCount
	}
	return []standard.Diagnostic{{Severity: DiagnosticSeverityError, Code: code, Message: fmt.Sprintf("roles %v contain %d cards; required total is %d", r.Roles, total, r.Exact)}}
}

// CopyLimitRule constrains normalized CardRef quantities across selected roles.
type CopyLimitRule struct {
	// Roles identifies the zones whose normalized CardRefs are aggregated.
	Roles []string
	// Max is the inclusive quantity allowed for each non-exempt CardRef.
	Max int
	// Code allows a profile to label the same generic violation in domain terms.
	Code string
	// Exempt identifies profile-specific exceptions such as MTG basic lands.
	Exempt func(string) bool
}

// Name identifies the copy-limit handler.
func (r CopyLimitRule) Name() string { return RuleHandlerNameCopyLimit }

// Validate reports references whose aggregate quantity exceeds Max.
func (r CopyLimitRule) Validate(context ValidationContext) []standard.Diagnostic {
	counts := make(map[string]int)
	for _, role := range r.Roles {
		for ref, quantity := range cardCounts(findRole(context.Deck, context.Profile, role), context.Profile) {
			counts[ref] += quantity
		}
	}
	code := r.Code
	if code == "" {
		code = DiagnosticCodeCopyLimit
	}
	diagnostics := make([]standard.Diagnostic, 0)
	refs := make([]string, 0, len(counts))
	for ref := range counts {
		refs = append(refs, ref)
	}
	// Stable diagnostic ordering keeps validation output reproducible for clients
	// that serialize or snapshot diagnostics, without affecting deck identity.
	sort.Strings(refs)
	for _, ref := range refs {
		quantity := counts[ref]
		if quantity <= r.Max || (r.Exempt != nil && r.Exempt(ref)) {
			continue
		}
		diagnostics = append(diagnostics, standard.Diagnostic{Severity: DiagnosticSeverityError, Code: code, Message: fmt.Sprintf("card appears %d times; copy limit is %d", quantity, r.Max), CardRef: ref})
	}
	return diagnostics
}

// findRole resolves a role through the active game's role namespace.
func findRole(deck standard.Deck, profile standard.GameProfile, role string) *standard.CardList {
	return standard.LookupRole(deck, profile, role)
}

// countCards totals quantities for one role without interpreting card identity.
// It is shared by role-count and combined-total handlers because both rules
// measure the same standard quantity semantics.
func countCards(list *standard.CardList) int {
	if list == nil {
		return 0
	}
	total := 0
	for _, item := range list.Items {
		total += item.Quantity
	}
	return total
}

// cardCounts aggregates normalized CardRefs for copy-limit handlers. Treatment
// is intentionally ignored because copy limits are game-rule identity checks,
// not printing identity checks.
func cardCounts(list *standard.CardList, profile standard.HashProfile) map[string]int {
	result := make(map[string]int)
	if list == nil {
		return result
	}
	for _, item := range list.Items {
		ref, _ := profile.NormalizeCardRef(item.CardRef)
		result[ref] += item.Quantity
	}
	return result
}
