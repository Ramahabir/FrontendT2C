package main

import (
	"context"
	"embed"
	"log"
	"myproject/api"
	"myproject/database"
	"net/http"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Initialize database
	err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.CloseDB()

	// Start API server in separate goroutine
	go startAPIServer()

	// Create Wails app instance
	app := NewApp()

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "Trash 2 Cash - Station Control",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown: func(ctx context.Context) {
			database.CloseDB()
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

func startAPIServer() {
	router := api.SetupRouter()
	
	log.Println("Starting API server on http://localhost:8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal("API server failed:", err)
	}
}

