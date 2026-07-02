package service

import (
	"log/slog"
	"sync"

	"collections/internal/model"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// scanRequest 扫描请求：FolderIDs 为空表示扫描所有文件夹
type scanRequest struct {
	FolderIDs []string
}

// scanState 扫描协调器内部状态（由 worker goroutine 独占写，GetScanStatus 通过 mu 读）
type scanState struct {
	mu         sync.RWMutex
	scanning   bool
	folderID   string
	folderName string
	scanned    int
	total      int
	queued     int
}

// initScanner 初始化扫描通道，启动唯一 worker goroutine。
// 必须在 main.go 中应用启动后调用一次。
func (m *MediaService) initScanner() {
	m.scanCh = make(chan scanRequest, 64)
	m.state = &scanState{}
	go m.scanWorker()
}

// enqueue 向扫描队列提交请求（非阻塞）。
// scanCh 为 nil 时静默跳过（测试模式）。队列满时丢弃请求并记录日志。
func (m *MediaService) enqueue(req scanRequest) {
	if m.scanCh == nil {
		return
	}
	select {
	case m.scanCh <- req:
		m.state.mu.Lock()
		m.state.queued++
		m.state.mu.Unlock()
	default:
		slog.Warn("scan queue full, request dropped", "folder_ids", req.FolderIDs)
	}
}

// scanWorker 唯一扫描 goroutine，串行处理所有扫描请求
func (m *MediaService) scanWorker() {
	for req := range m.scanCh {
		m.state.mu.Lock()
		m.state.queued--
		m.state.mu.Unlock()

		folderIDs := req.FolderIDs
		if len(folderIDs) == 0 {
			folders, err := m.Store.ListFolders()
			if err != nil {
				slog.Warn("scan worker: list folders failed", "err", err)
				continue
			}
			for _, f := range folders {
				folderIDs = append(folderIDs, f.ID)
			}
		}

		for _, fid := range folderIDs {
			m.executeScan(fid)
		}
	}
}

// executeScan 执行单个文件夹扫描并推送进度/完成事件
func (m *MediaService) executeScan(folderID string) {
	folder, err := m.Store.GetFolder(folderID)
	if err != nil {
		slog.Warn("scan worker: get folder failed", "folder_id", folderID, "err", err)
		return
	}

	m.state.mu.Lock()
	m.state.scanning = true
	m.state.folderID = folderID
	m.state.folderName = folder.Name
	m.state.scanned = 0
	m.state.total = 0
	m.state.mu.Unlock()

	m.emitProgress(folderID, folder.Name, 0, 0)

	result, err := m.scanFolderInternal(folderID, folder)
	if err != nil {
		slog.Warn("scan worker: scan failed", "folder_id", folderID, "err", err)
		result = &model.ScanComplete{FolderID: folderID}
	}

	m.state.mu.Lock()
	m.state.scanning = false
	m.state.folderID = ""
	m.state.folderName = ""
	m.state.mu.Unlock()

	m.emitComplete(result)
}

// emitProgress 推送 media:scan-progress 事件
func (m *MediaService) emitProgress(folderID, folderName string, scanned, total int) {
	app := application.Get()
	if app == nil {
		return
	}
	app.Event.Emit("media:scan-progress", model.ScanProgress{
		FolderID:   folderID,
		FolderName: folderName,
		Scanned:    scanned,
		Total:      total,
	})
}

// emitComplete 推送 media:scan-complete 事件
func (m *MediaService) emitComplete(result *model.ScanComplete) {
	app := application.Get()
	if app == nil {
		return
	}
	app.Event.Emit("media:scan-complete", *result)
}

// reportProgress 在文件处理循环中调用，更新内部状态并推送事件。
// state 为 nil 时静默跳过（测试模式）。
func (m *MediaService) reportProgress(folderID, folderName string, scanned, total int) {
	if m.state == nil {
		return
	}
	m.state.mu.Lock()
	m.state.scanned = scanned
	m.state.total = total
	m.state.mu.Unlock()
	m.emitProgress(folderID, folderName, scanned, total)
}
