package xcrawl3r

type Result struct {
	Type   ResultType
	Source string
	Value  string
	Error  error
}

type ResultType int

const (
	ResultURL ResultType = iota
	ResultError
)
