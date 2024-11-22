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
	err := os.Setenv("TODO_PORT", "8080")
	if err != nil { // Если прокинуть не смогли, запишем порт напрямую
		cfg.HTTPAdress = fmt.Sprintf("localhost:%s", "8080")
	} else { // Если получилось прокинуть, запускаемся по переменной окружения
		cfg.HTTPAdress = fmt.Sprintf("localhost:%s", os.Getenv("TODO_PORT"))
	}

	// Звёздочка, прокинем путь к БД в енв
	err = os.Setenv("TODO_DBFILE", "../scheduler.db")
	if err != nil { // Если прокинуть не смогли, запишем путь напрямую
		cfg.DBPath = "../scheduler.db"
	} else { // Если получилось прокинуть, впишем в путь переменную окружения
		cfg.DBPath = os.Getenv("TODO_DBFILE")
	}

	cfg.WebDir = "../web"

	return &cfg
}
