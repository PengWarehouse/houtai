-- 菜单表：支持父子层级，前端据此生成动态路由与侧边栏
-- component 留空表示「目录」，只作为父级分组，不注册路由
CREATE TABLE IF NOT EXISTS menus (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  parent_id  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '父级菜单 ID，0 表示顶级',
  title      VARCHAR(64)  NOT NULL COMMENT '菜单名称',
  path       VARCHAR(128) NOT NULL COMMENT '路由路径，如 /system/menu',
  icon       VARCHAR(64)  NOT NULL DEFAULT '' COMMENT 'Element Plus 图标组件名',
  component  VARCHAR(128) NOT NULL DEFAULT '' COMMENT 'views 下的组件路径，如 system/menu/index',
  sort       INT          NOT NULL DEFAULT 100 COMMENT '排序值，越小越靠前',
  status     TINYINT      NOT NULL DEFAULT 1 COMMENT '1 启用，0 禁用',
  created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_menus_parent (parent_id),
  KEY idx_menus_sort (sort)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='菜单表';

-- 初始菜单：固定 id 便于子菜单引用父级，INSERT IGNORE 保证可重复执行
INSERT IGNORE INTO menus (id, parent_id, title, path, icon, component, sort, status) VALUES
  (1, 0, '仪表盘', '/dashboard', 'Odometer', 'dashboard/index', 10, 1),
  (2, 0, '系统管理', '/system', 'Setting', '', 90, 1),
  (3, 2, '菜单管理', '/system/menu', 'Menu', 'system/menu/index', 10, 1),
  (4, 2, '用户管理', '/system/user', 'User', 'system/user/index', 20, 1),
  (5, 2, '角色管理', '/system/role', 'Role', 'system/role/index', 30, 1);

-- 资讯：目录 + 统一列表（新闻/视频在同一页按分类筛选）
-- id 用 9/10，避开库里历史遗留的重复「角色管理」(id=6)
-- 注意：role_menus 的清理放在 db/role.sql（该文件才创建 role_menus 表），
--       否则全新库执行本脚本时会因表不存在而失败。
DELETE FROM menus WHERE id IN (7, 8, 11);
UPDATE menus SET title = '资讯列表', path = '/info/list', icon = 'List',
                 component = 'info/list/index', sort = 10
 WHERE id = 10;
UPDATE menus SET icon = 'NewsPaper' WHERE id = 9;
INSERT IGNORE INTO menus (id, parent_id, title, path, icon, component, sort, status) VALUES
  (9, 0, '资讯', '/info', 'NewsPaper', '', 50, 1),
  (10, 9, '资讯列表', '/info/list', 'List', 'info/list/index', 10, 1);
