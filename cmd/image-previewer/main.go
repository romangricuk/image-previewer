package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/romangricuk/image-previewer/internal/app"
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

	if err := application.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Failed to run application: %v", err)
	}
}
