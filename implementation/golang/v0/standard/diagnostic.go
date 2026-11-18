package standard

// Diagnostic is a transport-neutral validation result. Legality diagnostics
// remain outside deck content identity and may change as rule profiles evolve.
// Reference: 20261117_01_tcg-decklist-graph-standard.md, lines 23-27 and 112.
type Diagnostic struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Role     string `json:"role,omitempty"`
	CardRef  string `json:"cardRef,omitempty"`
}
