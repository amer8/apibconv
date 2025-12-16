package model

import "time"

// Reference represents a $ref reference object.
type Reference struct {
	Ref string // $ref value
}

// Metadata holds common metadata about the API or its components.
type Metadata struct {
	CreatedAt  time.Time
	ModifiedAt time.Time
	Author     string
	Version    string
}

// IsEmpty checks if the reference is empty
func (r *Reference) IsEmpty() bool {
	return r.Ref == ""
}

// Resolve returns the resolved reference string
func (r *Reference) Resolve(base string) string {
	return r.Ref
}
