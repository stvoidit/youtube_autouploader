package ytuploader

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cheggaaa/pb/v3"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/youtube/v3"
)

const (
	tokenFile  = `youtube-token.json`
	secretFile = `client_secret.json`
)

// ClientYT - типа структуры для работы с API
type ClientYT struct {
	config *oauth2.Config // конфиг oauth2
	token  *oauth2.Token  // токен oauth2
}

// NewClient - вернуть новый клиент для Yutube
func NewClient() *ClientYT {
	var config = configFromFile()
	var token = tokenFromFile(config)
	var client = &ClientYT{
		config: config,
		token:  token,
	}
	client.refresh()
	return client
}

// UploadVideo - загрузить видео
func (c *ClientYT) UploadVideo(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	var _, name = filepath.Split(filename)
	var upload = &youtube.Video{
		Snippet: &youtube.VideoSnippet{
			Title: name,
		},
		Status: &youtube.VideoStatus{PrivacyStatus: "private"},
	}
	stat, _ := file.Stat()

	log.Println("Start upload", filename)
	var call = c.getService().Videos.Insert("id,snippet,status", upload)
	var bar = &costumePB{ProgressBar: pb.New64(stat.Size())}
	bar.Start()
	response, err := call.Media(file, googleapi.ChunkSize(256)).ProgressUpdater(bar.progressBar).Do()
	if err != nil {
		if err.Error() == `quotaExceeded` {
			fmt.Println("Первышена квота запросов к API")
			return err
		}
		return err
	}
	bar.Finish()
	log.Println("Finish upload", filename)
	log.Printf("https://studio.youtube.com/video/%s/edit\n", response.Id)
	return nil
}

type costumePB struct {
	*pb.ProgressBar
}

func (cpb *costumePB) progressBar(current, total int64) {
	cpb.SetCurrent(current)
}

// getService - новый экземпляр сервисного API YouTube
func (c *ClientYT) getService() *youtube.Service {
	var client = c.config.Client(context.Background(), c.token)
	service, err := youtube.New(client)
	if err != nil {
		// "Error creating YouTube client"
		panic(err)
	}
	return service
}

// Проверка валидности токена. После выполнения автоматически обновляет
// токен в файле и в текущем экземпляре объекта ClientYT
func (c *ClientYT) refresh() {
	if c.token.Valid() {
		return
	}
	defer log.Println("token refresh")
	refToken, err := c.config.TokenSource(oauth2.NoContext, c.token).Token()
	if err != nil {
		panic(err)
	}
	c.token = refToken
	saveToken(c.token)

}

func saveToken(token *oauth2.Token) {
	f, err := os.OpenFile(tokenFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(&token); err != nil {
		panic(err)
	}
	log.Println("token save")
}

func createToken(config *oauth2.Config) (token *oauth2.Token) {
	var authURL = config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	if err := exec.Command("rundll32", "url.dll,FileProtocolHandler", authURL).Start(); err != nil {
		fmt.Println("Не удалось открыть браузер")
	}
	var code string
	fmt.Printf("Go to the following link in your browser. After completing "+
		"the authorization flow, enter the authorization code on the command "+
		"line: \n%v\n\nCode: ", authURL)
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}
	token, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token %v", err)
	}
	return token
}

// чтение токена из файла
func tokenFromFile(config *oauth2.Config) *oauth2.Token {
	var token = new(oauth2.Token)
	f, err := os.Open(tokenFile)
	if err != nil {
		token = createToken(config)
		saveToken(token)
		return token
	}

	if err = json.NewDecoder(f).Decode(token); err != nil {
		panic(err)
	}
	defer f.Close()
	return token
}

// чтение секретки из файла
func configFromFile() *oauth2.Config {
	b, _ := ioutil.ReadFile(secretFile)
	config, err := google.ConfigFromJSON(b, youtube.YoutubeUploadScope)
	if err != nil {
		// log.Fatalf("Unable to read client secret file: %v", err)
		panic(err)
	}
	return config
}
