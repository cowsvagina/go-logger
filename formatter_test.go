package logger

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
)

func TestNewFormatter(t *testing.T) {
	_, err := NewFormatter("undefined")
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
				"foo":           "bar",
				"channel":       "xxx",
				logrus.ErrorKey: errors.New("e"),
			},
		}

		cases := []struct {
			path     []interface{}
			expected string
		}{
			{
				path:     []interface{}{"schema"},
				expected: string(APPLogsV1),
			},
			{
				path:     []interface{}{"channel"},
				expected: "xxx",
			},
			{
				path:     []interface{}{"level"},
				expected: "info",
			},
			{
				path:     []interface{}{"msg"},
				expected: "hello world",
			},
			{
				path:     []interface{}{"ctx", "foo"},
				expected: "bar",
			},
			{ // channel应该被format为一个单独的字段
				path:     []interface{}{"ctx", "channel"},
				expected: "",
			},
			{
				path:     []interface{}{"ctx", logrus.ErrorKey, "msg"},
				expected: "e",
			},
			{
				path:     []interface{}{"ctx", logrus.ErrorKey, "trace"},
				expected: "[]",
			},
		}

		data, err := f.Format(entry)
		if err != nil {
			t.Fatalf("Format() error, Expected=nil, Actual=%q", err.Error())
		}

		// 两次Format是为了校验Format的幂等性
		data1, err := f.Format(entry)
		if err != nil {
			t.Fatalf("Format() error, Expected=nil, Actual=%q", err.Error())
		}

		for _, c := range cases {
			if v := jsoniter.Get(data, c.path...).ToString(); v != c.expected {
				t.Fatalf(`Format() output %q, Expecteded=%q, Actual=%q`, c.path, c.expected, v)
			}

			if v := jsoniter.Get(data1, c.path...).ToString(); v != c.expected {
				t.Fatalf(`Format() output %q, Expecteded=%q, Actual=%q`, c.path, c.expected, v)
			}
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

		entry.Data[HTTPRequestReqKey] = req
		entry.Data[HTTPRequestUserKey] = 65535
		entry.Data["status"] = http.StatusAccepted
		entry.Data[logrus.ErrorKey] = errors.New("ee")

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
				path:     []interface{}{HTTPRequestUserKey},
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
			{
				path:     []interface{}{"extra", logrus.ErrorKey, "msg"},
				expected: "ee",
			},
			{
				path:     []interface{}{"extra", logrus.ErrorKey, "trace"},
				expected: "[]",
			},
		}

		data, err := f.Format(entry)
		if err != nil {
			t.Fatalf("Format() error, Expected=nil, Actual=%q", err.Error())
		}

		// 两次Format是为了校验Format的幂等性
		data1, err := f.Format(entry)
		if err != nil {
			t.Fatalf("Format() error, Expected=nil, Actual=%q", err.Error())
		}

		for _, c := range cases {
			if v := jsoniter.Get(data, c.path...).ToString(); v != c.expected {
				t.Fatalf(`Format() output %q, Expecteded=%q, Actual=%q`, c.path, c.expected, v)
			}

			if v := jsoniter.Get(data1, c.path...).ToString(); v != c.expected {
				t.Fatalf(`Format() output %q, Expecteded=%q, Actual=%q`, c.path, c.expected, v)
			}
		}
	})
}
