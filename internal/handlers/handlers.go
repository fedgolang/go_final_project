package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	nd "github.com/fedgolang/go_final_project/internal/lib/nextdate"
	"github.com/fedgolang/go_final_project/internal/storage"
	"github.com/golang-jwt/jwt"
)

var (
	limitForTasks = 50                                                                         // Максимальное кол-во возвращаемых тасков в GetTasks
	JWTSecret     = []byte("69612fb755d66b4a275896981874c46210f4afbac7673bcb0ce40d3c6a0160d5") // Секрет для токена
	envPass       = os.Getenv("TODO_PASSWORD")                                                 //
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
			prepareJSONResp(w, 400, resp)
			return
		}

		// Проверим, что заголовок не пустой
		if task.Title == "" {
			resp.Err = "Не указан заголовок задачи" // 400 пришел пустой title
			prepareJSONResp(w, 400, resp)
			return
		}

		// Проверим, что дата не пустая
		if task.Date == "" {
			task.Date = time.Now().Format("20060102")
		}

		date, err := time.Parse("20060102", task.Date)
		if err != nil {
			resp.Err = "Неверный формат даты, ожидается ГГГГММДД"
			prepareJSONResp(w, 400, resp)
			return
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
					prepareJSONResp(w, 400, resp)
					return
				}
			}
		}

		// Проводим запись в БД
		id, err := s.PostTask(task)
		if err != nil {
			// Так как описывает ошибку в самом методе, просто запишем её тут
			resp.Err = fmt.Sprint(err)
			prepareJSONResp(w, 400, resp)
			return
		}
		resp.ID = id

		prepareJSONResp(w, 201, resp)
	}
}

// Ручка, чтобы отдельно дёрнуть проверку даты
func NextDateHand(w http.ResponseWriter, r *http.Request) {
	// Объявим переменные и достанем параметры
	resp := Response{}
	now := r.URL.Query().Get("now")
	nowDate, err := time.Parse("20060102", now)
	if err != nil {
		resp.Err = "Неверный формат даты"
		prepareJSONResp(w, 400, resp)
		return
	}
	date := r.URL.Query().Get("date")
	repeat := r.URL.Query().Get("repeat")

	// Вычисляем следующую дату
	nextDate, err := nd.NextDate(nowDate, date, repeat)
	if err != nil {
		resp.Err = "Неверный формат даты"
		prepareJSONResp(w, 400, resp)
		return
	}

	// Здесь вызовем отдельно запись ответа, так как у нас в ответе строка
	// И требуется записать ее без кавычек
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(nextDate))
}

// Хендлер отвечает за возвращение набора тасок
func GetTasks(s *storage.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Объявим пустой слайс tasks, для случая если приходит пустой ответ из БД
		tasks := TasksResponse{Tasks: []storage.TaskNoEmpty{}}
		resp := Response{}
		today := time.Now().Format(`20060102`)

		// Попробуем достать GET параметр search
		search := r.URL.Query().Get("search")
		// Проверим, не дата ли нам пришла в поиске
		searchDate, okDate := validateAndFormatDate(search)

		var dbTasks []storage.TaskNoEmpty
		var err error

		// В зависимости от полученных данных по поиску, запустим функции для БД
		if okDate {
			dbTasks, err = s.GetTasksByDate(searchDate)
		} else if search != "" { // Если не дата, ищем по тексту
			dbTasks, err = s.GetTasksBySearch(limitForTasks, today, search)
		} else { // Если параметра нет, выводим ближайшие задачи
			dbTasks, err = s.GetTasks(limitForTasks, today)
		}

		if err != nil {
			resp.Err = fmt.Sprintf("ошибка при запросе задач: %s", err)
			prepareJSONResp(w, 400, resp)
			return
		}

		tasks.Tasks = dbTasks

		prepareJSONResp(w, 200, tasks)
	}
}

// Хендлер отвечает за поиск по ID таски в БД
func GetDataForEdit(s *storage.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		task := storage.Task{}
		resp := Response{}

		// Достанем с урла ID таски
		taskID := r.URL.Query().Get("id")

		// Если запрос пришел без параметра id, не запускаем запрос к БД
		// Прокидываем ошибку
		if taskID == "" {
			resp.Err = "Не указан идентификатор"
			prepareJSONResp(w, 400, resp)
			return
		}
		// Если же id есть, идём в БД искать таску, она должна быть одна
		task, err := s.GetTaskByID(taskID)
		if err != nil {
			// Ошибку описываем внутри метода
			resp.Err = fmt.Sprint(err)
			prepareJSONResp(w, 400, resp)
			return
		}

		prepareJSONResp(w, 200, task)
	}
}

// Хендлер отвечает за редактирование таски
func PutDataByID(s *storage.Scheduler) http.HandlerFunc {
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
			prepareJSONResp(w, 400, resp)
			return
		}

		// Проверим, что заголовок не пустой
		if task.Title == "" {
			resp.Err = "Не указан заголовок задачи"
			prepareJSONResp(w, 400, resp)
			return
		}

		// Проверим, что дата не пустая
		if task.Date == "" {
			task.Date = time.Now().Format("20060102")
		}

		// Проверим, что формат даты ожидаемый
		_, err = time.Parse("20060102", task.Date)
		if err != nil {
			resp.Err = "Неверный формат даты ожидается ГГГГММДД"
			prepareJSONResp(w, 400, resp)
			return
		}

		// Так как проверка repeat у нас внутри NextDate
		// Запустим функцию без сохранения переменной для проверки на ошибки
		if task.Repeat != "" {
			_, err = nd.NextDate(time.Now(), task.Date, task.Repeat)
			if err != nil {
				resp.Err = "Неверный формат repeat"
				prepareJSONResp(w, 400, resp)
				return
			}
		}

		// Отправим таску на апдейт в БД
		err = s.EditTask(task)
		if errors.Is(err, sql.ErrNoRows) {
			resp.Err = "Задача не найдена"
			prepareJSONResp(w, 400, resp)
			return
		}
		if err != nil {
			resp.Err = "Возникла проблема с изменением задачи"
			prepareJSONResp(w, 400, resp)
			return
		}

		prepareJSONResp(w, 200, resp)
	}
}

// Хендлер отвечает за обработку таски как выполненной
func TaskDone(s *storage.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		task := storage.Task{}
		resp := Response{}

		taskID := r.URL.Query().Get("id")

		// Если запрос пришел без параметра id, не запускаем запрос к БД
		// Прокидываем ошибку
		if taskID == "" {
			resp.Err = "Не указан идентификатор"
			prepareJSONResp(w, 400, resp)
			return
		}

		// Если же id есть, идём в БД искать таску, она должна быть одна
		task, err := s.GetTaskByID(taskID)
		if err != nil {
			// Ошибку описываем внутри метода
			resp.Err = fmt.Sprint(err)
			prepareJSONResp(w, 400, resp)
			return
		}

		// Если правил повторения нет, просто удаляем задачу
		if task.Repeat == "" {
			err := s.DeleteTaskByID(taskID)
			if err != nil {
				resp.Err = "Не удалось удалить задачу"
				prepareJSONResp(w, 400, resp)
				return
			}
		} else {
			// Если есть правило, вычислим дату
			nextDate, err := nd.NextDate(time.Now(), task.Date, task.Repeat)
			if err != nil {
				resp.Err = fmt.Sprint(err)
				prepareJSONResp(w, 400, resp)
				return
			}
			// Проблем при вычислении даты не возникло, присвоим новую дату и отправим на изменение
			task.Date = nextDate
			err = s.EditTask(task)
			if err != nil {
				// Ошибку описываем в методе, тут просто запишем
				resp.Err = fmt.Sprint(err)
				prepareJSONResp(w, 400, resp)
				return
			}
		}

		prepareJSONResp(w, 200, resp)
	}
}

// Хендлер отвечает за удаление таски из БД
func DeleteTask(s *storage.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := Response{}

		taskID := r.URL.Query().Get("id")

		if taskID == "" {
			resp.Err = "Не указан идентификатор"
			prepareJSONResp(w, 400, resp)
			return
		}
		err := s.DeleteTaskByID(taskID)
		if errors.Is(err, sql.ErrNoRows) {
			resp.Err = "Задача не найдена"
			prepareJSONResp(w, 400, resp)
			return
		} else if err != nil {
			// Ошибку описали в методе, запишем напрямую тут
			resp.Err = fmt.Sprint(err)
			prepareJSONResp(w, 400, resp)
			return
		}

		prepareJSONResp(w, 200, resp)
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
		prepareJSONResp(w, 400, resp)
		return
	}

	// Совпадают ли пароли из тела и окружения
	if req.Password != envPass {
		resp.Err = "Неверный пароль"
		prepareJSONResp(w, 400, resp)
		return
	}

	// Формируем токен из секрета
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"passwordHash": fmt.Sprintf("%x", JWTSecret),
		"exp":          time.Now().Add(8 * time.Hour).Unix(),
	})

	// Подписываем токен
	tokenString, err := token.SignedString(JWTSecret)
	if err != nil {
		resp.Err = "Ошибка при создании токена"
		prepareJSONResp(w, 400, resp)
		return
	}
	resp.Token = tokenString

	prepareJSONResp(w, 200, resp)

}

// middleware для аутентификации
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Если пароль не установлен, то пропускаем проверки
		if envPass == "" {
			next(w, r)
			return
		}

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

// Функция подготовки JSON ответа
func prepareJSONResp(w http.ResponseWriter, code int, resp interface{}) {
	// Сериализируем ответ в JSON
	// Используем resp как пустой интерфейс, чтобы можно было прокинуть любую из структур ответа
	// Или даже строку
	JSONResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("ошибка сериализации JSON ответа: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(code)
	w.Write(JSONResp)
}
