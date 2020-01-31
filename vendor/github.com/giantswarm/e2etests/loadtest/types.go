package loadtest

// LoadTestResults parses results from the Storm Forger API.
type LoadTestResults struct {
	Data LoadTestResultsData `json:"data"`
}

type LoadTestResultsData struct {
	Attributes LoadTestResultsDataAttributes `json:"attributes"`
}

type LoadTestResultsDataAttributes struct {
	BasicStatistics LoadTestResultsDataAttributesBasicStatistics `json:"basic_statistics"`
}

type LoadTestResultsDataAttributesBasicStatistics struct {
	Apdex75 float32 `json:"apdex_75"`
}
