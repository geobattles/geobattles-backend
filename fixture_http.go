//go:build matchfixture

package main

import "net/http"

// A provider call is a test failure even if the caller could recover from a
// transport error. The production build keeps the standard HTTP transport.
type fixtureTransport struct{}

func (fixtureTransport) RoundTrip(*http.Request) (*http.Response, error) {
	panic("matchfixture: unexpected outbound HTTP request")
}

func init() { http.DefaultTransport = fixtureTransport{} }
