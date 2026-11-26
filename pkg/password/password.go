// Package password 负责密码的哈希与校验（bcrypt）。
package password

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Cost 是 bcrypt 的计算强度，数值越大越慢越安全，10 是常用折中值。
const Cost = 10

// Hash 把明文密码转换成 bcrypt 哈希，用于入库。
func Hash(plain string) (string, error) {
	if plain == "" {
		return "", fmt.Errorf("密码不能为空")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), Cost)
	if err != nil {
		return "", fmt.Errorf("生成密码哈希失败: %w", err)
	}
	return string(hashed), nil
}

// Verify 校验明文密码是否与哈希匹配。
func Verify(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
