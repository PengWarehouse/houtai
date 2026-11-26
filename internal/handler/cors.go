package handler

import (
	"log/slog"
	"net/http"
	"strings"
)

// corsHeaders 是允许浏览器携带的请求头。
// Authorization 必须放开，前端靠 Bearer token 鉴权。
const corsHeaders = "Authorization, Content-Type, X-Requested-With"

// corsMethods 是允许的跨域方法。
const corsMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"

// cors 返回一个 CORS 中间件。
// 单 → 回显该 Origin（而不是 *），并补齐 CORS 响应头
//   - 预检请求（OPTIO
//
// allowedOrigins 为允许的来源列表，支持精确匹配（如 http://localhost:5173）
// 与通配符 "*"。列表为空表示完全关闭跨域（同源部署时不需要）。
//
// 处理逻辑：
//   - 请求没带 Origin（同源请求、curl、服务端调用）→ 直接放行，不加任何头
//   - Origin 命中白名NS + Access-Control-Request-Method）→ 直接返回 204，不进业务路由
//   - Origin 不在白名单 → 预检返回 403，普通请求放行但不加头（浏览器会自己拦下）
func cors(allowedOrigins []string) func(http.Handler) http.Handler {
	allowAll := false
	allowed := make(map[string]struct{}, len(allowedOrigins))

	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			allowAll = true
			continue
		}
		// 统一去掉结尾的斜杠，避免 http://a.com/ 与 http://a.com 匹配不上
		allowed[strings.TrimRight(o, "/")] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimRight(r.Header.Get("Origin"), "/")

			// 没有 Origin 说明不是跨域请求，直接放行
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// 只要带了 Origin，响应就必须声明 Vary，否则缓存可能串味
			w.Header().Add("Vary", "Origin")

			_, hit := allowed[origin]
			if !hit && !allowAll {
				if r.Method == http.MethodOptions {
					slog.Warn("cors origin rejected",
						"origin", origin, "path", r.URL.Path,
						"hint", "把该地址加入 CORS_ALLOW_ORIGINS")
					w.WriteHeader(http.StatusForbidden)
					return
				}
				// 普通请求交给业务，浏览器端看不到 CORS 头会自行拦截
				next.ServeHTTP(w, r)
				return
			}

			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Methods", corsMethods)
			h.Set("Access-Control-Allow-Headers", corsHeaders)
			h.Set("Access-Control-Expose-Headers", "Content-Length")
			// 预检结果缓存 10 分钟，减少 OPTIONS 次数
			h.Set("Access-Control-Max-Age", "600")

			// 预检请求不需要业务处理，直接结束
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
