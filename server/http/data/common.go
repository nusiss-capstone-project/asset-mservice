package data

type BaseResponse struct {
	Code   int         `json:"code"`
	ErrMsg string      `json:"err_msg,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}
