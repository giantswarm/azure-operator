package ignition

type Ignition struct {
	Debug Debug
	Path  string
}

type Debug struct {
	Enabled    string
	LogsPrefix string
	LogsToken  string
}
