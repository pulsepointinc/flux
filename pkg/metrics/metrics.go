package metrics

/*
Labels and so on for metrics used in Flux.
*/

const (
	LabelRoute   = "route"
	LabelMethod  = "method"
	LabelSuccess = "success"
	LabelName    = "name"

	// Labels for release metrics
	LabelAction      = "action"
	LabelReleaseType = "release_type"
	LabelReleaseKind = "release_kind"
	LabelStage       = "stage"
)
