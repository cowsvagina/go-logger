package main

import (
	"net/http"
	"net/url"
	"os"

	"github.com/cowsvagina/go-logger"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func main() {
	// 设置错误信息内调用栈的最大记录深度，默认:10
	logger.MaxStackTrace = 20

	appLogsV1Example()
	httpRequestV1Example()
}

func appLogsV1Example() {
	// 创建app.logs.v1规范的日志对象
	al, err := logger.NewLogger(logger.APPLogsV1)
	if err != nil {
		panic(err)
	}

	// al 是标准的logrus.Logger
	al.SetLevel(logrus.DebugLevel)
	al.SetOutput(os.Stdout)

	// 修改输出日期格式，默认time.RFC3339
	// if f, ok := al.Formatter.(*logger.APPLogsV1Formatter); ok {
	// 	f.TimeLayout = time.RFC3339Nano
	// }

	// OUTPUT: {"schema":"app.logs.v1","t":"2019-08-12T10:13:48.837899+08:00","l":"debug","c":"TEST","m":"test app.logs.v1 log","ctx":{"foo":"bar","error":"wow","stackTrace":["main.main /home/hsldymq/Development/Go/src/github.com/cowsvagina/go-logger/example/example.go:37","runtime.main /usr/local/opt/go/libexec/src/runtime/proc.go:200","runtime.goexit /usr/local/opt/go/libexec/src/runtime/asm_amd64.s:1337"]}}
	al.WithFields(logrus.Fields{
		"channel": "TEST",
		"foo":     "bar",
	}).
		WithError(errors.New("wow")).
		Debugf("test %s log", logger.APPLogsV1)
}

func httpRequestV1Example() {
	// 创建 http.request.v1规范的日志对象
	hl, err := logger.NewLogger(logger.HTTPRequestV1)
	if err != nil {
		panic(err)
	}

	// 修改输出日期格式，默认time.RFC3339
	// if f, ok := hl.Formatter.(*logger.HTTPRequestV1Formatter); ok {
	// 	f.TimeLayout = time.RFC3339Nano
	// }

	headers := http.Header{}
	headers.Set("x-test", "1")

	query := url.Values{}
	query.Set("foo", "bar")

	req := &http.Request{
		Header:     headers,
		RemoteAddr: "1.2.3.4:1234",
		Method:     http.MethodGet,
		URL:        &url.URL{Path: "/test", RawQuery: query.Encode()},
	}

	// OUTPUT: {"schema":"http.request.v1","time":"2019-08-12T10:13:48.838294+08:00","ip":"1.2.3.4","method":"GET","path":"/test","user":"123","headers":{"X-Test":"1"},"get":{"foo":"bar"},"extra":{"status":404}}
	hl.WithFields(logrus.Fields{
		"request": req,
		"status":  http.StatusNotFound,
		"user":    123,
	}).Info("")
}
