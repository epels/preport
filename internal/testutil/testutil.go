package testutil

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// AssertTestdataJSONEquals asserts the testdata at path is equal to the JSON
// data in r, ignoring any whitespace and formatting.
func AssertTestdataJSONEquals(t *testing.T, path string, r io.Reader) {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Unable to open testdata file: os: Open: %s", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Logf("Unable to close testdata file: os: File.Close: %s", err)
		}
	}()

	exp, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("Unable to read testdata file: io/ioutil: ReadAll: %s", err)
	}
	actual, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("Unable to read from reader: io/ioutil: ReadAll: %s", err)
	}

	assert.JSONEq(t, string(exp), string(actual))
}

// WriteTestdata copies the file content at path to w, and fails the test on
// error.
func WriteTestdata(t *testing.T, path string, w io.Writer) {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Unable to open testdata file: os: Open: %s", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Logf("Unable to close testdata file: os: File.Close: %s", err)
		}
	}()

	if _, err := io.Copy(w, f); err != nil {
		t.Fatalf("Unable to copy testdata from file to writer: io: Copy: %s", err)
	}
}

// NewTestServer is a light wrapper around httptest.NewServer that closes the
// server when the test finishes and omits the need for casting to
// http.HandlerFunc.
func NewTestServer(t *testing.T, hf http.HandlerFunc) *httptest.Server {
	t.Helper()

	ts := httptest.NewServer(hf)
	t.Cleanup(ts.Close)

	return ts
}
