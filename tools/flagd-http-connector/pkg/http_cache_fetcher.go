package flagdhttpconnector

import (
	"io"
	"log"
	"net/http"
)

// HttpCacheFetcher fetches HTTP content with ETag/Last-Modified caching.
// Not thread-safe.
type HttpCacheFetcher struct {
	cachedETag         string
	cachedLastModified string
}

// FetchContent fetches content using HTTP GET and applies ETag/Last-Modified caching headers.
// It updates cached headers if a 200 OK response is received.
func (f *HttpCacheFetcher) FetchContent(client *http.Client, req *http.Request) (*http.Response, string, error) {
	// Clone the request to avoid modifying the original
	reqCopy := req.Clone(req.Context())

	if f.cachedETag != "" {
		reqCopy.Header.Set("If-None-Match", f.cachedETag)
	}
	if f.cachedLastModified != "" {
		reqCopy.Header.Set("If-Modified-Since", f.cachedLastModified)
	}

	resp, err := client.Do(reqCopy)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		// Only drain if body is not nil and status is not 200
		if resp.StatusCode != http.StatusOK && resp.Body != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		f.cachedETag = resp.Header.Get("ETag")
		f.cachedLastModified = resp.Header.Get("Last-Modified")
		log.Println("[DEBUG] fetched new content")

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return resp, "", err
		}
		return resp, string(bodyBytes), nil

	case http.StatusNotModified:
		log.Println("[DEBUG] got 304 Not Modified")
		return resp, "", nil

	default:
		return resp, "", nil
	}
}
