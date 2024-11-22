package storage

import (
	"database/sql"
	"log"
	"os"
	"time"
)

type Scheduler struct {
	db *sql.DB
}

type Task struct {
	id      int
	date    time.Time
	title   string
	comment string
	repeat  string
}

// Функция открытия коннекта к БД
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
			date DATE, 
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

func (s Scheduler) GetTask(id int) (Task, error) {
	stmt, err := s.db.Prepare("SELECT id, date, title, comment, repeat FROM scheduler WHERE id =?")
	if err != nil {
		return Task{}, err
	}
	defer stmt.Close() // По завершению закроем коннект

	var task Task

	err = stmt.QueryRow(id).Scan(&task)
	if err != nil {
		return Task{}, err
	}

	return task, nil

}
