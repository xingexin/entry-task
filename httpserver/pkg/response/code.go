package response

// 业务错误码定义
const (
	// 成功
	CodeSuccess = 0

	// 客户端错误 (400-499)
	CodeBadRequest    = 40000 // 请求参数错误
	CodeInvalidParams = 40001 // 参数验证失败
	CodeInvalidFormat = 40002 // 格式错误

	// 认证错误 (401xx)
	CodeUnauthorized       = 40100 // 未认证
	CodeInvalidToken       = 40101 // Token无效
	CodeTokenExpired       = 40102 // Token过期
	CodeInvalidCredentials = 40103 // 用户名或密码错误

	// 权限错误 (403xx)
	CodeForbidden    = 40300 // 无权限
	CodeAccessDenied = 40301 // 访问被拒绝

	// 资源错误 (404xx)
	CodeNotFound     = 40400 // 资源不存在
	CodeUserNotFound = 40401 // 用户不存在

	// 业务错误 (409xx)
	CodeConflict       = 40900 // 资源冲突
	CodeUserExists     = 40901 // 用户已存在
	CodeUsernameExists = 40902 // 用户名已存在

	// 服务端错误 (500-599)
	CodeInternalServerError = 50000 // 服务器内部错误
	CodeDatabaseError       = 50001 // 数据库错误
	CodeRPCError            = 50002 // RPC调用错误
	CodeRedisError          = 50003 // Redis错误
	CodeServiceUnavailable  = 50004 // 服务不可用
)

// 错误信息映射
var CodeMessage = map[int]string{
	CodeSuccess: "OK",

	// 客户端错误
	CodeBadRequest:    "请求参数错误",
	CodeInvalidParams: "参数验证失败",
	CodeInvalidFormat: "格式错误",

	// 认证错误
	CodeUnauthorized:       "未认证",
	CodeInvalidToken:       "Token无效",
	CodeTokenExpired:       "Token已过期",
	CodeInvalidCredentials: "用户名或密码错误",

	// 权限错误
	CodeForbidden:    "无权限",
	CodeAccessDenied: "访问被拒绝",

	// 资源错误
	CodeNotFound:     "资源不存在",
	CodeUserNotFound: "用户不存在",

	// 业务错误
	CodeConflict:       "资源冲突",
	CodeUserExists:     "用户已存在",
	CodeUsernameExists: "用户名已存在",

	// 服务端错误
	CodeInternalServerError: "服务器内部错误",
	CodeDatabaseError:       "数据库错误",
	CodeRPCError:            "RPC调用错误",
	CodeRedisError:          "Redis错误",
	CodeServiceUnavailable:  "服务不可用",
}

// GetMessage 获取错误码对应的消息
func GetMessage(code int) string {
	if msg, ok := CodeMessage[code]; ok {
		return msg
	}
	return "未知错误"
}
