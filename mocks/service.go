package mocks

import (
	"net/http"
	"net/http/httptest"
)

// transportFunc is a function implementing the http.RoundTripper interface
type transportFunc func(r *http.Request) (*http.Response, error)

// RoundTrip implmements the http.RoundTripper interface by calling the
// underlying function
func (fn transportFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

// NewMockedService returns an *http.Client that calls the passed handler for
// all HTTP requests
func NewMockedService(handler http.Handler) *http.Client {
	return &http.Client{
		Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			return w.Result(), nil
		}),
	}
}
