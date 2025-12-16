# HTTP Server

## 项目架构

```
httpserver/
├── cmd/
│   └── httpserver/
│       └── main.go              # 主程序入口
├── internal/
│   ├── handler/                 # HTTP Handler
│   │   └── user_handler.go
│   ├── middleware/              # HTTP 中间件
│   │   └── middleware.go
│   └── router/                  # 路由注册
│       └── router.go
├── pkg/
│   ├── logger/                  # 日志工具
│   │   └── logger.go
│   └── response/                # 统一响应
│       ├── response.go
│       └── code.go
├── config/
│   ├── config.go
│   └── config.yaml              # 配置文件
├── uploads/
│   └── avatars/                 # 头像存储目录
└── static/
    └── default_avatar.png       # 默认头像
```

## 功能特性

### ✅ 已实现功能

1. **HTTP API**
   - 用户登录 (`POST /api/v1/auth/login`)
   - 用户登出 (`POST /api/v1/auth/logout`)
   - 获取用户信息 (`GET /api/v1/profile`)
   - 更新昵称 (`PATCH /api/v1/profile/nickname`)
   - 上传头像 (`POST /api/v1/profile/picture`)
   - 获取头像 (`GET /api/v1/profile/picture`)

2. **中间件**
   - **Recovery**：捕获 Panic
   - **CORS**：跨域支持
   - **Logger**：HTTP 请求日志

3. **gRPC Client**
   - 连接 TCP Server
   - 自动重连机制
   - Token 透传（通过 metadata）

4. **文件上传**
   - 文件大小验证（5MB）
   - 文件类型验证（jpg, png, webp）
   - 本地文件系统存储

## 启动步骤

### 1. 确保 TCP Server 已启动

HTTP Server 依赖 TCP Server，请先启动 TCP Server：

```bash
cd tcpserver
go run cmd/tcpserver/main.go -config config/config.yaml
```

### 2. 修改配置文件

编辑 `config/config.yaml`：

```yaml
server:
  host: "0.0.0.0"
  port: 8080           # HTTP Server 端口

grpc:
  host: "localhost"    # TCP Server 地址
  port: 50051          # TCP Server 端口

log:
  level: "info"
  output: "stdout"
```

### 3. 创建必要目录

```bash
mkdir -p uploads/avatars
mkdir -p static
```

### 4. 启动 HTTP Server

```bash
# 在项目根目录执行
cd httpserver
go run cmd/httpserver/main.go -config config/config.yaml
```

启动成功后，会看到：

```
INFO  HTTP Server 启动中...
INFO  配置加载成功
INFO  正在连接 gRPC Server...  addr=localhost:50051
INFO  gRPC 连接成功
INFO  Handler 创建成功
INFO  路由设置完成
INFO  HTTP Server 启动成功  addr=0.0.0.0:8080
```

## API 文档

### **1. 登录**

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "user00000001",
  "password": "P@ssw0rd!"
}

Response (成功):
{
  "code": 0,
  "message": "OK",
  "data": {
    "username": "user00000001",
    "nickname": "Sam",
    "avatar_url": "/api/v1/profile/picture"
  }
}

Response Header:
X-Auth-Token: session-token-here
```

### **2. 获取用户信息**

```http
GET /api/v1/profile
Authorization: Bearer session-token-here

Response:
{
  "code": 0,
  "message": "OK",
  "data": {
    "username": "user00000001",
    "nickname": "Sam",
    "avatar_url": "/api/v1/profile/picture"
  }
}
```

### **3. 更新昵称**

```http
PATCH /api/v1/profile/nickname
Authorization: Bearer session-token-here
Content-Type: application/json

{
  "nickname": "小明🚀"
}

Response:
{
  "code": 0,
  "message": "OK",
  "data": {
    "username": "user00000001",
    "nickname": "小明🚀",
    "avatar_url": "/api/v1/profile/picture"
  }
}
```

### **4. 上传头像**

```http
POST /api/v1/profile/picture
Authorization: Bearer session-token-here
Content-Type: multipart/form-data

Form Data:
  file: [binary]

Response:
{
  "code": 0,
  "message": "OK",
  "data": {
    "avatar_url": "/api/v1/profile/picture"
  }
}
```

### **5. 获取头像**

```http
GET /api/v1/profile/picture
Authorization: Bearer session-token-here

Response:
[图片二进制数据]
Content-Type: image/jpeg
```

### **6. 登出**

```http
POST /api/v1/auth/logout
Authorization: Bearer session-token-here

Response:
{
  "code": 0,
  "message": "OK",
  "data": {}
}
```

## 错误码

| Code | 说明 |
|------|------|
| 0 | 成功 |
| 40100 | 未认证 |
| 40103 | 用户名或密码错误 |
| 40104 | 无效的昵称 |
| 40006 | 文件过大 |
| 40007 | 不支持的文件类型 |
| 50002 | RPC 调用错误 |
| 50000 | 服务器内部错误 |

## 架构设计

### **请求流程**

```
客户端 (浏览器/Postman)
    ↓ HTTP 请求
Gin Router
    ↓ 中间件（Recovery, CORS, Logger）
Handler
    ↓ 提取 Token
gRPC Client
    ↓ metadata（Token）
TCP Server (gRPC)
    ↓ 业务逻辑
MySQL + Redis
    ↓
返回结果
    ↓
Handler 转换为 JSON
    ↓
HTTP 响应
```

### **文件存储**

```
上传：
客户端 → HTTP Server (保存到本地：./uploads/avatars/{userID}.jpg)
                    ↓
                   gRPC
                    ↓
              TCP Server (存储 URL：/uploads/avatars/{userID}.jpg)
                    ↓
                 数据库

获取：
客户端 → HTTP Server (gRPC 获取 URL)
                    ↓
           转换为本地路径
                    ↓
           返回图片二进制
```

## 测试示例

### 使用 curl 测试

```bash
# 1. 登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"user00000001","password":"P@ssw0rd!"}' \
  -v

# 从响应头获取 Token：X-Auth-Token

# 2. 获取用户信息
curl http://localhost:8080/api/v1/profile \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"

# 3. 更新昵称
curl -X PATCH http://localhost:8080/api/v1/profile/nickname \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{"nickname":"小明"}'

# 4. 上传头像
curl -X POST http://localhost:8080/api/v1/profile/picture \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -F "file=@avatar.jpg"

# 5. 获取头像
curl http://localhost:8080/api/v1/profile/picture \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  --output avatar.jpg

# 6. 登出
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

## 中间件说明

### **1. Recovery**
- 捕获所有 Panic
- 防止服务器崩溃
- 返回 500 错误

### **2. CORS**
- 允许跨域请求
- 支持所有来源 (`*`)
- 处理 OPTIONS 预检请求

### **3. Logger**
- 记录所有 HTTP 请求
- 包含：方法、路径、状态码、耗时、客户端 IP
- 方便监控和调试

## 依赖注入

```go
main.go
  ↓
创建 gRPC Client
  ↓
创建 Handler (注入 gRPC Client)
  ↓
设置路由 (注入 Handler)
  ↓
启动 HTTP Server
```

## 性能优化

1. **连接复用**：gRPC Client 复用一个连接
2. **超时控制**：每个 RPC 调用 3 秒超时
3. **异步日志**：不阻塞主流程
4. **文件直接返回**：使用 `c.File()` 高效返回图片

## 监控和日志

### 日志示例

```
[INFO]  2024-01-01 12:00:00  HTTP Server 启动成功  addr=0.0.0.0:8080
[INFO]  2024-01-01 12:00:01  HTTP 请求  method=POST path=/api/v1/auth/login status=200 duration=45ms client_ip=127.0.0.1
[ERROR] 2024-01-01 12:00:02  RPC调用失败  error=connection refused
```

## 下一步

HTTP Server 已完成，接下来可以：

1. 添加性能测试（wrk, jmeter）
2. 添加单元测试
3. 优化文件上传（支持更多格式）
4. 添加限流中间件
5. 集成 Prometheus 监控

