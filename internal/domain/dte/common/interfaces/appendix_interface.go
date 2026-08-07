package interfaces

// AppendixGetter is an interface that defines the getter methods that an appendix must implement
type AppendixGetter interface {
	GetField() string
	GetLabel() string
	GetValue() string
}

// AppendixSetter is an interface that defines the setter methods that an appendix must implement
type AppendixSetter interface {
	SetField(field string) error
	SetLabel(label string) error
	SetValue(value string) error
}

// Appendix is an interface that combines the getters and setters of Appendix
type Appendix interface {
	AppendixGetter
	AppendixSetter
}
