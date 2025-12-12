CREATE TABLE `username_mapping` (
                                   `username` VARCHAR(64) NOT NULL COMMENT '用户名（全局唯一）',
                                   `shard_index` TINYINT UNSIGNED NOT NULL COMMENT '分表索引 (0-3)，用户数据所在的表编号',
                                   `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
                                   PRIMARY KEY (`username`),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户名到分表索引映射表';
