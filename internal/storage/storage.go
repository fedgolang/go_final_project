package storage

import (
	"database/sql"
	"log"
	"os"
)

type Scheduler struct {
	db *sql.DB
}

type Task struct {
	Date    string `json:"date,omitempty"`
	Title   string `json:"title"`
	Comment string `json:"comment,omitempty"`
	Repeat  string `json:"repeat,omitempty"`
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
