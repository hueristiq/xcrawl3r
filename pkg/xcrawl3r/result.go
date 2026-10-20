package xcrawl3r

type ResultType int

type Result struct {
	Type   ResultType
	Source string
	Value  string
	Via    string
	Error  error
}

const (
	ResultURL ResultType = iota + 1
	ResultWarning
	ResultError
)
