package ch32fun

// Target is the initial hardware target declared by the specification.
type Target struct {
	Name string
}

// NewTarget creates a target descriptor for the CH32V003 board.
func NewTarget() *Target {
	return &Target{Name: "ch32v003"}
}
