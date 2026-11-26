-- 角色表：code 与 users.role 对应，是用户与权限之间的唯一纽带
CREATE TABLE IF NOT EXISTS roles (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  code        VARCHAR(32)  NOT NULL COMMENT '角色标识，与 users.role 对应，如 super_admin',
  name        VARCHAR(64)  NOT NULL COMMENT '角色名称，如 超级管理员',
  description VARCHAR(255) NOT NULL DEFAULT '' COMMENT '角色说明',
  sort        INT          NOT NULL DEFAULT 100 COMMENT '排序值，越小越靠前',
  status      TINYINT      NOT NULL DEFAULT 1 COMMENT '1 启用，0 禁用',
  created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_roles_code (code),
  KEY idx_roles_sort (sort)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色表';

-- 角色-菜单关联：一行代表某角色可访问某个菜单
-- 注意：super_admin 不写关联记录，服务层直接放行全部菜单（默认拥有所有权限）
CREATE TABLE IF NOT EXISTS role_menus (
  role_id    BIGINT UNSIGNED NOT NULL,
  menu_id    BIGINT UNSIGNED NOT NULL,
  created_at DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (role_id, menu_id),
  KEY idx_role_menus_menu (menu_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色菜单关联表';

-- 清理已合并的旧「新闻列表 / 视频列表」菜单授权（菜单 id 7/8/11 已删除）
DELETE FROM role_menus WHERE menu_id IN (7, 8, 11);

-- 内置角色：固定 id，方便下面的权限种子引用
INSERT IGNORE INTO roles (id, code, name, description, sort, status) VALUES
  (1, 'super_admin', '超级管理员', '系统内置角色，默认拥有全部菜单与操作权限，不可删除、不可停用', 10, 1),
  (2, 'admin',       '管理员',     '负责日常管理，可用菜单由超级管理员分配', 20, 1),
  (3, 'user',        '普通用户',   '仅具备基础浏览权限', 30, 1);

-- 演示权限：管理员可看仪表盘、系统管理、用户管理；普通用户只看仪表盘
INSERT IGNORE INTO role_menus (role_id, menu_id) VALUES
  (2, 1), (2, 2), (2, 4),
  (3, 1);
