package status

import "net/http"

func StatusUnauthorized() int {
	return http.StatusUnauthorized
}
