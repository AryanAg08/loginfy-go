package core

import "time"

// User represents an authenticated user in the system
type User struct {
	ID        string                 `json:"id" bson:"_id,omitempty"`
	Email     string                 `json:"email" bson:"email"`
	Password  string                 `json:"-" bson:"password"` // Never serialize password in JSON
	Roles     []string               `json:"roles" bson:"roles"`
	Metadata  map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" bson:"updated_at"`
}

// HasRole checks if the user has a specific role
func (u *User) HasRole(role string) bool {
	if u.Roles == nil {
		return false
	}
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the user has any of the specified roles
func (u *User) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if u.HasRole(role) {
			return true
		}
	}
	return false
}
