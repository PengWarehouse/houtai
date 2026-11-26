-- 预置超级管理员账号
-- 用户名：admin
-- 初始密码：Admin@123456   <-- 登录成功后请立即修改
-- 哈希由 pkg/password.Hash("Admin@123456") 生成（bcrypt cost=10）
-- 幂等：靠 uk_users_username 唯一键，重复执行不会插入第二条
INSERT IGNORE INTO users (username, password_hash, nickname, email, role, status)
VALUES ('admin',
        '$2a$10$Chg8vB0FdKU1jsV8tZjzGusG7Jsl2yAXkl1PJNLLzs4n6M4qouxMK',
        '超级管理员',
        'admin@example.com',
        'super_admin',
        1);
