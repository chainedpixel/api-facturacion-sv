package event

type PersistableEvent interface {
	Event
	UserID() uint
	BranchID() uint
}
