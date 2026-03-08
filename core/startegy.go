package core

type Strategy interface {
	Name() string
	Authenticate(ctx *Context) (*User, error)
}
