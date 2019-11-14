package logger

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// Standard 日志规范
type Standard string

const (
	// APPLogsV1 运行日志
	APPLogsV1 Standard = "app.logs.v1"
	// HTTPRequestV1 请求日志
	HTTPRequestV1 Standard = "http.request.v1"
)

var (
	// ErrFormatterNotFound 找不到对应规范的日志格式化对象
	ErrFormatterNotFound = fmt.Errorf("log formatter not found")
)

// NewLogger 创建新的日志对象
func NewLogger(s Standard) (*logrus.Logger, error) {
	f, err := NewFormatter(s)
	if err != nil {
		return nil, err
	}

	l := logrus.New()
	l.SetFormatter(f)
	return l, nil
}
