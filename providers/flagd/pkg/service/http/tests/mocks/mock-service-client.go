package mocks

import (
	"io"
	"net/http"
	"testing"
)

type ServiceClient struct {
	RequestSetup ServiceClientMockRequestSetup

	Testing *testing.T
}

type ServiceClientMockRequestSetup struct {
	InMethod string
	InUrl    string
	InBody   io.Reader

	OutRes *http.Response
	OutErr error
}

func (s *ServiceClient) Request(method string, url string, body io.Reader) (*http.Response, error) {
	if method != s.RequestSetup.InMethod {
		s.Testing.Errorf("unexpected value for method received, expected %v got %v", s.RequestSetup.InMethod, url)
	}
	if url != s.RequestSetup.InUrl {
		s.Testing.Errorf("unexpected value for url received, expected %v got %v", s.RequestSetup.InUrl, url)
	}
	return s.RequestSetup.OutRes, s.RequestSetup.OutErr
}
