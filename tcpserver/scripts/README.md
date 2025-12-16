# 测试数据生成脚本

## 功能说明

`generate_test_data.go` 用于批量生成 1000 万测试用户数据，用于性能测试。

## 使用步骤

### 1. 创建数据库表

首先执行 SQL 初始化脚本：

```bash
# 方式1：使用 mysql 命令行
mysql -u root -p entrytask < ../pkg/db/schema.sql

# 方式2：登录 MySQL 后执行
mysql -u root -p
USE entrytask;
SOURCE /path/to/entry-task/tcpserver/pkg/db/schema.sql;
```

### 2. 运行数据生成脚本

```bash
cd tcpserver/scripts

# 使用默认配置（推荐）
go run generate_test_data.go

# 或者自定义参数
go run generate_test_data.go \
  -dsn "root:password@tcp(localhost:3306)/entrytask?charset=utf8mb4&parseTime=True&loc=Local" \
  -workers 10 \
  -batch 5000
```

### 3. 参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-dsn` | MySQL 数据源名称 | `root:root@tcp(192.168.215.4:3306)/entrytask?charset=utf8mb4&parseTime=True&loc=Local` |
| `-workers` | 并发 worker 数量 | `10` |
| `-batch` | 每批插入数量 | `5000` |

## 性能数据

### 预期性能

- **数据量**：10,000,000 条
- **批次大小**：5000 条/批
- **并发数**：10 个 worker
- **预计时间**：
  - 机械硬盘：约 30-60 分钟
  - SSD：约 10-20 分钟
  - NVMe SSD：约 5-10 分钟
- **平均速度**：10,000 - 30,000 条/秒

### 实际表现

实际速度取决于：
- 数据库服务器性能（CPU、内存、磁盘 I/O）
- 网络延迟（如果数据库在远程服务器）
- MySQL 配置（InnoDB 缓冲池大小、日志文件大小等）

## 生成的数据格式

### 用户账号规则

- **用户名**：`user00000001` ~ `user10000000`（8 位数字，补齐前导零）
- **密码**：统一为 `P@ssw0rd!`
- **昵称**：`User1` ~ `User10000000`
- **头像**：默认为空

### 示例数据

| id | username | password_hash | nickname | profile_picture |
|----|----------|---------------|----------|-----------------|
| 1234567890 | user00000001 | $2a$10$... | User1 | |
| 1234567891 | user00000002 | $2a$10$... | User2 | |
| ... | ... | ... | ... | ... |

## 常见问题

### Q1: 执行脚本时提示 "users 表不存在"

**A:** 请先执行 `schema.sql` 创建表：
```bash
mysql -u root -p entrytask < ../pkg/db/schema.sql
```

### Q2: 如何清空已有数据重新生成？

**A:** 执行以下 SQL：
```sql
TRUNCATE TABLE users;
```

### Q3: 生成速度很慢怎么办？

**A:** 尝试以下优化：

1. **增加批次大小**：
   ```bash
   go run generate_test_data.go -batch 10000
   ```

2. **调整 MySQL 配置**（`my.cnf`）：
   ```ini
   [mysqld]
   innodb_buffer_pool_size = 4G
   innodb_log_file_size = 512M
   innodb_flush_log_at_trx_commit = 2
   sync_binlog = 0
   ```

3. **临时关闭索引**（仅在初始化时使用）：
   ```sql
   ALTER TABLE users DISABLE KEYS;
   -- 执行数据生成脚本
   ALTER TABLE users ENABLE KEYS;
   ```

### Q4: 如何验证数据是否正确？

**A:** 执行以下 SQL 验证：
```sql
-- 查看总数
SELECT COUNT(*) FROM users;

-- 查看示例数据
SELECT * FROM users LIMIT 10;

-- 检查用户名唯一性
SELECT COUNT(DISTINCT username) FROM users;

-- 测试登录查询性能
SELECT * FROM users WHERE username = 'user00000001';
```

## 测试登录

生成数据后，可以使用以下账号测试：

```bash
# 示例账号
用户名: user00000001
密码: P@ssw0rd!

用户名: user00500000
密码: P@ssw0rd!

用户名: user09999999
密码: P@ssw0rd!
```

## 注意事项

⚠️ **警告**：
- 生成 1000 万数据需要约 **4-5GB** 磁盘空间
- 执行过程中会占用较高的 CPU 和 I/O 资源
- 建议在非生产环境或低峰时段执行
- 数据生成过程中不要中断，否则可能导致数据不完整

## 卸载测试数据

如果需要删除测试数据：

```sql
-- 清空表（保留表结构）
TRUNCATE TABLE users;

-- 或者删除表
DROP TABLE users;
```

