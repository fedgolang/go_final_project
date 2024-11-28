package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	nd "github.com/fedgolang/go_final_project/internal/lib/nextdate"
	"github.com/fedgolang/go_final_project/internal/storage"
	"github.com/golang-jwt/jwt"
)

var (
	limitForTasks = 50                                                                         // Максимальное кол-во возвращаемых тасков в GetTasks
	JWTSecret     = []byte("69612fb755d66b4a275896981874c46210f4afbac7673bcb0ce40d3c6a0160d5") // Секрет для токена
)

type SignInRequest struct {
	Password string `json:"password"`
}

type SignInResponse struct {
	Token string `json:"token,omitempty"`
	Err   string `json:"error,omitempty"`
}

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

// Для реализации всех хендлеров будем пользоваться middleware

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
			// Доп проверка, если таска добавляется сегодня с датой > сегодня
			// То не учитываем NextDate и регаем таску с датой = дате создания
			if date.Truncate(24 * time.Hour).Before(time.Now().Truncate(24 * time.Hour)) {
				task.Date, err = nd.NextDate(time.Now(), task.Date, task.Repeat)
				if err != nil {
					resp.Err = fmt.Sprint(err)
				}
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

// Ручка, чтобы отдельно дёрнуть проверку даты
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
				} else {
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
		}

		// Проверим, нет ли ошибок в процессе
		// Если есть, вернём ошибку, если нет, вернём таски
		if resp.Err != "" {
			JSONResp, err = json.Marshal(resp)
			if err != nil {
				http.Error(w, "Ошибка при формировании ответа", http.StatusInternalServerError) // 500 не смогли записать ответ
				return
			}
		} else {
			JSONResp, err = json.Marshal(tasks)
			if err != nil {
				http.Error(w, "Ошибка при формировании ответа", http.StatusInternalServerError) // 500 не смогли записать ответ
				return
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(JSONResp)
	}
}

// Хендлер отвечает за поиск по ID таски в БД
func GetDataForEdit(s *storage.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var JSONResp []byte
		task := storage.TaskNoEmpty{}
		resp := Response{}

		// Достанем с урла ID таски
		taskID := r.URL.Query().Get("id")

		// Если запрос пришел без параметра id, не запускаем запрос к БД
		// Прокидываем ошибку
		if taskID == "" {
			resp.Err = "Не указан идентификатор"
		} else {
			// Если же id есть, идём в БД искать таску, она должна быть одна
			err := s.GetTaskByID(taskID).Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
			// Если по необходимому id нет задач, запишем ошибку
			if err == sql.ErrNoRows {
				resp.Err = "Задача не найдена"
			}
		}

		// Проверим, есть ли ошибки
		// Если есть, вернём JSON с ошибкой
		if resp.Err != "" {
			var err error
			JSONResp, err = json.Marshal(resp)
			if err != nil {
				http.Error(w, "Ошибка при формировании ответа", http.StatusInternalServerError) // 500 не смогли записать ответ
				return
			}
		} else { // Если нет, то вернём найденную таску
			var err error
			JSONResp, err = json.Marshal(task)
			if err != nil {
				http.Error(w, "Ошибка при формировании ответа", http.StatusInternalServerError) // 500 не смогли записать ответ
				return
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(JSONResp)
	}
}

// Хендлер отвечает за редактирование таски
func PutDataByID(s *storage.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var JSONResp []byte
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

		// Проверим, что формат даты ожидаемый
		_, err = time.Parse("20060102", task.Date)
		if err != nil {
			resp.Err = fmt.Sprint(err)
		}

		// Так как проверка repeat у нас внутри NextDate
		// Запустим функцию без сохранения переменной для проверки на ошибки
		if task.Repeat != "" {
			_, err = nd.NextDate(time.Now(), task.Date, task.Repeat)
			if err != nil {
				resp.Err = fmt.Sprint(err)
			}
		}

		// Отправим таску на апдейт в БД, если ошибок нет
		if resp.Err == "" {
			err := s.EditTask(task)
			if err == fmt.Errorf("нет записи") {
				resp.Err = "Задача не найдена"
			} else if err != nil {
				resp.Err = "Возникла проблема с изменением задачи"
			}
		}

		// Так как нам не надо возвращать таску после апдейта
		// Просто сериализуем структуру resp
		// Она нам и вернёт ошибку при наличии, либо пустой ответ при успехе
		JSONResp, err = json.Marshal(resp)
		if err != nil {
			http.Error(w, "Ошибка при формировании ответа", http.StatusInternalServerError) // 500 не смогли записать ответ
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(JSONResp)
	}
}

// Хендлер отвечает за обработку таски как выполненной
func TaskDone(s *storage.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var JSONResp []byte
		task := storage.Task{}
		resp := Response{}

		taskID := r.URL.Query().Get("id")

		// Если запрос пришел без параметра id, не запускаем запрос к БД
		// Прокидываем ошибку
		if taskID == "" {
			resp.Err = "Не указан идентификатор"
		} else {
			// Если же id есть, идём в БД искать таску, она должна быть одна
			err := s.GetTaskByID(taskID).Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
			// Если по необходимому id нет задач, запишем ошибку
			if err == sql.ErrNoRows {
				resp.Err = "Задача не найдена"
			}
		}

		// Если нет ошибок, обращаемся к БД
		if resp.Err == "" {
			// Если правил повторения нет, просто удаляем задачу
			if task.Repeat == "" {
				err := s.DeleteTaskByID(taskID)
				if err != nil {
					resp.Err = fmt.Sprint(err)
				}
			} else {
				// Если есть правило, вычислим дату
				nextDate, err := nd.NextDate(time.Now(), task.Date, task.Repeat)
				if err != nil {
					resp.Err = fmt.Sprint(err)
				} else {
					// Проблем при вычислении даты не возникло, присвоим новую дату и отправим на изменение
					task.Date = nextDate
					err = s.EditTask(task)
					if err != nil {
						resp.Err = fmt.Sprint(err)
					}
				}
			}
		}
		// Подготовим ответ, он либо пустой, либо ошибка
		JSONResp, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, "Ошибка при формировании ответа", http.StatusInternalServerError) // 500 не смогли записать ответ
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(JSONResp)
	}
}

// Хендлер отвечает за удаление таски из БД
func DeleteTask(s *storage.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var JSONResp []byte
		resp := Response{}

		taskID := r.URL.Query().Get("id")

		if taskID == "" {
			resp.Err = "Не указан идентификатор"
		} else {
			err := s.DeleteTaskByID(taskID)
			if err != nil {
				resp.Err = fmt.Sprint(err)
			}
		}

		// Подготовим ответ, он либо пустой, либо ошибка
		JSONResp, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, "Ошибка при формировании ответа", http.StatusInternalServerError) // 500 не смогли записать ответ
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(JSONResp)
	}
}

// Хендлер аутентификации
func SignInHandler(w http.ResponseWriter, r *http.Request) {
	// Для аутентификации поставим проверку на метод
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Достанем пароль
	var buf bytes.Buffer
	var req SignInRequest
	var resp SignInResponse

	// Читаем данные из тела и запишем в буфер
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest) // Вернём 400 если не смогли прочитать боди
		return
	}

	// Расшифруем JSON
	err = json.Unmarshal(buf.Bytes(), &req)
	if err != nil {
		resp.Err = "Ошибка десериализации JSON"
	}

	// Проверим, есть ли в окружении пароль
	envPass := os.Getenv("TODO_PASSWORD")

	// Совпадают ли пароли из тела и окружения
	if req.Password != envPass {
		resp.Err = "Неверный пароль"
	}

	if resp.Err == "" {
		// Формируем токен из секрета
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"passwordHash": fmt.Sprintf("%x", JWTSecret),
			"exp":          time.Now().Add(8 * time.Hour).Unix(),
		})

		// Подписываем токен
		tokenString, err := token.SignedString(JWTSecret)
		if err != nil {
			resp.Err = "Ошибка при создании токена"
		}
		resp.Token = tokenString
	}

	if resp.Token != "" {
		// Подготовим ответ, если токен сформировали
		JSONResp, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, "Ошибка при формировании ответа", http.StatusInternalServerError) // 500 не смогли записать ответ
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(JSONResp)
		// Специальный вывод токена для простоты его отслеживания
		fmt.Println(resp.Token)
	} else {
		// Подготовим ответ, если токена нет, возвращаем ошибку
		JSONResp, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, "Ошибка при формировании ответа", http.StatusInternalServerError) // 500 не смогли записать ответ
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(JSONResp)
	}
}

// middleware для аутентификации
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем куку
		cookie, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "auth required", http.StatusUnauthorized)
			return
		}

		// Парсим токен
		token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("wrong sign method: %v", t.Header["alg"])
			}
			return JWTSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "auth required", http.StatusUnauthorized)
			return
		}

		// Проверяем хэш пароля
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || claims["passwordHash"] != fmt.Sprintf("%x", JWTSecret) {
			http.Error(w, "auth required", http.StatusUnauthorized)
			return
		}

		// Если всё хорошо, вызываем следующий обработчик
		next(w, r)
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
