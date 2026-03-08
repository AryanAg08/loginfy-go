package core

type SessionManager interface {
	CreateSession(userId string) (string, error)
	ValidateSession(ctx *Context, token string) (string, error)
	DestroySession(ctx *Context, token string) error
}
