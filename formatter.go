package logger

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	HTTPRequestReqKey  = "request"
	HTTPRequestUserKey = "user"

	ChannelKey = "channel"
)

var (
	// MaxStackTrace 记录的错误信息的调用栈最大深度
	MaxStackTrace = 10

	_ logrus.Formatter = (*APPLogsV1Formatter)(nil)
	_ logrus.Formatter = (*HTTPRequestV1Formatter)(nil)

	emptyStack = make([]string, 0)

	appLogsV1Pool = sync.Pool{
		New: func() interface{} {
			return &APPLogsV1Data{
				Schema: string(APPLogsV1),
			}
		},
	}

	httpRequestV1Pool = sync.Pool{
		New: func() interface{} {
			return &HTTPRequestV1Data{
				Schema: string(HTTPRequestV1),
			}
		},
	}
)

// NewFormatter 获得日志规范对应的格式化对象
func NewFormatter(s Standard) (logrus.Formatter, error) {
	switch s {
	case APPLogsV1:
		return &APPLogsV1Formatter{
			TimeLayout: time.RFC3339,
		}, nil
	case HTTPRequestV1:
		return &HTTPRequestV1Formatter{
			TimeLayout: time.RFC3339,
		}, nil
	}

	return nil, errors.Wrapf(ErrFormatterNotFound, "log standard %q", s)
}

// APPLogsV1Data app.logs.v1日志输出内容
type APPLogsV1Data struct {
	Schema      string                 `json:"schema"`
	Service     string                 `json:"service,omitempty"`
	Environment string                 `json:"env,omitempty"`
	Channel     string                 `json:"channel"`
	Level       string                 `json:"level"`
	Time        string                 `json:"time"`
	Message     string                 `json:"msg"`
	Context     map[string]interface{} `json:"ctx,omitempty"`
}

// APPLogsV1Formatter app.logs.v1日志格式化
type APPLogsV1Formatter struct {
	// 时间格式，默认ISO8601，精确到秒
	TimeLayout  string
	Service     string
	Environment string
}

// Format implements logrus.Formatter interface
func (af *APPLogsV1Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	channel := ""
	context := logrus.Fields{}
	for k, v := range entry.Data {
		switch k {
		case ChannelKey:
			channel, _ = v.(string)
		default:
			if err, ok := v.(error); ok {
				context[k] = makeErrInfo(err)
				continue
			}
			context[k] = v
		}
	}

	data := appLogsV1Pool.Get().(*APPLogsV1Data)
	data.Time = entry.Time.Format(af.TimeLayout)
	data.Level = entry.Level.String()
	data.Service = af.Service
	data.Channel = channel
	data.Environment = af.Environment
	data.Message = entry.Message
	data.Context = context

	output, err := jsoniter.Marshal(data)
	if err != nil {
		appLogsV1Pool.Put(data)
		return nil, errors.Wrapf(err, "json encode %s log", APPLogsV1)
	}

	appLogsV1Pool.Put(data)
	return append(output, '\n'), nil
}

// HTTPRequestV1Data http.request.v1日志输出内容
type HTTPRequestV1Data struct {
	Schema      string            `json:"schema"`
	Service     string            `json:"service,omitempty"`
	Environment string            `json:"env,omitempty"`
	Level       string            `json:"level"`
	Time        string            `json:"time"`
	IP          string            `json:"ip"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	User        string            `json:"user,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Get         logrus.Fields     `json:"get,omitempty"`
	Post        logrus.Fields     `json:"post,omitempty"`
	Extra       logrus.Fields     `json:"extra,omitempty"`
}

// HTTPRequestV1Formatter http.request.v1日志格式化
type HTTPRequestV1Formatter struct {
	// 时间格式，默认ISO8601，精确到秒
	TimeLayout  string
	Service     string
	Environment string
}

// Format implements logrus.Formatter interface
func (hf *HTTPRequestV1Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	rv, ok := entry.Data[HTTPRequestReqKey]
	if !ok {
		return nil, errors.New(`require "request"`)
	}

	req, ok := rv.(*http.Request)
	if !ok {
		return nil, errors.Errorf(`"request" type MUST be *http.Request, got %s`, reflect.TypeOf(rv).String())
	}

	uid := ""
	extra := logrus.Fields{}
	for k, v := range entry.Data {
		switch k {
		case HTTPRequestReqKey:
			continue
		case HTTPRequestUserKey:
			uid = fmt.Sprintf("%v", v)
		default:
			if err, ok := v.(error); ok {
				extra[k] = makeErrInfo(err)
				continue
			}
			extra[k] = v
		}
	}

	data := httpRequestV1Pool.Get().(*HTTPRequestV1Data)
	data.Service = hf.Service
	data.Environment = hf.Environment
	data.Level = entry.Level.String()
	data.Time = entry.Time.Format(hf.TimeLayout)
	data.IP = strings.Split(req.RemoteAddr, ":")[0]
	data.Method = req.Method
	data.Path = req.URL.Path
	data.User = uid
	data.Headers = map[string]string{}
	data.Get = logrus.Fields{}
	data.Post = logrus.Fields{}
	data.Extra = extra

	for k, v := range req.Header {
		if len(v) > 1 {
			data.Headers[k] = strings.Join(v, ", ")
		} else {
			data.Headers[k] = v[0]
		}
	}

	if q := req.URL.Query(); len(q) > 0 {
		for k, v := range q {
			if len(v) > 1 {
				data.Get[k] = v
			} else {
				data.Get[k] = v[0]
			}
		}
	}

	if vals := req.PostForm; len(vals) > 0 {
		for k, v := range vals {
			if len(v) > 1 {
				data.Post[k] = v
			} else {
				data.Post[k] = v[0]
			}
		}
	}

	output, err := jsoniter.Marshal(data)
	if err != nil {
		httpRequestV1Pool.Put(data)
		return nil, errors.Wrapf(err, "json encode %s log", HTTPRequestV1)
	}

	httpRequestV1Pool.Put(data)
	return append(output, '\n'), nil
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

// stackTrace 从错误信息中获取调用栈信息
func stackTrace(err error) []string {
	if err, ok := err.(stackTracer); ok {
		return strings.Split(
			strings.ReplaceAll(
				strings.TrimLeft(
					fmt.Sprintf("%+v", err.StackTrace()),
					"\n",
				),
				"\n\t",
				" ",
			),
			"\n",
		)
	}

	return emptyStack
}

func makeErrInfo(err error) logrus.Fields {
	errInfo := logrus.Fields{}
	trace := make([]string, 0)
	if st := stackTrace(err); len(st) > 0 {
		if len(st) >= MaxStackTrace {
			st = st[:MaxStackTrace]
		}
		trace = st
	}
	errInfo["msg"] = err.Error()
	errInfo["trace"] = trace

	return errInfo
}
