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

	// Звёздочка, прокинем порт в енв
	todoPort := os.Getenv("TODO_PORT")
	if todoPort != "8080" { // Если прокинуть не смогли, запишем порт напрямую
		cfg.HTTPAdress = fmt.Sprintf("localhost:%s", "8080")
	} else { // Если получилось прокинуть, запускаемся по переменной окружения
		cfg.HTTPAdress = fmt.Sprintf("0.0.0.0:%s", os.Getenv("TODO_PORT"))
	}

	// Звёздочка, прокинем путь к БД в енв
	todoDBfile := os.Getenv("TODO_DBFILE")
	if todoDBfile != "/app/scheduler.db" { // Если прокинуть не смогли, запишем путь напрямую
		cfg.DBPath = "../scheduler.db"
	} else { // Если получилось прокинуть, впишем в путь переменную окружения
		cfg.DBPath = os.Getenv("TODO_DBFILE")
	}

	cfg.WebDir = "/app/web"

	return &cfg
}
