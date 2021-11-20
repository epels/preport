package notifier_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/epels/preport/internal/testutil"
	"github.com/epels/preport/notifier"
)

func TestNewSlack(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		sc, err := notifier.NewSlack("https://example.com", "bearer")
		require.NoError(t, err)
		assert.NotNil(t, sc)
	})
	t.Run("Invalid baseURL", func(t *testing.T) {
		_, err := notifier.NewSlack("ftp://example.com", "bearer")
		require.Error(t, err)
	})
	t.Run("Empty baseURL", func(t *testing.T) {
		_, err := notifier.NewSlack("", "bearer")
		require.Error(t, err)
	})
	t.Run("Empty bearer", func(t *testing.T) {
		_, err := notifier.NewSlack("https://example.com", "")
		require.Error(t, err)
	})
}

func TestSlack(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		ts := testutil.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/chat.postMessage", r.URL.Path)
			assert.Equal(t, "Bearer super-secret", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json; charset=utf-8", r.Header.Get("Content-Type"))
			testutil.AssertTestdataJSONEquals(t, "testdata/ok_request.json", r.Body)

			testutil.WriteTestdata(t, "testdata/ok_response.json", w)
		})

		sc, err := notifier.NewSlack(ts.URL, "super-secret")
		require.NoError(t, err)
		err = sc.Notify(context.Background(), "general", "Just testing")
		require.NoError(t, err)
	})

	t.Run("Round trip failed", func(t *testing.T) {
		invalidBaseURL := "https://DF977BEA-4295-4758-AFF9-0EBCB1F509E2.fail"
		sc, err := notifier.NewSlack(invalidBaseURL, "super-secret")
		require.NoError(t, err)
		err = sc.Notify(context.Background(), "general", "Just testing")
		require.Error(t, err)
	})

	t.Run("Internal error", func(t *testing.T) {
		ts := testutil.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		sc, err := notifier.NewSlack(ts.URL, "super-secret")
		require.NoError(t, err)
		err = sc.Notify(context.Background(), "general", "Just testing")
		require.Error(t, err)
	})

	t.Run("Unexpected response", func(t *testing.T) {
		ts := testutil.NewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			testutil.WriteTestdata(t, "testdata/unexpected_response_response.json", w)
		})

		sc, err := notifier.NewSlack(ts.URL, "super-secret")
		require.NoError(t, err)
		err = sc.Notify(context.Background(), "general", "Just testing")
		require.Error(t, err)
	})
}
