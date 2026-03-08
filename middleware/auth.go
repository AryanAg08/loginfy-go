package middleware

import (
	"net/http"

	globalConstants "github.com/AryanAg08/loginfy.go/pkg/constants"
	globalStatus "github.com/AryanAg08/loginfy.go/pkg/status"
)

func RequireAuth() http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			token := r.Header.Get("Authorization")

			if token == "" {
				http.Error(w, globalConstants.AuthUnauthorized, globalStatus.StatusUnauthorized())
				return
			}

			next.ServeHTTP(w, r)
		})
}
