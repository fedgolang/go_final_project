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
		cfg.HTTPAdress = fmt.Sprintf("0.0.0.0:%s", os.Getenv("TODO_PORT"))
	}

	// Звёздочка, достанем путь к БД из енв
	todoDBfile := os.Getenv("TODO_DBFILE")
	if todoDBfile == "" { // Если прокинуть не смогли, запишем путь напрямую
		cfg.DBPath = "./scheduler.db"
	} else { // Если получилось прокинуть, впишем в путь переменную окружения
		cfg.DBPath = os.Getenv("TODO_DBFILE")
	}

	// Если енв пустой, то запуск локальный, ищем web в корне
	// Если контейнер, web в /app/web
	webDir := os.Getenv("TODO_WEBDIR")
	if webDir == "" {
		cfg.WebDir = "./web"
	} else {
		cfg.WebDir = "/app/web"
	}

	JWTPass := os.Getenv("TODO_PASSWORD")
	if JWTPass == "" {
		_ = os.Setenv("TODO_PASSWORD", "JWT_PASS789")
	}

	return &cfg
}
