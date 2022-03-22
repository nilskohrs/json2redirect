// Package json2redirect is a plugin
package json2redirect

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"

	"k8s.io/client-go/util/jsonpath"
)

// Config the plugin configuration.
type Config struct {
	JSONPath string `json:"jsonPath"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// JSON2Redirect a Traefik plugin.
type JSON2Redirect struct {
	jsonPath *jsonpath.JSONPath
	next     http.Handler
}

// HTTPClient is a reduced interface for http.Client.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// New creates a new Json2Redirect plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	jsonPath := jsonpath.New(config.JSONPath)
	err := jsonPath.Parse(config.JSONPath)
	if err != nil {
		return nil, err
	}

	return &JSON2Redirect{jsonPath: jsonPath, next: next}, nil
}

func (c *JSON2Redirect) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	req.Header.Set("Accept-Encoding", "identity")
	wrappedWriter := &responseBuffer{
		ResponseWriter: rw,
	}

	c.next.ServeHTTP(wrappedWriter, req)

	bodyBytes := wrappedWriter.bodyBuffer.Bytes()

	contentEncoding := wrappedWriter.Header().Get("Content-Encoding")
	if contentEncoding != "" && contentEncoding != "identity" {
		if _, err := rw.Write(bodyBytes); err != nil {
			log.Printf("Content encoding not supported by : %v", err)
		}
		rw.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = rw.Write([]byte("Content encoding not supported"))
		return
	}

	jsonBody := interface{}(nil)
	err := json.Unmarshal(bodyBytes, &jsonBody)
	if err != nil {
		rw.WriteHeader(http.StatusUnsupportedMediaType)
		_, _ = rw.Write([]byte(err.Error()))
		return
	}

	jsonPathResult, err := c.jsonPath.FindResults(jsonBody)
	if err != nil {
		rw.WriteHeader(http.StatusNotFound)
		_, _ = rw.Write([]byte(err.Error()))
		return
	}

	if len(jsonPathResult) < 1 || !jsonPathResult[0][0].CanInterface() {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	redirectURL, err := url.Parse(jsonPathResult[0][0].Interface().(string))
	if err != nil {
		rw.WriteHeader(http.StatusNotFound)
		_, _ = rw.Write([]byte(err.Error()))
		return
	}

	rw.Header().Set("Location", redirectURL.String())
	rw.WriteHeader(http.StatusTemporaryRedirect)
}

type responseBuffer struct {
	bodyBuffer bytes.Buffer
	statusCode int

	http.ResponseWriter
}

func (r *responseBuffer) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

func (r *responseBuffer) Write(p []byte) (int, error) {
	if r.statusCode == 0 {
		r.WriteHeader(http.StatusOK)
	}

	return r.bodyBuffer.Write(p)
}

func (r *responseBuffer) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("%T is not a http.Hijacker", r.ResponseWriter)
	}

	return hijacker.Hijack()
}

func (r *responseBuffer) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
