# go-logger 由业务导向定义了两种日志格式

**使用方式见[example.go](./example/example.go)**

## Why?

最常见的日志打印方式就是一行字符串，比如"user login failed"，但是我们常常需要在日志记录内包含一些上下文信息，比如用户ID，比如"user login failed, user_id: 123"。

这种日志虽然在人工排查的时候可以提供一些帮助，但是有两个问题:

- 程序使用非常不方便，每个人有可能写出不同的形式，无法通用解析
- 程序员在输出的时候，需要考虑写法，比如用"user id: 123"还是"user_id 123"之类的，虽然很简单，但仍然带来了额外的抉择负担

因此采用json格式的日志输出是最简单的做法，`{"msg":"user login failed", "user_id":123}`，无论是程序解析还是日志输出时都很方便。

## How?

### 普通日志 (app.logs.v1)

即使我们把日志格式统一为json格式，最好统一遵循同样的json schema更方便解析和查询，因此我们约定统一的日志公共字段如下：

```
{
    schema: (string),   // schema 日志规格, 描述了它属于哪一种格式的日志, 既应该包含哪些字段
    service: (string),  // service 服务名称
    env: (string),      // environment 部署环境
    channel: (string),  // channel 日志类别
    level: (string),    // level 日志级别
    time: (string),     // time 日志时间, ISO8601
    msg: (string),      // message
    ctx: {...}          // context 自定义上下文数据
}
```

为减少日志字符串传输开销，公共字段都使用了字面缩写。

## HTTP Request日志规范 (http.request.v1)

### Why?

HTTP接口的请求日志比较特殊，无法套用上面的日志规范，原因是

- 没有Level
- 不需要Message
- 内容相对固定

### 字段定义

```
{
    schema: (string),
    service: (string),
    env: (string),
    time: (string),
    ip: (string),
    method: (string),
    path: (string),
    user: (string),
    headers: {
        (string): (any),
        ...
    },
    get: {
        (string): (any),
        ...
    },
    post: {
        (string): (any),
        ...
    },
    extra: {    // 扩展信息，e.g: runtime
        (string): (any),
        ...
    }
    error: {    // 错误信息
        msg: (string)
        stackTrace: [
            ...
        ]    
    }    
}
```
