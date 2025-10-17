package mock

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type RoundTripper struct {
	RoundTripFunc      func(req *http.Request) *http.Response
	Err                error
	LastRequest        *http.Request
	CallCount          int
	CollectorCallCount int

	CollectorRequests []string
	RequestBodies     []string
}

func (m *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req), m.Err
}

func NewDefaultMockClient() *http.Client {
	return NewMockClient(nil, nil)
}

// NewMockClient creates a new http.Client with the mock RoundTripper.
func NewMockClient(roundTripFunc func(req *http.Request) *http.Response, err error) *http.Client {
	roundTripper := &RoundTripper{}
	if roundTripFunc == nil {
		roundTripFunc = roundTripper.DefaultRoundTripFunc
	}
	roundTripper.RoundTripFunc = roundTripFunc
	roundTripper.Err = err
	return &http.Client{
		Transport: roundTripper,
	}
}

// GetLastRequest returns the last request made by the mock client.
func (m *RoundTripper) GetLastRequest() *http.Request {
	return m.LastRequest
}

func (m *RoundTripper) DefaultRoundTripFunc(req *http.Request) *http.Response {
	//read req body and store it
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		m.RequestBodies = append(m.RequestBodies, string(bodyBytes))
	}

	if req.URL.Path == "/v1/data/collector" {
		m.CollectorCallCount++
		m.CollectorRequests = append(m.CollectorRequests, string(bodyBytes))
		return &http.Response{
			StatusCode: http.StatusOK,
		}
	}

	m.CallCount++
	mockPath := "./testdata/mock_responses/%s.json"
	flagName := strings.Replace(req.URL.Path, "/ofrep/v1/evaluate/flags/", "", -1)
	if flagName == "unauthorized" {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(bytes.NewReader([]byte(""))),
		}
	}

	content, err := os.ReadFile(fmt.Sprintf(mockPath, flagName))
	if err != nil {
		content, _ = os.ReadFile(fmt.Sprintf(mockPath, "flag_not_found"))
	}
	statusCode := http.StatusOK

	if strings.Contains(string(content), "errorCode") {
		statusCode = http.StatusBadRequest
	}
	if strings.Contains(string(content), "FLAG_NOT_FOUND") {
		statusCode = http.StatusNotFound
	}

	body := io.NopCloser(bytes.NewReader(content))
	return &http.Response{
		StatusCode: statusCode,
		Body:       body,
	}
}
