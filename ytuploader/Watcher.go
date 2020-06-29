package ytuploader

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

type folder struct {
	Path    string `yaml:"path"`
	Mask    string `yaml:"mask"`
	maskRgx *regexp.Regexp
}

// Watcher - Watcher
type Watcher struct {
	store        Store            // подключение в БД и репозиторий запросов
	watchingChan chan TrackedFile // канал для чтения файлов из папки
	newfileChan  chan TrackedFile // канал для выбрасывания новых файлов в папке
	Timeout      time.Duration    `yaml:"timeout"` // задержка после чтения папки
	Folders      []folder         `yaml:"folder"`  // папки для мониторинга
}

// TrackedFile - TrackedFile
type TrackedFile struct {
	os.FileInfo
	dir string
}

// Fullpath - полный путьк  файлу
func (tf *TrackedFile) Fullpath() string {
	return filepath.Join(tf.dir, tf.Name())
}

// Close - закончить мониторинг
func (w *Watcher) Close() {
	close(w.watchingChan)
	close(w.newfileChan)
}

func (w *Watcher) readConfig(configFile string) {
	r, err := os.Open(configFile)
	if err != nil {
		// TODO: сделать создание дефолтного конфига
		log.Fatalln(err)
	}
	defer r.Close()
	if err := yaml.NewDecoder(r).Decode(&w); err != nil {
		log.Fatalln(err)
	}
	for i, folder := range w.Folders {
		folder.Path = filepath.FromSlash(folder.Path)
		if folder.Mask == "" {
			folder.Mask = `.*`
		}
		rgx, err := regexp.Compile(folder.Mask)
		if err != nil {
			log.Fatalln(err)
		}
		w.Folders[i].maskRgx = rgx
	}
}

// NewWatcher - мониторинг папок с файлами
func NewWatcher(configFile string) Watcher {
	var configPath, _ = filepath.Abs(configFile)
	var w = Watcher{
		store:        NewStore(),
		watchingChan: make(chan TrackedFile),
		newfileChan:  make(chan TrackedFile),
	}
	w.readConfig(configPath)
	return w
}

// Watch - Watch
func (w *Watcher) Watch() chan TrackedFile {
	for _, folder := range w.Folders {
		w.checkOnFirst(folder)
		go w.wather(folder)
	}
	go func() {
		for {
			select {
			case f := <-w.watchingChan:
				if w.store.AddFile(f.dir, f.Name(), f.ModTime().Unix()) {
					w.newfileChan <- f
				}
			}
		}
	}()
	return w.newfileChan
}

func (w *Watcher) wather(dir folder) {
	for {
		files, err := ioutil.ReadDir(dir.Path)
		if err != nil {
			log.Println("ERROR:", err)
			return
		}
		for _, file := range files {
			if !file.IsDir() && dir.maskRgx.MatchString(file.Name()) {
				w.watchingChan <- TrackedFile{dir: dir.Path, FileInfo: file}
			}
		}
		time.Sleep(w.Timeout)
	}
}

func (w *Watcher) checkOnFirst(dir folder) {
	var countOnStart = w.store.FolderCount(dir.Path)
	if countOnStart != 0 {
		return
	}
	fmt.Println("Инициализация новой папки, добавление файлов без загрузки:")
	files, err := ioutil.ReadDir(dir.Path)
	if err != nil {
		log.Println("ERROR:", err)
		return
	}
	for _, file := range files {
		if !file.IsDir() && dir.maskRgx.MatchString(file.Name()) {
			fmt.Println(filepath.Join(dir.Path, file.Name()))
			w.store.AddFile(dir.Path, file.Name(), file.ModTime().Unix())
		}
	}
}
