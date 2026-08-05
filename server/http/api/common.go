package api

import (
	"github.com/gin-gonic/gin"
)

const (
	SUCCESS = 200
	ERROR   = 500
)

var MsgFlags = map[int]string{
	SUCCESS: "ok",
}

// GetMsg 获取状态码对应信息
func GetMsg(code int) string {
	msg, ok := MsgFlags[code]
	if ok {
		return msg
	}
	return MsgFlags[ERROR]
}

// Response 基础序列化器
type Response struct {
	Status int         `json:"status"`
	Data   interface{} `json:"data"`
	Msg    string      `json:"msg"`
	Error  string      `json:"error"`
}

// RespSuccess 带data成功返回
func RespSuccess(ctx *gin.Context, data interface{}, code ...int) *Response {
	status := SUCCESS
	if code != nil {
		status = code[0]
	}

	r := &Response{
		Status: status,
		Data:   data,
		Msg:    GetMsg(status),
	}

	return r
}

// RespError 错误返回
func RespError(ctx *gin.Context, err error, code ...int) *Response {
	status := ERROR
	if code != nil {
		status = code[0]
	}

	r := &Response{
		Status: status,
		Msg:    GetMsg(status),
		Data:   nil,
		Error:  err.Error(),
	}

	return r
}
