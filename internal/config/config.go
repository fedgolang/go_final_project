package config

import (
	"fmt"
	"os"
)

// Создадим структуру описывающую наш конфиг
type Config struct {
	HTTPAdress string
	DBPath     string
	WebDir     string
}

func Load() *Config {
	var cfg Config

	// Прокинем проверки, если нет переменных окружения, то запуск локальный
	// Звёздочка, достанем порт из енв
	todoPort := os.Getenv("TODO_PORT")
	if todoPort == "" { // Если прокинуть не смогли, запишем порт напрямую
		cfg.HTTPAdress = fmt.Sprintf("localhost:%s", "7540")
	} else { // Если получилось прокинуть, запускаемся по переменной окружения
		cfg.HTTPAdress = fmt.Sprintf("0.0.0.0:%s", todoPort)
	}

	// Так как в контейнере и в локальном запуске БД по одному пути
	// Запишем напрямую
	cfg.DBPath = "./scheduler.db"

	// Если енв пустой, то запуск локальный, ищем web в корне
	// Если контейнер, web в /app/web
	webDir := os.Getenv("TODO_WEBDIR")
	if webDir == "" {
		cfg.WebDir = "./web"
	} else {
		cfg.WebDir = webDir
	}

	return &cfg
}
