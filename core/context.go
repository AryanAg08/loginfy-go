package core

import "net/http"

const (
	userContextKey = "__loginfy_user__"
)

// Context holds the request context for authentication operations
type Context struct {
	Request   *http.Request
	Response  http.ResponseWriter
	Loginfy   *Loginfy
	RequestID string // unique identifier for this request
	data      map[string]interface{}
}

// GetString retrieves a string value from the context data
func (c *Context) GetString(key string) string {
	if c.data == nil {
		return ""
	}
	if val, ok := c.data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// Set stores a value in the context data
func (c *Context) Set(key string, value interface{}) {
	if c.data == nil {
		c.data = make(map[string]interface{})
	}
	c.data[key] = value
}

// Get retrieves a value from the context data
func (c *Context) Get(key string) (interface{}, bool) {
	if c.data == nil {
		return nil, false
	}
	val, ok := c.data[key]
	return val, ok
}

// SetUser stores the authenticated user in the context
func (c *Context) SetUser(user *User) {
	c.Set(userContextKey, user)
}

// GetUser retrieves the authenticated user from the context
func (c *Context) GetUser() (*User, bool) {
	val, ok := c.Get(userContextKey)
	if !ok {
		return nil, false
	}
	user, ok := val.(*User)
	return user, ok
}

// HasUser checks if an authenticated user exists in the context
func (c *Context) HasUser() bool {
	_, ok := c.GetUser()
	return ok
}
