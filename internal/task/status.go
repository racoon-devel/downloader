package task

// Status represents Task's internal state
type Status int

const (
	StatusConnecting Status = iota
	StatusActive
	StatusError
)
