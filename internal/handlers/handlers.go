package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	nd "github.com/fedgolang/go_final_project/internal/lib/nextdate"
	"github.com/fedgolang/go_final_project/internal/storage"
)

type Response struct {
	ID  int    `json:"id,omitempty"`
	Err string `json:"error,omitempty"`
}

func PostTask(s *storage.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		task := storage.Task{}
		resp := Response{}
		var buf bytes.Buffer

		// Читаем данные из тела и запишем в буфер
		_, err := buf.ReadFrom(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest) // Вернём 400 если не смогли прочитать боди
			return
		}

		// Расшифруем JSON
		err = json.Unmarshal(buf.Bytes(), &task)
		if err != nil {
			resp.Err = "Ошибка десериализации JSON"
		}

		// Проверим, что заголовок не пустой
		if task.Title == "" {
			resp.Err = "Не указан заголовок задачи" // 400 пришел пустой title
		}

		// Проверим, что дата не пустая
		if task.Date == "" {
			task.Date = time.Now().Format("20060102")
		}

		date, err := time.Parse("20060102", task.Date)
		if err != nil {
			resp.Err = fmt.Sprint(err)
		}

		// Проверим, что дата не прошлое, если повторения нет
		if task.Repeat == "" {
			if date.Before(time.Now()) {
				task.Date = time.Now().Format("20060102")
			}
		} else { // Если не пустое повторение, вычислим следующую дату из NextDate()
			task.Date, err = nd.NextDate(time.Now(), task.Date, task.Repeat)
			if err != nil {
				resp.Err = "Дата представлена в формате, отличном от 20060102"
			}
		}

		// Запись проводит только в случае отсутствия ошибки
		if resp.Err == "" {
			id, err := s.PostTask(task)
			if err != nil {
				if err == fmt.Errorf("неверный формат repeat") {
					resp.Err = "Правило повторения указано в неправильном формате"
				} else {
					resp.Err = fmt.Sprint(err)
				}
			}
			resp.ID = id
		}
		JSONResp, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest) // 400 не смогли записать ответ
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		w.Write(JSONResp)

	}
}
