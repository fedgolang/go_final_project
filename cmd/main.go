package main

import (
	"net/http"

	"github.com/fedgolang/go_final_project/internal/config"
	"github.com/fedgolang/go_final_project/internal/storage"
	"github.com/go-chi/chi"

	_ "modernc.org/sqlite"
)

func main() {
	r := chi.NewRouter()
	cfg := config.Load()

	// Открываем коннект к БД
	_, db := storage.NewScheduler(cfg.DBPath)
	defer db.Close() // Закроем коннект по окончанию работы

	// На chi не получилось просто прокинуть FileServer, без StripPrefix он не видит css и js
	r.Handle("/*", http.StripPrefix("/", http.FileServer(http.Dir(cfg.WebDir))))

	r.Post("/api/task")

	http.ListenAndServe(cfg.HTTPAdress, r)

}
