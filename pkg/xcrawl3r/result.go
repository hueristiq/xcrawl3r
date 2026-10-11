package xcrawl3r

type Result struct {
	Type   ResultType
	Value  string
	Source string
	Via    string
	Error  error
}

type ResultType int

const (
	ResultURL ResultType = iota
	ResultError
	ResultWarning
)
