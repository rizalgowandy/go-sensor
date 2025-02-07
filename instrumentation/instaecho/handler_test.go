// (c) Copyright IBM Corp. 2021
// (c) Copyright Instana Inc. 2016

// +build go1.15

package instaecho_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	instana "github.com/instana/go-sensor"
	"github.com/instana/go-sensor/instrumentation/instaecho"
	"github.com/instana/testify/assert"
	"github.com/instana/testify/require"
	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
)

func TestMain(m *testing.M) {
	instana.InitSensor(&instana.Options{
		Service: "echo-test",
		Tracer: instana.TracerOptions{
			CollectableHTTPHeaders: []string{"x-custom-header-1", "x-custom-header-2"},
		},
	})

	os.Exit(m.Run())
}

func TestPropagation(t *testing.T) {
	traceIDHeader := "0000000000001234"
	spanIDHeader := "0000000000004567"

	recorder := instana.NewTestRecorder()
	tracer := instana.NewTracerWithEverything(nil, recorder)

	sensor := instana.NewSensorWithTracer(tracer)

	engine := instaecho.New(sensor)
	engine.GET("/foo/:id", func(c echo.Context) error {

		parent, ok := instana.SpanFromContext(c.Request().Context())
		assert.True(t, ok)

		sp := parent.Tracer().StartSpan("sub-call", opentracing.ChildOf(parent.Context()))
		sp.Finish()

		c.Response().Header().Set("x-custom-header-2", "response")
		return c.JSON(200, []byte("{}"))
	}).Name = "foos"

	req := httptest.NewRequest("GET", "https://example.com/foo/1?SECRET_VALUE=%3Credacted%3E&myPassword=%3Credacted%3E&q=term&sensitive_key=%3Credacted%3E", nil)

	req.Header.Add(instana.FieldT, traceIDHeader)
	req.Header.Add(instana.FieldS, spanIDHeader)
	req.Header.Add(instana.FieldL, "1")
	req.Header.Set("X-Custom-Header-1", "request")

	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	// Response headers assertions
	assert.NotEmpty(t, w.Header().Get("X-Instana-T"))
	assert.NotEmpty(t, w.Header().Get("X-Instana-S"))
	assert.NotEmpty(t, w.Header().Get("X-Instana-L"))
	assert.NotEmpty(t, w.Header().Get("Traceparent"))
	assert.NotEmpty(t, w.Header().Get("Tracestate"))

	spans := recorder.GetQueuedSpans()
	require.Len(t, spans, 2)

	entrySpan, interSpan := spans[1], spans[0]

	assert.EqualValues(t, instana.EntrySpanKind, entrySpan.Kind)
	assert.EqualValues(t, instana.IntermediateSpanKind, interSpan.Kind)

	assert.Equal(t, entrySpan.TraceID, interSpan.TraceID)
	assert.Equal(t, entrySpan.SpanID, interSpan.ParentID)

	assert.Equal(t, traceIDHeader, instana.FormatID(entrySpan.TraceID))
	assert.Equal(t, spanIDHeader, instana.FormatID(entrySpan.ParentID))

	// ensure that entry span contains all necessary data
	require.IsType(t, instana.HTTPSpanData{}, entrySpan.Data)
	entrySpanData := entrySpan.Data.(instana.HTTPSpanData)

	assert.Equal(t, instana.HTTPSpanTags{
		Method:       "GET",
		Status:       http.StatusOK,
		Path:         "/foo/1",
		PathTemplate: "/foo/:id",
		URL:          "",
		Host:         "example.com",
		RouteID:      "foos",
		Protocol:     "https",
		Params:       "SECRET_VALUE=%3Credacted%3E&myPassword=%3Credacted%3E&q=term&sensitive_key=%3Credacted%3E",
		Headers: map[string]string{
			"x-custom-header-1": "request",
			"x-custom-header-2": "response",
		},
	}, entrySpanData.Tags)
}

func TestPropagationWithError(t *testing.T) {
	traceIDHeader := "0000000000001234"
	spanIDHeader := "0000000000004567"

	recorder := instana.NewTestRecorder()
	tracer := instana.NewTracerWithEverything(nil, recorder)

	sensor := instana.NewSensorWithTracer(tracer)

	engine := instaecho.New(sensor)
	engine.GET("/foo", func(c echo.Context) error {
		parent, ok := instana.SpanFromContext(c.Request().Context())
		assert.True(t, ok)

		sp := parent.Tracer().StartSpan("sub-call", opentracing.ChildOf(parent.Context()))
		sp.Finish()
		c.Response().Header().Set("x-custom-header-2", "response")

		return errors.New("MY-BAD-TEST-ERROR")
	}).Name = "foos"

	req := httptest.NewRequest("GET", "https://example.com/foo?SECRET_VALUE=%3Credacted%3E&myPassword=%3Credacted%3E&q=term&sensitive_key=%3Credacted%3E", nil)

	req.Header.Add(instana.FieldT, traceIDHeader)
	req.Header.Add(instana.FieldS, spanIDHeader)
	req.Header.Add(instana.FieldL, "1")
	req.Header.Set("X-Custom-Header-1", "request")

	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	// Response headers assertions
	assert.NotEmpty(t, w.Header().Get("X-Instana-T"))
	assert.NotEmpty(t, w.Header().Get("X-Instana-S"))
	assert.NotEmpty(t, w.Header().Get("X-Instana-L"))
	assert.NotEmpty(t, w.Header().Get("Traceparent"))
	assert.NotEmpty(t, w.Header().Get("Tracestate"))

	spans := recorder.GetQueuedSpans()
	require.Len(t, spans, 3)

	interSpan, entrySpan, logSpan := spans[0], spans[1], spans[2]

	assert.EqualValues(t, instana.EntrySpanKind, entrySpan.Kind)
	assert.EqualValues(t, instana.IntermediateSpanKind, interSpan.Kind)

	assert.Equal(t, entrySpan.TraceID, interSpan.TraceID)
	assert.Equal(t, entrySpan.SpanID, interSpan.ParentID)

	assert.Equal(t, traceIDHeader, instana.FormatID(entrySpan.TraceID))
	assert.Equal(t, spanIDHeader, instana.FormatID(entrySpan.ParentID))

	// ensure that entry span contains all necessary data
	require.IsType(t, instana.HTTPSpanData{}, entrySpan.Data)
	entrySpanData := entrySpan.Data.(instana.HTTPSpanData)

	assert.Equal(t, instana.HTTPSpanTags{
		Method:   "GET",
		Status:   http.StatusInternalServerError,
		Path:     "/foo",
		URL:      "",
		Host:     "example.com",
		RouteID:  "foos",
		Protocol: "https",
		Params:   "SECRET_VALUE=%3Credacted%3E&myPassword=%3Credacted%3E&q=term&sensitive_key=%3Credacted%3E",
		Headers: map[string]string{
			"x-custom-header-1": "request",
			"x-custom-header-2": "response",
		},
		Error: "Internal Server Error",
	}, entrySpanData.Tags)

	// assert that log message has been recorded within the span interval
	assert.GreaterOrEqual(t, logSpan.Timestamp, entrySpan.Timestamp)
	assert.LessOrEqual(t, logSpan.Duration, entrySpan.Duration)

	require.IsType(t, instana.LogSpanData{}, logSpan.Data)
	logData := logSpan.Data.(instana.LogSpanData)

	assert.Equal(t, instana.LogSpanTags{
		Level:   "ERROR",
		Message: `error: "Internal Server Error"`,
	}, logData.Tags)
}
