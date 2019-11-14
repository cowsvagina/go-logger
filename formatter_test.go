package logger

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
)

func TestNewFormatter(t *testing.T) {
	_, err := NewFormatter(Standard("undefined"))
	if err == nil {
		t.Fatalf("Test NewFormatter(), Expected=%q, Actual=nil", ErrFormatterNotFound)
	}

	_, err = NewFormatter(APPLogsV1)
	if err != nil {
		t.Fatalf("Test NewFormatter(), Expected=nil, Actual=%q", err.Error())
	}

	_, err = NewFormatter(HTTPRequestV1)
	if err != nil {
		t.Fatalf("Test NewFormatter(), Expected=nil, Actual=%q", err.Error())
	}
}

func TestFormatterOutput(t *testing.T) {
	t.Run("APPLogsV1", func(t *testing.T) {
		t.Parallel()

		f := &APPLogsV1Formatter{}
		entry := &logrus.Entry{
			Level:   logrus.InfoLevel,
			Time:    time.Now(),
			Message: "hello world",
			Data: logrus.Fields{
				"foo": "bar",
			},
		}

		data, err := f.Format(entry)
		if err != nil {
			t.Fatalf("Format() error, Expected=nil, Actual=%q", err.Error())
		}

		if v := jsoniter.Get(data, "schema").ToString(); v == "" {
			t.Fatalf(`Format() output "schema", Expected=%q, Actual=%q`, APPLogsV1, v)
		} else if v := jsoniter.Get(data, "l").ToString(); v != entry.Level.String() {
			t.Fatalf(`Format() output "l", Expected=%q, Actual=%q`, entry.Level.String(), v)
		} else if v := jsoniter.Get(data, "m").ToString(); v != entry.Message {
			t.Fatalf(`Format() output "m", Expected=%q, Actual=%q`, entry.Message, v)
		} else if v := jsoniter.Get(data, "ctx", "foo").ToString(); v != "bar" {
			t.Fatalf(`Format() output "ctx.foo", Expected=%q, Actual=%q`, "bar", v)
		}
	})

	t.Run("HTTPRequestV1", func(t *testing.T) {
		t.Parallel()

		f := &HTTPRequestV1Formatter{}
		entry := &logrus.Entry{
			Time: time.Now(),
			Data: logrus.Fields{},
		}

		if _, err := f.Format(entry); err == nil {
			t.Fatal(`Format() error, Expected return error`)
		}

		entry.Data["request"] = ""
		if _, err := f.Format(entry); err == nil {
			t.Fatal(`Format() error, Expected return error`)
		}

		query := url.Values{}
		query.Set("foo", "bar")

		form := url.Values{}
		form.Set("foo", "baz")

		headers := http.Header{}
		headers.Set("x-test", "1")

		req := &http.Request{
			RemoteAddr: "1.2.3.4:1234",
			Header:     headers,
			Method:     http.MethodPost,
			URL: &url.URL{
				Path:     "/api",
				RawQuery: query.Encode(),
			},
			PostForm: form,
		}

		entry.Data["request"] = req
		entry.Data["user"] = 65535
		entry.Data["status"] = http.StatusAccepted

		data, err := f.Format(entry)
		if err != nil {
			t.Fatalf("Format() error, Expected=nil, Actual=%q", err.Error())
		}

		cases := []struct {
			path     []interface{}
			expected string
		}{
			{
				path:     []interface{}{"schema"},
				expected: string(HTTPRequestV1),
			},
			{
				path:     []interface{}{"ip"},
				expected: "1.2.3.4",
			},
			{
				path:     []interface{}{"method"},
				expected: req.Method,
			},
			{
				path:     []interface{}{"path"},
				expected: req.URL.Path,
			},
			{
				path:     []interface{}{"user"},
				expected: "65535",
			},
			{
				path:     []interface{}{"get", "foo"},
				expected: "bar",
			},
			{
				path:     []interface{}{"post", "foo"},
				expected: "baz",
			},
			{
				path:     []interface{}{"headers", "X-Test"},
				expected: "1",
			},
			{
				path:     []interface{}{"extra", "status"},
				expected: fmt.Sprintf("%d", http.StatusAccepted),
			},
		}

		for _, c := range cases {
			if v := jsoniter.Get(data, (c.path)...).ToString(); v != c.expected {
				t.Fatalf(`Format() output %q, Expecteded=%q, Actual=%q`, c.path, c.expected, v)
			}
		}
	})
}
