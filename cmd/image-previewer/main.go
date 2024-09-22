package main

import (
	"flag"
	"github.com/romangricuk/image-previewer/internal/app"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	configPath := flag.String("config", "/etc/remains-loader/config.yaml", "путь к файлу конфигурации")
	flag.Parse()

	application, err := app.NewApplication(*configPath)
	if err != nil {
		log.Fatalf("Error on create application: %v", err)
	}

	// Обработка сигналов завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		if err := application.Shutdown(); err != nil {
			application.Logger.Errorf("Error during shutdown: %v", err)
		}
	}()

	if err := application.Run(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to run application: %v", err)
	}
}
