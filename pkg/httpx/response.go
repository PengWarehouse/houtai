// Package httpx 提供统一的 HTTP 响应封装。
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// 业务状态码约定：
//   - 成功：Code = 200
//   - 失败：Code = 0，失败原因写在 Message 里
//
// HTTP 状态码仍然保持语义正确（200/400/401/403/404/500），
// 方便代理、网关和浏览器按标准处理；前端以 Code 判定业务成败。
const (
	// CodeSuccess 业务成功。
	CodeSuccess = http.StatusOK
	// CodeFailure 业务失败，具体原因见 Message。
	CodeFailure = 0
)

// Envelope 是统一的响应结构。
type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// JSON 以指定状态码输出 JSON 响应。
func JSON(w http.ResponseWriter, status int, payload Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("write response failed", "err", err)
	}
}

// OK 输出成功响应（code=200），data 为业务数据。
func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, Envelope{Code: CodeSuccess, Message: "ok", Data: data})
}

// Success 输出不带业务数据的成功响应，用于「保存成功」这类只关心结果的场景。
func Success(w http.ResponseWriter, message string) {
	if message == "" {
		message = "ok"
	}
	JSON(w, http.StatusOK, Envelope{Code: CodeSuccess, Message: message})
}

// Error 输出失败响应，code 固定为 0，失败原因放在 message。
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, Envelope{Code: CodeFailure, Message: message})
}
