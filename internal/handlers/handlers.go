package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	nd "github.com/fedgolang/go_final_project/internal/lib/nextdate"
	"github.com/fedgolang/go_final_project/internal/storage"
)

var (
	limitForTasks = 50 // Максимальное кол-во возвращаемых тасков в GetTasks
)

// Структура для ответа после POST
type Response struct {
	ID  int    `json:"id,omitempty"`
	Err string `json:"error,omitempty"`
}

// Структура для ответа GET запроса
// Так как задач может быть много, то у нас массив респонсов
type TasksResponse struct {
	Tasks []storage.TaskNoEmpty `json:"tasks"`
}

// Функция, возвращающая нам хендлер, чтобы тут работать с БД
// Хендлер отвечает за добавление таски в БД
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
				resp.Err = fmt.Sprint(err)
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

		// Сериализируем ответ в JSON
		JSONResp, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, "Ошибка при формировании ответа", http.StatusInternalServerError) // 500 не смогли записать ответ
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		w.Write(JSONResp)

	}
}

func NextDateHand(w http.ResponseWriter, r *http.Request) {
	// Объявим переменные и достанем параметры
	resp := ""
	textError := ""
	now := r.URL.Query().Get("now")
	nowDate, err := time.Parse("20060102", now)
	if err != nil {
		textError = "Неверный формат даты"
	}
	date := r.URL.Query().Get("date")
	repeat := r.URL.Query().Get("repeat")

	// Вычисляем следующую дату
	nextDate, err := nd.NextDate(nowDate, date, repeat)
	if err != nil {
		textError = "Неверный формат даты"
	}

	// Проверяем на наличие ошибки
	if textError != "" {
		resp = textError
	} else {
		resp = nextDate
	}

	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(resp))
}

// Функция, возвращающая нам хендлер, чтобы тут работать с БД
// Хендлер отвечает за возвращение набора тасок
func GetTasks(s *storage.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Объявим пустой слайс tasks, для случая если приходит пустой ответ из БД
		tasks := TasksResponse{Tasks: []storage.TaskNoEmpty{}}
		task := storage.TaskNoEmpty{}
		resp := Response{}
		var JSONResp []byte
		today := time.Now().Format(`20060102`)

		// Запросим из БД нужные таски
		dbRows, err := s.GetTasks(limitForTasks, today)
		if err != nil {
			resp.Err = "ошибка при получении данных"
		}

		// Попробуем достать GET параметр search
		search := r.URL.Query().Get("search")
		// Проверим, не дата ли нам пришла в поиске
		searchDate, okDate := validateAndFormatDate(search)

		// Если строки из БД есть, то работаем с ними
		if dbRows != nil {
			for dbRows.Next() {
				err := dbRows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
				if err != nil {
					resp.Err = "ошибка при чтении данных"
				}

				// Если в поиске дата, ищем по дате
				if okDate {
					if searchDate == task.Date {
						tasks.Tasks = append(tasks.Tasks, task)
					}
				} else if search != "" { // Если не дата, будем фильтровать по комменту и заголовку
					if strings.Contains(task.Comment, search) || strings.Contains(task.Title, search) {
						tasks.Tasks = append(tasks.Tasks, task)
					}
				} else { // Если параметра нет, выводим всё
					tasks.Tasks = append(tasks.Tasks, task)
				}
			}
		}

		// Проверим, нет ли ошибок в процессе
		// Если есть, вернём ошибку, если нет, вернём таски
		if resp.Err != "" {
			JSONResp, err = json.Marshal(resp)
		} else {
			JSONResp, err = json.Marshal(tasks)
		}
		if err != nil {
			http.Error(w, "Ошибка при формировании ответа", http.StatusInternalServerError) // 500 не смогли записать ответ
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(JSONResp)
	}
}

// Проверка на соответствие формату для поиска по дате и форматирование к 20060102
func validateAndFormatDate(s string) (string, bool) {
	r := regexp.MustCompile(`^\d{2}\.\d{2}\.\d{4}$`)
	if r.MatchString(s) {
		t, err := time.Parse("02.01.2006", s)
		if err != nil {
			return "", false
		}
		return t.Format("20060102"), true
	}
	return "", false
}
