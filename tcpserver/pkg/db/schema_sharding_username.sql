-- =============================================================================
-- 用户表分表设计（按username哈希分表 + 雪花ID）
-- =============================================================================
-- 分表策略：table_index = hash(username) % 4
-- 主键策略：使用雪花ID（Snowflake ID），不使用AUTO_INCREMENT
-- 数据量：1000万数据分成4张表，每张表约250万数据
-- 
-- 优势：
-- ✅ 登录查询：根据username直接定位到一张表，性能最优
-- ✅ 数据分布：哈希算法保证数据均匀分布
-- ✅ 扩展性：可平滑扩展到8张、16张表
-- ✅ 唯一性：同一username永远路由到同一张表，自动保证全局唯一
-- =============================================================================

-- 删除已存在的表（可选，仅用于重新创建）
-- DROP TABLE IF EXISTS `users_0`;
-- DROP TABLE IF EXISTS `users_1`;
-- DROP TABLE IF EXISTS `users_2`;
-- DROP TABLE IF EXISTS `users_3`;

-- =============================================================================
-- 用户表 - 分表0
-- =============================================================================
CREATE TABLE IF NOT EXISTS `users_0` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID，雪花ID（分布式唯一，不使用自增）',
    `username` VARCHAR(64) NOT NULL COMMENT '用户名，唯一，用于登录（不可修改）',
    `password_hash` VARCHAR(255) NOT NULL COMMENT '密码哈希值（使用bcrypt算法，cost=10）',
    `nickname` VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '昵称（支持完整Unicode字符集，包括中文、emoji等）',
    `profile_picture` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '头像URL或文件路径',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_username` (`username`) COMMENT '用户名唯一索引，用于登录查询',
    KEY `idx_created_at` (`created_at`) COMMENT '创建时间索引，用于按时间排序'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表-分表0（按username哈希路由）';

-- =============================================================================
-- 用户表 - 分表1
-- =============================================================================
CREATE TABLE IF NOT EXISTS `users_1` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID，雪花ID（分布式唯一，不使用自增）',
    `username` VARCHAR(64) NOT NULL COMMENT '用户名，唯一，用于登录（不可修改）',
    `password_hash` VARCHAR(255) NOT NULL COMMENT '密码哈希值（使用bcrypt算法，cost=10）',
    `nickname` VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '昵称（支持完整Unicode字符集，包括中文、emoji等）',
    `profile_picture` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '头像URL或文件路径',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_username` (`username`) COMMENT '用户名唯一索引，用于登录查询',
    KEY `idx_created_at` (`created_at`) COMMENT '创建时间索引，用于按时间排序'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表-分表1（按username哈希路由）';

-- =============================================================================
-- 用户表 - 分表2
-- =============================================================================
CREATE TABLE IF NOT EXISTS `users_2` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID，雪花ID（分布式唯一，不使用自增）',
    `username` VARCHAR(64) NOT NULL COMMENT '用户名，唯一，用于登录（不可修改）',
    `password_hash` VARCHAR(255) NOT NULL COMMENT '密码哈希值（使用bcrypt算法，cost=10）',
    `nickname` VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '昵称（支持完整Unicode字符集，包括中文、emoji等）',
    `profile_picture` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '头像URL或文件路径',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_username` (`username`) COMMENT '用户名唯一索引，用于登录查询',
    KEY `idx_created_at` (`created_at`) COMMENT '创建时间索引，用于按时间排序'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表-分表2（按username哈希路由）';

-- =============================================================================
-- 用户表 - 分表3
-- =============================================================================
CREATE TABLE IF NOT EXISTS `users_3` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID，雪花ID（分布式唯一，不使用自增）',
    `username` VARCHAR(64) NOT NULL COMMENT '用户名，唯一，用于登录（不可修改）',
    `password_hash` VARCHAR(255) NOT NULL COMMENT '密码哈希值（使用bcrypt算法，cost=10）',
    `nickname` VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '昵称（支持完整Unicode字符集，包括中文、emoji等）',
    `profile_picture` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '头像URL或文件路径',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_username` (`username`) COMMENT '用户名唯一索引，用于登录查询',
    KEY `idx_created_at` (`created_at`) COMMENT '创建时间索引，用于按时间排序'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表-分表3（按username哈希路由）';

-- =============================================================================
-- 索引说明
-- =============================================================================
-- 1. PRIMARY KEY (id)
--    - 雪花ID主键，保证全局唯一
--    - 趋势递增，利于B+树索引
--    - 用于内部关联和快速查询
--
-- 2. UNIQUE KEY (username)
--    - 用户名唯一索引，登录查询的核心索引
--    - 覆盖索引，无需回表查询
--    - 同一username永远路由到同一张表，保证全局唯一
--
-- 3. KEY (created_at)
--    - 用于按注册时间排序
--    - 用于统计分析
--
-- =============================================================================
-- 分表路由规则
-- =============================================================================
-- 路由算法：table_index = hash(username) % 4
-- 
-- 示例：
-- hash("user00000001") % 4 = 0 -> users_0
-- hash("user00000002") % 4 = 1 -> users_1
-- hash("admin")        % 4 = 2 -> users_2
-- hash("zhangsan")     % 4 = 3 -> users_3
--
-- 哈希算法：SHA256（保证均匀分布）
--
-- =============================================================================
-- 使用说明
-- =============================================================================
-- 1. 执行此脚本创建表：
--    mysql -u username -p database_name < schema_sharding_username.sql
--
-- 2. 验证表创建成功：
--    SHOW TABLES LIKE 'users_%';
--
-- 3. 查看表结构：
--    DESC users_0;
--
-- 4. 查看索引：
--    SHOW INDEX FROM users_0;
--
-- =============================================================================
-- 性能优化建议
-- =============================================================================
-- 1. 连接池配置：
--    - max_connections: 根据实际并发调整（建议500-1000）
--    - connection_timeout: 10s
--
-- 2. InnoDB缓冲池：
--    - innodb_buffer_pool_size: 设置为物理内存的70-80%
--
-- 3. 查询缓存：
--    - query_cache_type: OFF（MySQL 8.0已移除）
--
-- 4. 慢查询日志：
--    - slow_query_log: ON
--    - long_query_time: 1（记录超过1秒的查询）
--
-- =============================================================================
-- 数据迁移和扩容
-- =============================================================================
-- 如需扩容到8张表（每张表约125万数据）：
-- 1. 创建users_4, users_5, users_6, users_7
-- 2. 修改路由规则：table_index = hash(username) % 8
-- 3. 迁移数据：使用一致性哈希减少数据迁移量
--
-- =============================================================================
