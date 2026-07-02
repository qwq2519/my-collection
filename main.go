package main

import (
	"embed"
	"log"
	"log/slog"
	"os"

	"collections/internal/service"
	"collections/internal/store"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	})))

	// 1. 初始化 Store（BuntDB + Bleve）
	s, err := store.New("persist")
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	defer s.Close()

	// 2. 创建 Service，注入 Store
	urlService := &service.URLService{Store: s}
	noteService := &service.NoteService{Store: s}
	uploadService := &service.UploadService{Store: s}
	settingService := &service.SettingService{Store: s}
	mediaService := &service.MediaService{Store: s, FFmpegPathFunc: settingService.FFmpegBinPath}

	// 3. 创建 Wails 应用
	app := application.New(application.Options{
		Name:        "资料收藏夹",
		Description: "个人资料收藏与管理工具",
		Services: []application.Service{
			application.NewService(urlService),
			application.NewService(noteService),
			application.NewService(uploadService),
			application.NewService(settingService),
			application.NewService(mediaService),
		},
		Assets: application.AssetOptions{
			Handler: service.NewAssetHandler(application.AssetFileServerFS(assets), "persist"),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// 4. 启动扫描协调器（单 goroutine 串行处理所有扫描请求）
	mediaService.StartScanner()

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:    "资料收藏夹",
		Width:    1100,
		Height:   720,
		MinWidth: 780,
		MinHeight: 500,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 36,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
