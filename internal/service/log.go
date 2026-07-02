package service

import (
	"log/slog"
	"runtime"
	"strings"

	"collections/internal/store"
	"collections/internal/util"
)

// logError 用于 service 公开方法的 defer 错误日志拦截。
// 方法返回时若 err != nil，自动通过 runtime.Caller 获取方法名并记录 slog.Error。
//
//	func (s *XXXService) Method(...) (_ T, err error) {
//	    defer logError(&err)
//	    ...
//	}
func logError(err *error) {
	if *err == nil {
		return
	}
	pc, _, _, ok := runtime.Caller(1)
	method := "unknown"
	if ok {
		name := runtime.FuncForPC(pc).Name()
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}
		method = name
	}
	slog.Error("service call failed", "method", method, "error", *err)
}

// adjustTagCounts 计算新旧标签的差值，批量更新标签注册表 count。
// prefix 为标签类型前缀（"url_tag" 或 "media_tag"）。
func adjustTagCounts(s *store.Store, prefix string, newTags, oldTags []string) {
	deltas := util.ComputeTagDeltas(newTags, oldTags)
	if len(deltas) == 0 {
		return
	}
	if err := s.BatchAdjustTagCounts(prefix, deltas); err != nil {
		slog.Warn("failed to adjust tag counts", "prefix", prefix, "err", err)
	}
}
