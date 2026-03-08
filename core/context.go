package core

import "net/http"

type Context struct {
	Request  *http.Request
	Response http.ResponseWriter
	Loginfy  *Loginfy
}
