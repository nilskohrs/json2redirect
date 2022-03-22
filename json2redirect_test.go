package json2redirect_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nilskohrs/json2redirect"
)

func TestShouldChangeHost(t *testing.T) {
	cfg := json2redirect.CreateConfig()

	cfg.JSONPath = "$.objects[0].redirect"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = rw.Write([]byte(`{
  "objects": [
    {"redirect":"https://url.to/redirect.to"}
  ]
}`))
	})

	handler, err := json2redirect.New(ctx, next, cfg, "json2redirect")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://internal.url/", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected responsecode %v, but actual value was: %v", http.StatusTemporaryRedirect, recorder.Code)
	}
	if recorder.Header().Get("Location") != "https://url.to/redirect.to" {
		t.Errorf("expected responsecode %v, but actual value was: %v", "https://url.to/redirect.to", recorder.Header().Get("Location"))
	}
}
