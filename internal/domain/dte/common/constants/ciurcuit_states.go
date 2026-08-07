package constants

type State int32

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)
