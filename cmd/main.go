package main

import (
	"net/http"

	"github.com/fedgolang/go_final_project/internal/config"
	"github.com/fedgolang/go_final_project/internal/handlers"
	"github.com/fedgolang/go_final_project/internal/storage"
	"github.com/go-chi/chi"

	_ "modernc.org/sqlite"
)

func main() {
	r := chi.NewRouter()
	cfg := config.Load()

	// Открываем коннект к БД
	s, db := storage.NewScheduler(cfg.DBPath)
	defer db.Close() // Закроем коннект по окончанию работы

	// На chi не получилось просто прокинуть FileServer, без StripPrefix он не видит css и js
	r.Handle("/*", http.StripPrefix("/", http.FileServer(http.Dir(cfg.WebDir))))

	// Хендлер на добавление таски
	r.Post("/api/task", handlers.PostTask(s))

	// Хендлер для вычисления следующей даты
	r.Get("/api/nextdate", handlers.NextDateHand)

	// Хендлер для вывода ближайших задач
	r.Get("/api/tasks", handlers.GetTasks(s))

	http.ListenAndServe(cfg.HTTPAdress, r)

}
