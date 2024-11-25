package storage

import (
	"database/sql"
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
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

// Функция для запроса у БД лимитированное кол-во тасок, ближайшее к текущей дате
func (s Scheduler) GetTasks(limit int, today string) (*sql.Rows, error) {
	// Производим выборку данных из БД, сортированных по возрастанию даты
	// Так же дата должна быть больше сегодняшней, так как ищем ближайшие задачи
	// И ограничиваем лимитом, он нам приходит как аргумент limit
	stmt, err := s.db.Prepare("SELECT id, date, title, comment, repeat " +
		"FROM scheduler WHERE date >= ? " +
		"ORDER BY date ASC " +
		"LIMIT ? ")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	res, err := stmt.Query(today, limit)
	// Обработаем ошибку отсутствия строк, иначе рухнем с err
	if err == sql.ErrNoRows {
		return res, nil
	} else if err != nil {
		return nil, err
	}

	return res, nil

}

// Функция поиска в БД таски по ID
func (s Scheduler) GetTaskByID(id string) *sql.Row {
	// Подготовим запрос к БД
	stmt, err := s.db.Prepare("SELECT id, date, title, comment, repeat " +
		"FROM scheduler WHERE id =?")
	if err != nil {
		return nil
	}
	defer stmt.Close()

	// Так как id ключ с автоинкрементом, задача всегда будет одна
	// Поэтому пользуемся QueryRow
	res := stmt.QueryRow(id)

	return res
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
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(task.Date, task.Title, task.Comment, task.Repeat, task.ID)
	if err != nil {
		return err
	}

	// Проверяем количество затронутых строк
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("нет записи")
	}

	return nil
}
