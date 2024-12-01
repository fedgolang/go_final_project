package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
)

type Scheduler struct {
	db *sql.DB
}

type Task struct {
	ID      string `json:"id"`
	Date    string `json:"date,omitempty"`
	Title   string `json:"title"`
	Comment string `json:"comment,omitempty"`
	Repeat  string `json:"repeat,omitempty"`
}

// Не надумал более логичного решения проблемы, что нам иногда нужны все поля
// Вне зависимости, пустые они или нет
// Поэтому костыльный дубль структуры выше без omitempty
type TaskNoEmpty struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

// Функция открытия коннекта и создания БД, если её нет
func NewScheduler(DBPath string) (*Scheduler, *sql.DB) {
	// Проверим, что файлик с БД существует
	_, err := os.Stat(DBPath)

	var install bool
	if err != nil {
		install = true
	}

	db, err := sql.Open("sqlite", DBPath)
	if err != nil {
		log.Fatal(err) // Если к БД коннекта нет, падаем
	}

	if install {
		query_create := `CREATE TABLE scheduler(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date VARCHAR(255), 
			title VARCHAR(255),
			comment VARCHAR(255),
			repeat VARCHAR(126)
		);`
		query_index := `CREATE INDEX IF NOT EXISTS date_scheduler ON scheduler (date)`

		_, err = db.Exec(query_create)
		if err != nil {
			log.Fatal(err) // Если БД не смогли создать, падаем
		}

		_, err = db.Exec(query_index)
		if err != nil {
			log.Printf("Проблема с созданием индексов: %s", err) // Если не смогли создать индексы, просто проинформируем
		}
	}

	return &Scheduler{db: db}, db

}

// Функция инсерта в БД таски
func (s *Scheduler) PostTask(task Task) (int, error) {
	stmt, err := s.db.Prepare("INSERT INTO scheduler(date, title, comment, repeat) values(?,?,?,?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(task.Date, task.Title, task.Comment, task.Repeat)
	if err != nil {
		return 0, fmt.Errorf("ошибка при попытке добавить таску в БД: %s", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("ошибка при попытке добавить таску в БД: %s", err)
	}

	return int(id), nil
}

// Функция для запроса у БД лимитированное кол-во тасок, ближайшее к текущей дате
func (s Scheduler) GetTasks(limit int, today string) ([]TaskNoEmpty, error) {
	stmt, err := s.db.Prepare("SELECT id, date, title, comment, repeat " +
		"FROM scheduler WHERE date >= ? " +
		"ORDER BY date ASC " +
		"LIMIT ?")
	if err != nil {
		return nil, fmt.Errorf("ошибка при подготовке запроса: %s", err)
	}
	defer stmt.Close()

	// Запустим скрипт с нашими аргументами
	rows, err := stmt.Query(today, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %s", err)
	}
	defer rows.Close()

	// Пройдемся по всем полученным строкам и запишем их в слайс слайсов
	tasks := []TaskNoEmpty{}
	for rows.Next() {
		var task TaskNoEmpty
		if err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat); err != nil {
			return nil, fmt.Errorf("ошибка при чтении строки: %s", err)
		}
		tasks = append(tasks, task)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("ошибка при возврате строк: %s", err)
	}
	return tasks, nil
}

// Отдельная функция для поиска по дате
func (s Scheduler) GetTasksByDate(date string) ([]TaskNoEmpty, error) {
	stmt, err := s.db.Prepare("SELECT id, date, title, comment, repeat " +
		"FROM scheduler WHERE date = ?")
	if err != nil {
		return nil, fmt.Errorf("ошибка при подготовке запроса: %s", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(date)
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %s", err)
	}
	defer rows.Close()

	// Пройдемся по всем полученным строкам и запишем их в слайс слайсов
	tasks := []TaskNoEmpty{}
	for rows.Next() {
		var task TaskNoEmpty
		if err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat); err != nil {
			return nil, fmt.Errorf("ошибка при чтении строки: %s", err)
		}
		tasks = append(tasks, task)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("ошибка при возврате строк: %s", err)
	}

	return tasks, nil
}

// Отдельная функция для поиска по тексту, заголовок или коммент
func (s Scheduler) GetTasksBySearch(limit int, today, search string) ([]TaskNoEmpty, error) {
	stmt, err := s.db.Prepare("SELECT id, date, title, comment, repeat " +
		"FROM scheduler WHERE date >= ? AND (title LIKE ? OR comment LIKE ?) " +
		"ORDER BY date ASC " +
		"LIMIT ?")
	if err != nil {
		return nil, fmt.Errorf("ошибка при подготовке запроса: %s", err)
	}
	defer stmt.Close()

	// Так же добавим % для использования LIKE в скрипте
	searchPattern := "%" + search + "%"
	rows, err := stmt.Query(today, searchPattern, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %s", err)
	}
	defer rows.Close()

	// Пройдемся по всем полученным строкам и запишем их в слайс слайсов
	tasks := []TaskNoEmpty{}
	for rows.Next() {
		var task TaskNoEmpty
		if err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat); err != nil {
			return nil, fmt.Errorf("ошибка при чтении строки: %s", err)
		}
		tasks = append(tasks, task)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("ошибка при возврате строк: %s", err)
	}

	return tasks, nil
}

// Функция поиска в БД таски по ID
func (s Scheduler) GetTaskByID(id string) (Task, error) {
	task := Task{}
	// Подготовим запрос к БД
	stmt, err := s.db.Prepare("SELECT id, date, title, comment, repeat " +
		"FROM scheduler WHERE id =?")
	if err != nil {
		return task, fmt.Errorf("ошибка при попытке найти задание в БД: %s", err)
	}
	defer stmt.Close()

	// Так как id ключ с автоинкрементом, задача всегда будет одна
	// Поэтому пользуемся QueryRow
	query := stmt.QueryRow(id)
	err = query.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if errors.Is(err, sql.ErrNoRows) {
		return task, fmt.Errorf("задача не найдена")
	}

	return task, nil
}

func (s *Scheduler) EditTask(task Task) error {
	// Подготовим запрос к БД
	stmt, err := s.db.Prepare("UPDATE scheduler SET " +
		"date =?, " +
		"title =?, " +
		"comment =?, " +
		"repeat =? " +
		"WHERE id =? ")
	if err != nil {
		return fmt.Errorf("ошибка при попытке изменить задачу: %s", err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(task.Date, task.Title, task.Comment, task.Repeat, task.ID)
	if err != nil {
		return fmt.Errorf("ошибка при попытке изменить задачу: %s", err)
	}

	// Проверяем количество затронутых строк
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка при попытке изменить задачу: %s", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (s *Scheduler) DeleteTaskByID(id string) error {
	// Подготовим запрос к БД
	stmt, err := s.db.Prepare("DELETE FROM scheduler WHERE id=?")
	if err != nil {
		return fmt.Errorf("ошибка при попытке удалить задачу: %s", err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(id)
	if err != nil {
		return fmt.Errorf("ошибка при попытке удалить задачу: %s", err)
	}

	// Проверяем количество затронутых строк
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка при попытке удалить задачу: %s", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil

}
