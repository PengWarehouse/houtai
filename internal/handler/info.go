package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"goProject/internal/service"
	"goProject/pkg/httpx"
)

// InfoService 声明 handler 需要的资讯能力，由 service.InfoCatalog 实现。
type InfoService interface {
	List(ctx context.Context, kind, keyword string, page, size int) (*service.InfoListResult, error)
	Save(ctx context.Context, input service.InfoSaveInput) (*service.InfoSaveResult, error)
}

// listInfoItems 处理 GET /api/info/items?kind=&keyword=&page=1&size=10。
func (s *Server) listInfoItems(w http.ResponseWriter, r *http.Request) {
	if s.infoSvc == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "资讯服务不可用")
		return
	}

	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	size, _ := strconv.Atoi(query.Get("size"))
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 500 {
		size = 50
	}

	result, err := s.infoSvc.List(r.Context(), query.Get("kind"), query.Get("keyword"), page, size)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInfoKind):
			httpx.Error(w, http.StatusBadRequest, err.Error())
		default:
			slog.Warn("list info items failed", "err", err)
			httpx.Error(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	if result == nil {
		result = &service.InfoListResult{Items: []service.InfoDetail{}, Total: 0}
	}
	if result.Items == nil {
		result.Items = []service.InfoDetail{}
	}
	httpx.OK(w, result)
}

// saveInfoItems 处理 POST /api/info/items { kind, items: [...] }。
// 只有用户勾选后前端才会提交；服务端仍会逐条校验并去重，未通过的不会写库。
func (s *Server) saveInfoItems(w http.ResponseWriter, r *http.Request) {
	if s.infoSvc == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "资讯服务不可用")
		return
	}

	var req service.InfoSaveInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	result, err := s.infoSvc.Save(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInfoKind), errors.Is(err, service.ErrInfoInvalid):
			httpx.Error(w, http.StatusBadRequest, err.Error())
		default:
			slog.Warn("save info items failed", "err", err)
			httpx.Error(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	httpx.OK(w, result)
}
