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

	"github.com/mitchellh/pointerstructure"
)

// Config the plugin configuration.
type Config struct {
	Pointer string `json:"pointer"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// JSON2Redirect a Traefik plugin.
type JSON2Redirect struct {
	pointer *pointerstructure.Pointer
	next    http.Handler
}

// HTTPClient is a reduced interface for http.Client.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// New creates a new Json2Redirect plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	pointer, err := pointerstructure.Parse(config.Pointer)
	if err != nil {
		return nil, err
	}

	return &JSON2Redirect{pointer: pointer, next: next}, nil
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

	result, err := c.pointer.Get(jsonBody)
	if err != nil {
		rw.WriteHeader(http.StatusNotFound)
		_, _ = rw.Write([]byte(err.Error()))
		return
	}

	switch result.(type) {
	default:
		rw.WriteHeader(http.StatusNotFound)
		return
	case string:
	}

	redirectURL, err := url.Parse(result.(string))
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
