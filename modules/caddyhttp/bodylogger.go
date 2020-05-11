// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package caddyhttp

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	//"net/http/httputil"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(BodyLogger{})
}

// BodyLogger is a middleware for logging request and response body. Only for debug.
type BodyLogger struct {
	// LogRequest log request body when true
	LogRequest bool `json:"log_request,omitempty"`
	// LogResponse log response body when true
	LogResponse bool `json:"log_response,omitempty"`
	logger      *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (BodyLogger) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.body_logger",
		New: func() caddy.Module { return new(BodyLogger) },
	}
}

func (bl *BodyLogger) Provision(ctx caddy.Context) error {
	bl.logger = ctx.Logger(bl)
	return nil
}

func (bl BodyLogger) ServeHTTP(w http.ResponseWriter, r *http.Request, next Handler) error {
	logger := bl.logger.With(
		zap.Object("request", LoggableHTTPRequest{Request: r}),
	)

	if r.Body == nil {
		return next.ServeHTTP(w, r)
	}

	// dump, err := httputil.DumpRequest(r, true)
	// if err != nil {
	// 	logger.Error("Failed to read body", zap.String("err", err.Error()))
	// } else {
	// 	logger.Debug("request", zap.String("body", string(dump)))

	// }

	if bl.LogRequest {
		if body, err := ioutil.ReadAll(r.Body); err == nil {
			logger.Debug("request", zap.String("body", string(body)))
			r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		} else {
			logger.Error("Failed to read body", zap.String("err", err.Error()))
		}
	}

	if !bl.LogResponse {
		return next.ServeHTTP(w, r)
	}

	lrw := &loggingResponseWriter{
		ResponseWriter: w,
		buf:            &bytes.Buffer{},
	}

	err := next.ServeHTTP(lrw, r)
	if lrw.buf != nil {
		logger.Debug("response", zap.String("body", lrw.buf.String()))

		if _, err := io.Copy(w, lrw.buf); err != nil {
			logger.Error("Failed to write response", zap.String("err", err.Error()))
		}
	}

	return err
}

// Interface guard
var _ MiddlewareHandler = (*BodyLogger)(nil)

// loggingResponseWriter response writer with buffer for response body
type loggingResponseWriter struct {
	http.ResponseWriter
	buf *bytes.Buffer
}

func (lrw *loggingResponseWriter) Write(p []byte) (int, error) {
	return lrw.buf.Write(p)
}
