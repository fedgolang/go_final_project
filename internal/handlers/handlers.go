package handlers

import (
	"net/http"

	"github.com/fedgolang/go_final_project/internal/storage"
)

type Task struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment,omitempty"`
	Repeat  string `json:"repeat"`
}

func PostTask(s *storage.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}
