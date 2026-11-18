package mtg

// MTG-specific constants remain in the game package so core contracts cannot
// accidentally inherit Magic roles, formats, treatment fields, or diagnostics.
// Reference: 20261117_01_tcg-decklist-graph-standard.md, lines 23-27 and 51-65.
const (
	// GameID identifies the Magic: The Gathering profile namespace.
	GameID = "mtg"
	// TreatmentProfileID identifies the baseline MTG treatment vocabulary.
	TreatmentProfileID = "mtg/treatment/1"
	// RoleSideboard is the MTG sideboard role defined by this profile.
	RoleSideboard = "side"
	// RoleCommanders is the MTG commander-zone role defined by this profile.
	RoleCommanders = "commanders"

	// FormatConstructed is the baseline MTG constructed format identifier.
	FormatConstructed = "constructed"
	// FormatCommander is the baseline MTG Commander format identifier.
	FormatCommander = "commander"

	// CodeMainCardCount identifies MTG main-list count violations.
	CodeMainCardCount = "main-card-count"
	// CodeSideboardTooLarge identifies an MTG sideboard-size violation.
	CodeSideboardTooLarge = "sideboard-too-large"
	// CodeInvalidCommanderCount identifies an MTG commander-count violation.
	CodeInvalidCommanderCount = "invalid-commander-count"
	// CodeCommanderTotal identifies an MTG combined commander total violation.
	CodeCommanderTotal = "commander-total"
	// CodeSingletonViolation identifies an MTG singleton violation.
	CodeSingletonViolation = "singleton-violation"

	// TreatmentSet is the MTG set/release treatment field.
	TreatmentSet = "set"
	// TreatmentCollectorNumber is the MTG collector-number treatment field.
	TreatmentCollectorNumber = "collectorNumber"
	// TreatmentLanguage is the MTG language treatment field.
	TreatmentLanguage = "language"
	// TreatmentFinish is the MTG finish treatment field.
	TreatmentFinish = "finish"
	// TreatmentFoil is the MTG foil treatment field.
	TreatmentFoil = "foil"
	// TreatmentBorder is the MTG border treatment field.
	TreatmentBorder = "border"
	// TreatmentFrame is the MTG frame treatment field.
	TreatmentFrame = "frame"
	// TreatmentEdition is the MTG edition treatment field.
	TreatmentEdition = "edition"
	// TreatmentArtwork is the MTG artwork treatment field.
	TreatmentArtwork = "artwork"
	// TreatmentReskin is the MTG reskin treatment field.
	TreatmentReskin = "reskin"
)
