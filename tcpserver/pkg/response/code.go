package response

const (
	// 成功
	CodeSuccess = 0

	// 客户端错误 (400xx)
	CodeBadRequest          = 40000
	CodeInvalidParams       = 40001
	CodeInvalidFormat       = 40002
	CodeInvalidNickname     = 40003
	CodeInvalidFile         = 40004
	CodeFileTooLarge        = 40005
	CodeUnsupportedFileType = 40006

	// 认证错误 (401xx)
	CodeUnauthorized             = 40100
	CodeInvalidToken             = 40101
	CodeTokenExpired             = 40102
	CodeInvalidAccountOrPassword = 40103

	// 权限错误 (403xx)
	CodeForbidden    = 40300
	CodeAccessDenied = 40301

	// 资源错误 (404xx)
	CodeNotFound     = 40400
	CodeUserNotFound = 40401

	// 业务冲突 (409xx)
	CodeConflict       = 40900
	CodeUserExists     = 40901
	CodeUsernameExists = 40902

	// 服务端错误 (500xx)
	CodeInternalServerError = 50000
	CodeDatabaseError       = 50001
	CodeRPCError            = 50002
	CodeRedisError          = 50003
	CodeServiceUnavailable  = 50004
)

var messages = map[int32]string{
	CodeSuccess:                  "OK",
	CodeBadRequest:               "请求参数错误",
	CodeInvalidParams:            "参数验证失败",
	CodeInvalidFormat:            "格式错误",
	CodeInvalidNickname:          "昵称无效",
	CodeInvalidFile:              "无效文件",
	CodeFileTooLarge:             "文件过大",
	CodeUnsupportedFileType:      "不支持的文件类型",
	CodeUnauthorized:             "未认证",
	CodeInvalidToken:             "Token无效",
	CodeTokenExpired:             "Token已过期",
	CodeInvalidAccountOrPassword: "用户名或密码错误",
	CodeForbidden:                "无权限",
	CodeAccessDenied:             "访问被拒绝",
	CodeNotFound:                 "资源不存在",
	CodeUserNotFound:             "用户不存在",
	CodeConflict:                 "资源冲突",
	CodeUserExists:               "用户已存在",
	CodeUsernameExists:           "用户名已存在",
	CodeInternalServerError:      "服务器内部错误",
	CodeDatabaseError:            "数据库错误",
	CodeRPCError:                 "RPC调用错误",
	CodeRedisError:               "Redis错误",
	CodeServiceUnavailable:       "服务不可用",
}

func GetMessage(code int32) string {
	if msg, ok := messages[code]; ok {
		return msg
	}
	return "未知错误"
}
