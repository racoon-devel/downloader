package task

type Status int

const (
	StatusConnecting Status = iota
	StatusActive
	StatusError
)
