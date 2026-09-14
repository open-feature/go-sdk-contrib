package testframework

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	testbed "github.com/open-feature/flagd-testbed/v3"
)

// The testbed assets - compose stack, Gherkin suites, certificate - come from the
// flagd-testbed Go module rather than from a git submodule, so that this framework
// can be consumed like any other module.
//
// Almost nothing has to reach the filesystem: the compose runner takes the stack as
// a reader, godog reads the suites from an fs.FS, and the flag definitions are baked
// into the testbed image. Only the certificate does, because the provider takes a
// path to it - so it is written out lazily, and only once per test binary.

// TestbedSource is where a runner reads the testbed assets from.
type TestbedSource struct {
	// dir is a local testbed checkout, empty when the embedded assets are used.
	dir string
}

// NewTestbedSource returns the assets of the flagd-testbed release this framework is
// built against, or those of a local testbed checkout when dir is not empty.
func NewTestbedSource(dir string) TestbedSource {
	return TestbedSource{dir: dir}
}

// Compose returns the compose stack.
func (s TestbedSource) Compose() (io.Reader, error) {
	if s.dir == "" {
		return bytes.NewReader(testbed.ComposeYAML()), nil
	}

	content, err := os.ReadFile(filepath.Join(s.dir, testbed.ComposeFileName))
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(content), nil
}

// Gherkin returns the Gherkin suites, to be handed to godog as Options.FS.
func (s TestbedSource) Gherkin() fs.FS {
	if s.dir == "" {
		return testbed.Gherkin()
	}

	return os.DirFS(filepath.Join(s.dir, "gherkin"))
}

// CACertPath returns the path of the certificate flagd is served with in the TLS
// configurations, writing it out first if it is only embedded.
func (s TestbedSource) CACertPath() (string, error) {
	if s.dir != "" {
		return filepath.Join(s.dir, filepath.FromSlash(testbed.CACertFile)), nil
	}

	return embeddedCACert()
}

// Version reports the flagd-testbed release the embedded assets come from.
func Version() string {
	return testbed.Version()
}

var embeddedCACert = sync.OnceValues(func() (string, error) {
	dir, err := os.MkdirTemp("", "flagd-testbed-*")
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, filepath.Base(testbed.CACertFile))
	content, err := fs.ReadFile(testbed.FS(), testbed.CACertFile)
	if err != nil {
		return "", err
	}

	return path, os.WriteFile(path, content, 0o600)
})
