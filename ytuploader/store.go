package ytuploader

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3" //импорт драйвера SQLite3
)

// Store - БД
type Store struct {
	db   *sql.DB
	lock *sync.Mutex
}

// NewStore - подключение к БД
func NewStore() Store {
	var pwd, _ = os.Getwd()
	var dataFolder = filepath.Join(pwd, "data")
	if err := os.MkdirAll(dataFolder, os.ModePerm); err != nil {
		panic(err)
	}
	conn, err := sql.Open("sqlite3", filepath.Join(dataFolder, "monitor.sqlite3.db"))
	if err != nil {
		panic(err)
	}
	if err := conn.Ping(); err != nil {
		panic(err)
	}
	var store = Store{
		db:   conn,
		lock: new(sync.Mutex),
	}
	store.init()
	return store
}

func (s *Store) init() {
	s.lock.Lock()
	defer s.lock.Unlock()
	const createTabke = `CREATE TABLE IF NOT EXISTS monitor (
		folder TEXT(400) NOT NULL,
		filename TEXT(300) NOT NULL,
		"modify" INTEGER NOT NULL,
		sent NUMERIC DEFAULT 0 NOT NULL
	);
	CREATE INDEX IF NOT EXISTS monitor_folder_IDX ON monitor (folder DESC,filename DESC,"modify" DESC);`
	if _, err := s.db.Exec(createTabke); err != nil {
		panic(err)
	}
}

// FolderCount - количество файлов по папке
func (s *Store) FolderCount(folder string) (count uint64) {
	const selectCount = `SELECT COUNT(*) FROM monitor WHERE folder = $1`
	if err := s.db.QueryRow(selectCount, folder).Scan(&count); err != nil {
		return 0
	}
	return count
}

// AddFile - добавлние файла
func (s *Store) AddFile(dir, name string, timestamp int64) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	const insert = `INSERT INTO monitor
		(folder, filename, "modify")
		SELECT $1, $2, $3
		WHERE NOT EXISTS (
			SELECT 1 FROM monitor m 
			WHERE m.folder=$1 AND m.filename =$2 AND m."modify" = $3);`
	result, err := s.db.Exec(insert, dir, name, timestamp)
	if err != nil {
		panic(err)
	}
	if isnew, err := result.RowsAffected(); err == nil && isnew == 1 {
		return true
	}
	return false
}
