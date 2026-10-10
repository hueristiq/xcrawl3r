package xcrawl3r

type Result struct {
	Type  ResultType
	Value string
	Error error
}

type ResultType int

const (
	ResultURL ResultType = iota
	ResultError
)
