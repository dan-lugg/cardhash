package standard

// ChangeScope identifies structural or treatment-inclusive comparison.
// Reference: 20261117_01_tcg-decklist-graph-standard.md, lines 163-175.
type ChangeScope string

const (
	// StructureScope ignores treatment fields.
	StructureScope ChangeScope = "structure"
	// TreatmentScope includes profile-recognized treatment fields.
	TreatmentScope ChangeScope = "treatment"
)

// ChangeItem is the standard's normalized signed quantity operation.
type ChangeItem struct {
	Role      string    `json:"role"`
	Quantity  int       `json:"quantity"`
	CardRef   string    `json:"cardRef"`
	Treatment Treatment `json:"treatment,omitempty"`
}

// ChangeSet is a materialized comparison result whose endpoint references are
// metadata and whose normalized items define its semantic hash.
type ChangeSet struct {
	Scope               ChangeScope  `json:"scope"`
	SourceStructureHash string       `json:"sourceStructureHash,omitempty"`
	TargetStructureHash string       `json:"targetStructureHash,omitempty"`
	SourceTreatmentHash string       `json:"sourceTreatmentHash,omitempty"`
	TargetTreatmentHash string       `json:"targetTreatmentHash,omitempty"`
	Items               []ChangeItem `json:"items"`
}
