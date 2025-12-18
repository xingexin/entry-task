# 🚀 快速开始 - 100万用户压测

## 📋 前提条件

1. ✅ 数据库已有1000万用户数据
2. ✅ TCP服务器和HTTP服务器已启动

---

## ⚡ 超快速生成100万Token（1-2分钟）⚡⚡⚡

```bash
cd /Users/chuyao.zhuo/GolandProjects/entry-task/tcpserver/scripts

# 首次需要安装uuid依赖（redis/go-redis/v9项目中已有）
go get github.com/google/uuid

# 使用超快速脚本（直接Redis，绕过bcrypt）
go run batch_create_sessions.go -count 1000000
```

**预期输出：**
```
========================================
直接创建Redis Session（跳过bcrypt验证）
生成 1000000 个Session (并发: 500)
========================================
✅ Redis连接成功
进度: 12.5% (125000/1000000) 成功: 125000 失败: 0
进度: 25.8% (258000/1000000) 成功: 258000 失败: 0
进度: 50.3% (503000/1000000) 成功: 503000 失败: 0
进度: 75.6% (756000/1000000) 成功: 756000 失败: 0
进度: 99.9% (999000/1000000) 成功: 999000 失败: 0

========================================
✅ Session创建完成！
   成功: 1000000/1000000 (100.0%)
   失败: 0 (0.0%)
   耗时: 85.32s
   平均速度: 11721 Session/秒
========================================
✅ Token已保存到: tokens_1000000.txt
✅ Lua格式Token已保存到: tokens_1000000.lua

💡 提示：这些token有效期为2小时
```

**性能对比：**
- 旧方案（batch_login_fast.go）：1000000用户 ≈ 1.5-2小时
- 新方案（batch_create_sessions.go）：1000000用户 ≈ **1-2分钟**
- **提速：60-120倍！** 🚀

---

## 📊 修改压测脚本

### 修改 wrk_random_users.lua

```lua
-- 第7行，修改为：
local tokens = dofile("tokens_1000000.lua")
```

---

## 🎯 开始压测

### 场景1: 固定用户（200个）

```bash
# 清空缓存
redis-cli -h 192.168.215.2 -p 6379 --scan --pattern "user:*" | xargs redis-cli -h 192.168.215.2 -p 6379 DEL

# 压测30秒
wrk -t 10 -c 200 -d 30s --script wrk_fixed_users.lua http://localhost:8080/api/v1/profile
```

**预期结果：**
- QPS: > 3000
- P50延迟: < 100ms
- 缓存命中率: ~95%

---

### 场景2: 随机用户（100万个）

```bash
# 清空缓存
redis-cli -h 192.168.215.2 -p 6379 --scan --pattern "user:*" | xargs redis-cli -h 192.168.215.2 -p 6379 DEL

# 压测30秒
wrk -t 10 -c 200 -d 30s --script wrk_random_users.lua http://localhost:8080/api/v1/profile
```

**预期结果：**
- QPS: 1200-1800（根据固定QPS的35-50%）
- P50延迟: 150-250ms
- 缓存命中率: ~30-40%

---

## 📈 理想的性能差异

如果你的固定用户QPS = 3500：

```
固定用户（200个）：
  QPS: 3500
  P50延迟: 57ms
  缓存命中率: 95%
  主要瓶颈: Redis读取

随机用户（100万个）：
  QPS: 1200-1750  (固定的35-50%)
  P50延迟: 150-250ms
  缓存命中率: 30-40%
  主要瓶颈: MySQL查询

差异明显度: ⭐⭐⭐⭐⭐
```

---

## ⚠️ 常见问题

### Q1: 生成token时失败很多怎么办？

**A:** 检查服务是否正常：
```bash
curl http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"user00000001","password":"Test@123"}'
```

### Q2: 生成速度慢怎么办？

**A:** 确保使用 `batch_login_fast.go`（并发500），不要用 `batch_login.go`（并发50）

### Q3: 随机用户和固定用户QPS一样怎么办？

**A:** 
1. 用户池太小（至少50万-100万）
2. Redis缓存了所有用户（压测前清空缓存）
3. 检查 wrk_random_users.lua 是否正确加载了大用户池

### Q4: 100万用户token文件太大？

**A:** 
- `tokens_1000000.txt`: 约 60MB
- `tokens_1000000.lua`: 约 60MB
- 总计约 120MB，正常范围

---

## 🎯 完整测试流程（一次性）

```bash
# 1. 进入脚本目录
cd /Users/chuyao.zhuo/GolandProjects/entry-task/tcpserver/scripts

# 2. 生成100万token（约5分钟）
go run batch_login_fast.go -count 1000000

# 3. 修改lua脚本（手动编辑wrk_random_users.lua第7行）
# local tokens = dofile("tokens_1000000.lua")

# 4. 固定用户压测
redis-cli -h 192.168.215.2 -p 6379 FLUSHDB
wrk -t 10 -c 200 -d 30s --script wrk_fixed_users.lua http://localhost:8080/api/v1/profile

# 5. 随机用户压测
redis-cli -h 192.168.215.2 -p 6379 FLUSHDB
wrk -t 10 -c 200 -d 30s --script wrk_random_users.lua http://localhost:8080/api/v1/profile

# 6. 对比结果！
```

---

## 💡 优化建议

如果性能不达标，检查：

1. **数据库连接池**：`config.yaml` 中 `max_open_conns` 应 > 200
2. **Redis连接池**：`config.yaml` 中 `pool_size` 应 > 100
3. **日志级别**：改为 `warn` 减少IO
4. **数据库索引**：确保 `users` 表的 `id` 字段有索引

---

祝压测顺利！🎉

