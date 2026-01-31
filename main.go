package main

import (
	"context"
	"embed"
	"log"

	"flash-db/internal/database"
	"flash-db/internal/storage"
	"flash-db/services"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

const encryptionKey = "FlashDB-AES256-Secret-Key-2024"

func main() {
	// Initialize storage
	store, err := storage.New(encryptionKey)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize connection pool
	pool := database.NewPool()
	defer pool.Close()

	// Create services
	connectionService := services.NewConnectionService(store, pool)
	explorerService := services.NewExplorerService(pool)
	queryService := services.NewQueryService(pool)
	fileService := services.NewFileService()

	// Create application with options
	err = wails.Run(&options.App{
		Title:     "FlashDB",
		Width:     1280,
		Height:    800,
		MinWidth:  1024,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup: func(ctx context.Context) {
			connectionService.SetContext(ctx)
			explorerService.SetContext(ctx)
			queryService.SetContext(ctx)
			fileService.SetContext(ctx)
		},
		Bind: []interface{}{
			connectionService,
			explorerService,
			queryService,
			fileService,
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            true,
				UseToolbar:                 false,
			},
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})

	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
