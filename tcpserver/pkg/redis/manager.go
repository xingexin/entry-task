package redis

// Manager Redis统一管理器接口
type Manager interface {
	// GetClient 获取基础Redis客户端
	GetClient() Client

	// GetSession 获取Session管理器
	GetSession() SessionManager

	// GetLoginLimiter 获取登录限制器
	GetLoginLimiter() LoginLimiter

	// GetUserCache 获取用户缓存管理器
	GetUserCache() UserCache
}

// manager Redis统一管理器实现
type manager struct {
	client       Client
	session      SessionManager
	loginLimiter LoginLimiter
	userCache    UserCache
}

// NewManager 创建Redis管理器
func NewManager(client Client) Manager {
	return &manager{
		client:       client,
		session:      NewSessionManager(client),
		loginLimiter: NewLoginLimiter(client),
		userCache:    NewUserCache(client),
	}
}

// GetClient 获取基础Redis客户端
func (m *manager) GetClient() Client {
	return m.client
}

// GetSession 获取Session管理器
func (m *manager) GetSession() SessionManager {
	return m.session
}

// GetLoginLimiter 获取登录限制器
func (m *manager) GetLoginLimiter() LoginLimiter {
	return m.loginLimiter
}

// GetUserCache 获取用户缓存管理器
func (m *manager) GetUserCache() UserCache {
	return m.userCache
}
