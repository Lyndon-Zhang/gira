package gins

import (
	"github.com/gin-gonic/gin"
	"github.com/Lyndon-Zhang/gira/codes"
)

type BaseJsonResponse struct {
	Code int32  `json:"code"`
	Msg  string `json:"msg"`
}

func (self *BaseJsonResponse) SetCode(v int32) {
	self.Code = v
}

func (self *BaseJsonResponse) SetMsg(v string) {
	self.Msg = v
}

type JsonResponse interface {
	SetCode(v int32)
	SetMsg(v string)
}

// 返回json response
func HttpJsonResponse(g *gin.Context, httpCode int, err error, data JsonResponse) {
	errorCode := codes.Code(err)
	errorMsg := codes.Msg(err)
	if data == nil {
		resp := BaseJsonResponse{
			Code: errorCode,
			Msg:  errorMsg,
		}
		g.JSON(httpCode, resp)
	} else {
		data.SetCode(errorCode)
		data.SetMsg(errorMsg)
		g.JSON(httpCode, data)
	}
}
