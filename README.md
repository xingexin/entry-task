# 运行方法

***只有这个markdown是人写的，其他的全部是ai生成***

## 启动服务
### 运行 TCP Server
`go run ./tcpserver/cmd/tcpserver/main.go -config ./tcpserver/config/config.yaml`

### 运行 HTTP Server
`go run ./httpserver/cmd/httpserver/main.go -config ./httpserver/config/config.yaml`


## 测试方式

**先cd到scripts目录下**

`cd tcpserver/scripts`

**运行`batch_create_sessions.go`先批量预登录，这里演示批量创建200个**

`go run batch_create_sessions.go -count 200`

**修改wrk_fixed_users.lua或者wrk_random_users.lua里的测试文件名`local tokens = dofile("tokens_10000000.lua")`成你生成的文件名**

**接下来可以开始测试**

### 场景1: 200并发固定用户（目标QPS > 3000）
`wrk -t 10 -c 200 -d 60s --script wrk_fixed_users.lua
http://localhost:8080/api/v1/profile`


### 场景2: 200并发随机用户（目标QPS > 1000）
`wrk -t 10 -c 200 -d 60s --script wrk_random_users.lua
http://localhost:8080/api/v1/profile`

### 场景3: 2000并发固定用户（目标QPS > 3000）
`wrk -t 10 -c 2000 -d 60s --script wrk_fixed_users.lua
http://localhost:8080/api/v1/profile`


### 场景4: 2000并发随机用户（目标QPS > 1000）
`wrk -t 10 -c 2000 -d 60s --script wrk_random_users.lua
http://localhost:8080/api/v1/profile`

# 已知bug

上传图片后，若服务器内存储的图片被意外删除且数据库中的路径没有改变
，将会无法再上传图片，只能手动操作数据库删除对应的URL