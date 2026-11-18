package engine

// Engine constants identify implementation-level algorithms, diagnostics, and
// fallback profile behavior. They are intentionally outside standard so the
// ontology package remains independent of one Go engine's hash envelope and
// validation vocabulary.
const (
	// GenericGameID identifies the profile-independent fallback namespace.
	GenericGameID = "generic"
	// GenericTreatmentProfileID identifies the baseline generic treatment policy.
	GenericTreatmentProfileID = "generic/treatment/1"
	// DiagnosticSeverityError is the baseline failed-rule severity.
	DiagnosticSeverityError          = "error"
	DiagnosticCodeMissingProfile     = "missing-profile"
	DiagnosticCodeInvalidDeck        = "invalid-deck"
	DiagnosticCodeNilRuleHandler     = "nil-rule-handler"
	DiagnosticCodeMissingRole        = "missing-role"
	DiagnosticCodeCardCountTooSmall  = "card-count-too-small"
	DiagnosticCodeCardCountTooLarge  = "card-count-too-large"
	DiagnosticCodeCombinedCardCount  = "combined-card-count"
	DiagnosticCodeCopyLimit          = "copy-limit"
	RuleHandlerNameRequiredRole      = "required-role"
	RuleHandlerNameCardCount         = "card-count"
	RuleHandlerNameCombinedCardCount = "combined-card-count"
	RuleHandlerNameCopyLimit         = "copy-limit"
	HashDomainStructureList          = "tcg/structure-list/1"
	HashDomainTreatmentListPrefix    = "tcg/treatment-list/1/"
	HashDomainStructureDeck          = "tcg/structure-deck/1"
	HashDomainTreatmentDeck          = "tcg/treatment-deck/1"
	HashDomainStructureChangeSet     = "tcg/structure-changeset/1"
	HashDomainTreatmentChangeSet     = "tcg/treatment-changeset/1"
)
