package api

import "github.com/gin-gonic/gin"

type Response struct {
	Code    int `json:"code"`
	Message string `json:"message"`
	Data    any `json:"data"`
}

func writeSuccess(ctx *gin.Context, data any) {
	ctx.JSON(200, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

func writeError(ctx *gin.Context, httpStatus int, code int, message string) {
	ctx.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    gin.H{},
	})
}