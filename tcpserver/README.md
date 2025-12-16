# TCP Server (gRPC)

## 项目架构

```
tcpserver/
├── cmd/
│   └── tcpserver/
│       └── main.go              # 主程序入口
├── internal/
│   ├── dto/                     # 数据传输对象
│   │   ├── user_dto.go
│   │   ├── auth_dto.go
│   │   ├── converter.go
│   │   └── validator.go
│   ├── middleware/              # gRPC 拦截器
│   │   └── grpc_interceptor.go
│   ├── model/                   # 数据库模型
│   │   └── user_model.go
│   ├── repository/              # 数据访问层
│   │   └── user_repository.go
│   ├── rpchandler/              # gRPC Handler
│   │   └── user_handler.go
│   └── service/                 # 业务逻辑层
│       └── user_service.go
├── pkg/
│   ├── container/               # 依赖注入容器
│   ├── db/                      # 数据库工具
│   ├── logger/                  # 日志工具
│   └── redis/                   # Redis 工具
└── config/
    ├── config.go
    └── config.yaml              # 配置文件
```

## 功能特性

### ✅ 已实现功能

1. **gRPC 服务**
   - 用户登录 (`Login`)
   - 用户登出 (`Logout`)
   - 获取用户信息 (`GetProfile`)
   - 更新昵称 (`UpdateNickname`)
   - 更新头像 (`UpdateProfilePicture`)

2. **中间件（拦截器）**
   - **Panic 恢复**：捕获程序崩溃，返回友好错误
   - **日志记录**：记录所有 RPC 请求和响应
   - **鉴权验证**：基于 Redis Session Token 的认证
   - **性能监控**：记录每个 RPC 的执行时间

3. **数据缓存**
   - **用户缓存**：优先从 Redis 读取用户信息
   - **负缓存**：防止缓存穿透
   - **延迟双删**：保证缓存一致性
   - **缓存降级**：Redis 故障不影响核心业务

4. **安全机制**
   - **Session Token**：基于 Redis 的会话管理
   - **登录限流**：防止暴力破解（5次失败限制）
   - **密码加密**：bcrypt 哈希存储
   - **白名单机制**：公开接口无需鉴权

## 启动步骤

### 1. 配置环境

确保以下服务已启动：
- MySQL 5.5+
- Redis

### 2. 修改配置文件

编辑 `config/config.yaml`：

```yaml
server:
  host: "0.0.0.0"
  tcp_port: 50051        # gRPC 监听端口

database:
  host: "localhost"      # MySQL 地址
  port: 3306
  username: "root"
  password: "your_password"
  database: "entrytask"

redis:
  host: "localhost"      # Redis 地址
  port: 6379
```

### 3. 创建数据库表

```sql
CREATE DATABASE entrytask CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE entrytask;

CREATE TABLE users (
    id BIGINT UNSIGNED PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    nickname VARCHAR(50) NOT NULL,
    profile_picture VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### 4. 启动 TCP Server

```bash
# 在项目根目录执行
cd tcpserver
go run cmd/tcpserver/main.go -config config/config.yaml
```

启动成功后，会看到：

```
INFO  TCP Server 启动中...
INFO  配置加载成功
INFO  依赖注入容器初始化成功
INFO  RedisManager 初始化成功
INFO  gRPC Server 创建成功，拦截器链已注册
INFO  gRPC 服务注册成功
INFO  TCP Server 启动成功  addr=0.0.0.0:50051
```

## gRPC 拦截器执行顺序

```
客户端请求
    ↓
RecoveryInterceptor     ← 第1层：捕获 Panic
    ↓
LoggingInterceptor      ← 第2层：记录日志
    ↓
AuthInterceptor         ← 第3层：验证 Token
    ↓
MetricsInterceptor      ← 第4层：性能监控
    ↓
Handler (业务逻辑)
    ↓
响应返回（反向经过所有拦截器）
```

## 鉴权机制

### 白名单（不需要 Token）

- `/user.UserService/Login` - 登录接口

### 受保护接口（需要 Token）

- `/user.UserService/Logout`
- `/user.UserService/GetProfile`
- `/user.UserService/UpdateNickname`
- `/user.UserService/UpdateProfilePicture`

### Token 传递方式

客户端需要在 gRPC metadata 中传递 `authorization` 字段：

```go
ctx := metadata.NewOutgoingContext(context.Background(),
    metadata.Pairs("authorization", "your-session-token"),
)
```

## 测试示例

### 使用 grpcurl 测试

```bash
# 1. 登录
grpcurl -plaintext -d '{
  "username": "user00000001",
  "password": "P@ssw0rd!"
}' localhost:50051 user.UserService/Login

# 2. 获取用户信息（需要 Token）
grpcurl -plaintext \
  -H "authorization: your-token-here" \
  -d '{"token": "your-token-here"}' \
  localhost:50051 user.UserService/GetProfile
```

## 监控和日志

### 日志级别

在 `config.yaml` 中配置：

```yaml
log:
  level: "info"    # debug, info, warn, error
  output: "stdout" # stdout, file
```

### 性能监控

每个 RPC 请求都会记录：
- 方法名
- 执行时间
- 成功/失败状态

示例：

```
INFO  gRPC 请求开始  method=/user.UserService/Login
DEBUG RPC 性能指标  method=/user.UserService/Login duration=45ms success=true
INFO  gRPC 请求成功  method=/user.UserService/Login duration=45ms
```

## 依赖注入

使用 `go.uber.org/dig` 管理依赖：

```go
Container
├── Config
├── *sqlx.DB
├── redis.Client
├── redis.Manager
├── UserRepository
├── UserService
└── UserServiceHandler
```

## 错误码设计

| 错误码 | 说明 |
|-------|------|
| 0 | 成功 |
| 40001 | 参数错误 |
| 40002 | 用户名或密码错误 |
| 40003 | Token 无效或已过期 |
| 40004 | 用户不存在 |
| 42901 | 请求过于频繁 |
| 50001 | 内部错误 |

## 性能优化

1. **数据库连接池**：100 个最大连接
2. **Redis 连接池**：10 个连接
3. **缓存优先**：用户信息优先从 Redis 读取
4. **异步缓存更新**：不阻塞主流程
5. **批量插入**：支持事务批量创建用户

## 下一步

TCP Server 已完成，接下来需要：

1. 实现 HTTP Server
2. HTTP Server 通过 gRPC Client 调用 TCP Server
3. 实现文件上传功能
4. 性能测试（QPS 3000+）

