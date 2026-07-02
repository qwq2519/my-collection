package service

import (
	"sync"

	"collections/internal/model"
	"collections/internal/store"
	"collections/internal/util"
)

// SettingService 应用设置业务逻辑层。
// 公开方法即前端可调用接口（通过 Wails 绑定）。
// FFmpegBinPath 由 scan worker goroutine 调用，与前端调用存在并发，
// 因此 cachedFFmpeg 通过 ffmpegMu 保护。
type SettingService struct {
	Store *store.Store

	ffmpegMu     sync.RWMutex
	cachedFFmpeg *model.FFmpegStatus
}

// GetSettings 返回应用设置摘要（设置页进入时调用）
func (s *SettingService) GetSettings() (_ *model.AppSettings, err error) {
	defer logError(&err)
	return &model.AppSettings{
		PersistDir:  s.Store.PersistDir(),
		FFmpeg:      s.GetFFmpegStatus(),
		IndexStatus: s.Store.GetDirtyIndexStatus(),
	}, nil
}

// GetFFmpegStatus 返回 ffmpeg 可用状态，首次调用后缓存结果
func (s *SettingService) GetFFmpegStatus() model.FFmpegStatus {
	s.ffmpegMu.RLock()
	cached := s.cachedFFmpeg
	s.ffmpegMu.RUnlock()
	if cached != nil {
		return *cached
	}

	s.ffmpegMu.Lock()
	defer s.ffmpegMu.Unlock()
	if s.cachedFFmpeg != nil {
		return *s.cachedFFmpeg
	}
	status := util.DetectFFmpeg(s.Store.PersistDir())
	s.cachedFFmpeg = &status
	return status
}

// RecheckFFmpeg 清除缓存并重新检测 ffmpeg（用户安装后点击"重新检测"）
func (s *SettingService) RecheckFFmpeg() model.FFmpegStatus {
	s.ffmpegMu.Lock()
	s.cachedFFmpeg = nil
	s.ffmpegMu.Unlock()
	return s.GetFFmpegStatus()
}

// FFmpegBinPath 返回 ffmpeg 可执行文件路径（供 MediaService 等内部调用）。
// 不可用时返回空字符串。
func (s *SettingService) FFmpegBinPath() string {
	status := s.GetFFmpegStatus()
	if status.Available {
		return status.BinPath
	}
	return ""
}
