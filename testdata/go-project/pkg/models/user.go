package models

import "fmt"

// User represents a user in the system.
type User struct {
	ID    string
	Name  string
	Email string
}

// UserFilter defines criteria for filtering users.
type UserFilter struct {
	NamePrefix string
	Limit      int
}

// Validatable is implemented by types that can validate themselves.
type Validatable interface {
	Validate() error
}

// Validate checks if the user data is valid.
func (u *User) Validate() error {
	if u.Name == "" {
		return ErrEmptyName
	}
	return nil
}

// FormatName returns the user's display name.
func FormatName(u *User) string {
	if u == nil {
		return ""
	}
	return u.Name
}

var ErrEmptyName = fmt.Errorf("name cannot be empty")
