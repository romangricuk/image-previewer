package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/romangricuk/image-previewer/internal/config"
	"github.com/romangricuk/image-previewer/internal/handler"
	"github.com/romangricuk/image-previewer/internal/logger"
)

type Application struct {
	Config *config.Config
	Logger logger.Logger // Используем интерфейс logger.Logger
	Server *http.Server
}

func NewApplication(configPath string) (*Application, error) {
	// Загрузка конфигурации
	cfg, err := config.Load(configPath)
	if err != nil {
		err = fmt.Errorf("on config load: %w", err)
		return nil, err
	}
	// Инициализация логгера
	log := logger.New(cfg)

	// Создание экземпляра приложения
	app := &Application{
		Config: cfg,
		Logger: log,
	}

	// Инициализация маршрутов
	app.initRoutes()

	return app, nil
}

func (app *Application) Run() error {
	app.Logger.Infof("Server is running on port %s", app.Config.AppPort)
	return app.Server.ListenAndServe()
}

func (app *Application) Shutdown() error {
	app.Logger.Info("Shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), app.Config.ShutdownTimeout)
	defer cancel()
	return app.Server.Shutdown(ctx)
}

func (app *Application) initRoutes() {
	// Создаем HTTP-обработчики
	mux := http.NewServeMux()
	mux.HandleFunc("/fill/", handler.NewImageHandler(app.Config, app.Logger))

	// Настраиваем сервер
	app.Server = &http.Server{
		Addr:              ":" + app.Config.AppPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
