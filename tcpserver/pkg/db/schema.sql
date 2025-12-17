-- =============================================================================
-- 用户表设计（单表 + 雪花ID）
-- =============================================================================
-- 设计说明：
-- - 单表设计，支持 1000 万用户数据
-- - 主键使用雪花ID（Snowflake ID），趋势递增，利于B+树索引
-- - username 唯一索引，用于登录查询
-- - 支持完整 Unicode 字符集（utf8mb4），包括 emoji
-- =============================================================================

-- 删除已存在的表（可选，仅用于重新创建）
-- DROP TABLE IF EXISTS `users`;

-- =============================================================================
-- 用户表
-- =============================================================================
CREATE TABLE IF NOT EXISTS `users` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID，雪花ID（分布式唯一，不使用自增）',
    `username` VARCHAR(64) NOT NULL COMMENT '用户名，唯一，用于登录（不可更改）',
    `password_hash` VARCHAR(255) NOT NULL COMMENT '密码哈希值（使用bcrypt算法，cost=10）',
    `nickname` VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '昵称（支持完整Unicode字符集，包括中文、emoji等）',
    `profile_picture` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '头像URL或文件路径',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_username` (`username`) COMMENT '用户名唯一索引，用于登录查询',
    KEY `idx_created_at` (`created_at`) COMMENT '创建时间索引，用于按时间排序'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表（单表设计，支持1000万数据）';

-- =============================================================================
-- 索引说明
-- =============================================================================
-- 1. PRIMARY KEY (id)
--    - 雪花ID主键，保证全局唯一
--    - 趋势递增，利于B+树索引，避免页分裂
--    - 用于内部关联和快速查询
--
-- 2. UNIQUE KEY (username)
--    - 用户名唯一索引，登录查询的核心索引
--    - 覆盖索引，无需回表查询
--    - 保证用户名全局唯一
--
-- 3. KEY (created_at)
--    - 用于按注册时间排序
--    - 用于统计分析和数据归档
--
-- =============================================================================
-- 使用说明
-- =============================================================================
-- 1. 创建数据库（如果不存在）：
--    CREATE DATABASE IF NOT EXISTS `entrytask` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
--
-- 2. 执行此脚本创建表：
--    mysql -u root -p entrytask < schema.sql
--
-- 3. 验证表创建成功：
--    USE entrytask;
--    SHOW TABLES;
--    DESC users;
--
-- 4. 查看索引：
--    SHOW INDEX FROM users;
--
-- =============================================================================
-- 性能优化建议（针对 1000 万数据）
-- =============================================================================
-- 1. MySQL 配置优化：
--    [mysqld]
--    # InnoDB 缓冲池（设置为物理内存的 70-80%）
--    innodb_buffer_pool_size = 8G
--    
--    # 连接数
--    max_connections = 1000
--    
--    # 日志文件大小
--    innodb_log_file_size = 512M
--    
--    # 写入优化
--    innodb_flush_log_at_trx_commit = 2
--    sync_binlog = 0
--    
--    # 字符集
--    character_set_server = utf8mb4
--    collation_server = utf8mb4_unicode_ci
--
-- 2. 查询优化：
--    - 使用 username 查询时，会命中 UNIQUE KEY，性能最优
--    - 使用 id 查询时，直接使用主键，性能最优
--    - 避免全表扫描，尽量使用索引
--
-- 3. 批量插入优化：
--    - 使用事务批量提交（每 1000-5000 条提交一次）
--    - 关闭自动提交：SET autocommit=0;
--    - 使用 prepared statement 提高性能
--
-- =============================================================================
-- 测试数据生成
-- =============================================================================
-- 使用提供的 Go 脚本生成 1000 万测试数据：
--    cd tcpserver/scripts
--    go run generate_test_data.go
--
-- =============================================================================
-- 数据量估算
-- =============================================================================
-- 单条记录大小估算：
-- - id: 8 bytes
-- - username: 平均 20 bytes
-- - password_hash: 60 bytes (bcrypt)
-- - nickname: 平均 20 bytes
-- - profile_picture: 平均 50 bytes
-- - created_at: 8 bytes
-- - updated_at: 8 bytes
-- - 索引开销: 约 100 bytes
-- 总计：约 300 bytes/条
--
-- 1000 万条记录：
-- - 数据: 300 bytes × 10,000,000 = 3GB
-- - 索引: 约 1GB
-- - 总计: 约 4-5GB（包括碎片和其他开销）
--
-- =============================================================================


