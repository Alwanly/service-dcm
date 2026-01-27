package wrapper

type JSONResult struct {
	Code    int         `json:"-"`
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func ResponseSuccess(httpCode int, data interface{}) JSONResult {
	return JSONResult{
		Code:    httpCode,
		Success: true,
		Message: "Success",
		Data:    data,
	}
}

func ResponseFailed(httpCode int, message string, data interface{}) JSONResult {
	return JSONResult{
		Code:    httpCode,
		Success: false,
		Message: message,
		Data:    data,
	}
}
