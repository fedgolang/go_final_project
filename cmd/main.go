package main

import (
	"log"
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
	r.Post("/api/task", handlers.AuthMiddleware(handlers.PostTask(s)))

	// Хендлер для вычисления следующей даты
	r.Get("/api/nextdate", handlers.NextDateHand)

	// Хендлер для вывода ближайших тасок
	r.Get("/api/tasks", handlers.AuthMiddleware(handlers.GetTasks(s)))

	// Хендлер для вывода таски по ID
	r.Get("/api/task", handlers.AuthMiddleware(handlers.GetDataForEdit(s)))

	// Хендлер для редактирования таски
	r.Put("/api/task", handlers.AuthMiddleware(handlers.PutDataByID(s)))

	// Хендлер для выполнения таски
	r.Post("/api/task/done", handlers.AuthMiddleware(handlers.TaskDone(s)))

	// Хендлер для удаления таски
	r.Delete("/api/task", handlers.AuthMiddleware(handlers.DeleteTask(s)))

	// Хендлер для аутентификации
	r.Post("/api/signin", handlers.SignInHandler)

	log.Printf("Сервис запущен по адресу: %s", cfg.HTTPAdress)

	http.ListenAndServe(cfg.HTTPAdress, r)

}
