-- 用户表：所有账号（含超级管理员）都存放在这里
-- 幂等：重复执行不会报错
CREATE TABLE IF NOT EXISTS users (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
    username      VARCHAR(64)     NOT NULL COMMENT '登录名，唯一',
    password_hash VARCHAR(255)    NOT NULL COMMENT 'bcrypt 哈希，禁止存明文',
    nickname      VARCHAR(64)     NOT NULL DEFAULT '' COMMENT '昵称',
    email         VARCHAR(128)             DEFAULT NULL COMMENT '邮箱',
    role          VARCHAR(32)     NOT NULL DEFAULT 'user' COMMENT '角色：super_admin / admin / user',
    status        TINYINT         NOT NULL DEFAULT 1 COMMENT '1=正常 0=禁用',
    last_login_at DATETIME                 DEFAULT NULL COMMENT '最后登录时间',
    created_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_users_username (username),
    KEY idx_users_role (role)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT = '用户表';

-- 资讯条目：AI 问询提取后，由用户勾选入库的新闻 / 视频
-- kind = news（新闻）或 video（视频）；url 前缀唯一索引用于兜底去重
CREATE TABLE IF NOT EXISTS info_items (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
    kind         VARCHAR(16)     NOT NULL COMMENT '类型：news=新闻 video=视频',
    title        VARCHAR(255)    NOT NULL COMMENT '标题',
    url          VARCHAR(1024)   NOT NULL COMMENT '链接地址',
    source       VARCHAR(128)    NOT NULL DEFAULT '' COMMENT '来源 / 作者',
    published_at DATE                     DEFAULT NULL COMMENT '发布日期',
    created_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '入库时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_info_items_kind_url (kind, url(191)),
    KEY idx_info_items_kind_created (kind, created_at)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT = '资讯条目表';
