package status

import "net/http"

func StatusUnauthorized() int {
	return http.StatusUnauthorized
}

func StatusForbidden() int {
	return http.StatusForbidden
}

func StatusInternalServerError() int {
	return http.StatusInternalServerError
}

func StatusBadRequest() int {
	return http.StatusBadRequest
}

func StatusOK() int {
	return http.StatusOK
}

func StatusCreated() int {
	return http.StatusCreated
}

func StatusNoContent() int {
	return http.StatusNoContent
}
