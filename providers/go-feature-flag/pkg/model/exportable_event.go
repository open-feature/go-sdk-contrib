package model

type ExportableEvent interface {
	// ToMap returns the event as a map of strings to any.
	ToMap() (map[string]any, error)
}
