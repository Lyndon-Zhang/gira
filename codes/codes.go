package codes

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Logger interface {
	Error(...interface{})
}

var defaultLogger Logger

func SetLogger(l Logger) {
	defaultLogger = l
}

// 错误码常量
// 1-255 系统预留
const (
	CodeOk                             = 0
	CodeUnknown                        = 1
	CodeNullPointer                    = 2
	CodeNullObject                     = 3
	CodeInvalidSdkToken                = 4
	CodeInvalidArgs                    = 5
	CodeInvalidJwt                     = -20
	CodeJwtExpire                      = 7
	CodeTODO                           = 8
	CodeInvalidPassword                = 9
	CodePayOrderStatusInvalid          = 10
	CodeAccountPlatformNotSupport      = 11
	CodePayOrderAccountPlatformInvalid = 12
	CodePayOrderAmountInvalid          = 13
	CodeMax                            = 255
)
const (
	msg_Ok                             = "成功。"
	msg_Unknown                        = "未知错误。"
	msg_NullObject                     = "null object。"
	msg_InvalidSdkToken                = "无效的sdk token。"
	msg_InvalidArgs                    = "无效参数。"
	msg_InvalidJwt                     = "请重新登录。"
	msg_JwtExpire                      = "token已过期。"
	msg_TODO                           = "TODO。"
	msg_InvalidPassword                = "账号密码不对。"
	msg_PayOrderStatusInvalid          = "pay order status invalid"
	msg_AccountPlatformNotSupport      = "account platform not support"
	msg_PayOrderAccountPlatformInvalid = "pay order account platform invalid"
	msg_PayOrderAmountInvalid          = "pay order amount invalid"
)

var (
	ErrOk                             = New(CodeOk, msg_Ok)
	ErrUnknown                        = New(CodeUnknown, msg_Unknown)
	ErrNullObject                     = New(CodeNullObject, msg_NullObject)
	ErrInvalidSdkToken                = New(CodeInvalidSdkToken, msg_InvalidSdkToken)
	ErrInvalidArgs                    = New(CodeInvalidArgs, msg_InvalidArgs)
	ErrInvalidJwt                     = New(CodeInvalidJwt, msg_InvalidJwt)
	ErrJwtExpire                      = New(CodeJwtExpire, msg_JwtExpire)
	ErrTODO                           = New(CodeTODO, msg_TODO)
	ErrInvalidPassword                = New(CodeInvalidPassword, msg_InvalidPassword)
	ErrPayOrderStatusInvalid          = New(CodePayOrderStatusInvalid, msg_PayOrderStatusInvalid)
	ErrAccountPlatformNotSupport      = New(CodeAccountPlatformNotSupport, msg_AccountPlatformNotSupport)
	ErrPayOrderAccountPlatformInvalid = New(CodePayOrderAccountPlatformInvalid, msg_PayOrderAccountPlatformInvalid)
	ErrPayOrderAmountInvalid          = New(CodePayOrderAmountInvalid, msg_PayOrderAmountInvalid)
)

func Unwrap(err error) error {
	u, ok := err.(interface {
		Unwrap() error
	})
	if !ok {
		return nil
	}
	return u.Unwrap()
}

// from pkg "golang.org/x/xerrors"
func Is(err error, target error) bool {
	if target == nil {
		return err == target
	}
	isComparable := reflect.TypeOf(target).Comparable()
	for {
		if isComparable && err == target {
			return true
		}
		if x, ok := err.(interface{ Is(error) bool }); ok && x.Is(target) {
			return true
		}
		// TODO: consider supporting target.Is(err). This would allow
		// user-definable predicates, but also may allow for coping with sloppy
		// APIs, thereby making it easier to get away with them.
		if err = Unwrap(err); err == nil {
			return false
		}
	}
}

// TODO错误
func TraceErrTODO(values ...interface{}) *TraceError {
	return ErrTODO.TraceWithSkip(1, values...)
}

// wrap error
func Trace(err error, values ...interface{}) *TraceError {
	var kvs map[string]interface{}
	if len(values)%2 != 0 {
	} else if len(values) == 0 {
	} else {
		kvs = make(map[string]interface{})
		for i := 0; i < len(values); i += 2 {
			j := i + 1
			if k, ok := values[i].(string); ok {
				kvs[k] = values[j]
			}
		}
	}
	e := &TraceError{
		err:    err,
		values: kvs,
		stack:  takeStacktrace(1),
	}
	e.Print()
	return e
}

// 创建error_code
func New(code int32, msg string, values ...interface{}) *CodeError {
	var kvs map[string]interface{}
	if len(values)%2 != 0 {
	} else if len(values) == 0 {
	} else {
		kvs = make(map[string]interface{})

		for i := 0; i < len(values); i += 2 {
			j := i + 1
			if k, ok := values[i].(string); ok {
				kvs[k] = values[j]
			}
		}
	}
	return &CodeError{
		Code:   code,
		Msg:    msg,
		Values: kvs,
	}
}

// 提取错误码
func Code(err error) int32 {
	if err == nil {
		return 0
	}
	if e, ok := err.(*CodeError); ok {
		return e.Code
	}
	if e1, ok := err.(*TraceError); ok {
		if e2, ok := e1.err.(*CodeError); ok {
			return e2.Code
		}
	}
	s := strings.Split(err.Error(), ":")
	if len(s) == 0 {
		return CodeUnknown
	}
	i, err := strconv.Atoi(s[0])
	if err != nil {
		return CodeUnknown
	}
	return int32(i)
}

// 提取错误描述
func Msg(err error) string {
	if err == nil {
		return msg_Ok
	}
	if e, ok := err.(*CodeError); ok {
		return e.Msg
	}
	if e1, ok := err.(*TraceError); ok {
		if e2, ok := e1.err.(*CodeError); ok {
			return e2.Msg
		}
	}
	s := strings.Split(err.Error(), ":")
	if len(s) < 2 {
		return err.Error()
	}
	return s[1]
}

// 提取调用栈
func Stacktrace(err error) string {
	if err == nil {
		return ""
	}
	if e, ok := err.(*TraceError); ok {
		return string(e.stack)
	}
	return ""
}

type CodeError struct {
	Code   int32
	Msg    string
	Values map[string]interface{}
}

// 拷贝error并且附加stack
func (e *CodeError) Trace(values ...interface{}) *TraceError {
	return e.TraceWithSkip(1, values...)
}

func (e *CodeError) TraceWithSkip(skip int, values ...interface{}) *TraceError {
	var kvs map[string]interface{}
	if len(values)%2 != 0 {
	} else if len(values) == 0 {
	} else {
		kvs = make(map[string]interface{})
		for i := 0; i < len(values); i += 2 {
			j := i + 1
			if k, ok := values[i].(string); ok {
				kvs[k] = values[j]
			}
		}
	}
	err := &TraceError{err: e, stack: takeStacktrace(skip + 1), values: kvs}
	err.Print()
	return err
}

// 格式化错误字符串
func (e *CodeError) Error() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("%d: %s", e.Code, e.Msg))
	if e.Values != nil {
		for k, v := range e.Values {
			sb.WriteString(fmt.Sprintf("\n%s: %v", k, v))
		}
	}
	return sb.String()
}

func (e *CodeError) Is(err error) bool {
	if c, ok := err.(*CodeError); ok {
		return e.Code == c.Code
	}
	return e == err
}

// 跟踪错误信息
type TraceError struct {
	err    error
	stack  string
	values map[string]interface{}
}

func (e *TraceError) Print() {
	if defaultLogger != nil {
		defaultLogger.Error(e.Error())
	}
}

func (e *TraceError) Error() string {
	sb := strings.Builder{}
	sb.WriteString(e.err.Error())
	if e.values != nil {
		for k, v := range e.values {
			sb.WriteString(fmt.Sprintf("\n%s: %v", k, v))
		}
	}
	sb.WriteString("\n")
	sb.WriteString(e.stack)
	return sb.String()
}

func (e *TraceError) Unwrap() error {
	return e.err
}

func (e *TraceError) Is(err error) bool {
	return e == err
}
