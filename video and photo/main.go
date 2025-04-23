package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"text/template"
)

const (
	vkAPIURL    = "https://api.vk.com/method/"
	apiVersion  = "5.199"                                                                   // актуальная версия VK API
	accessToken = "acf650ffacf650ffacf650ffbcafd9bacbaacf6acf650ffc4f19bc3a40adcc9a36dea49" // замените на ваш токен
)

var groups = []string{"smeyaka"} // Замените на ваши группы

type Size struct {
	Type   string `json:"type"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Photo struct {
	ID      int    `json:"id"`
	OwnerID int    `json:"owner_id"`
	Sizes   []Size `json:"sizes"` // Массив с копиями изображения в разных размерах
}

type Video struct {
	ID          int    `json:"id"`
	OwnerID     int    `json:"owner_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Player      string `json:"player"` // URL для воспроизведения видео
}

type Attachment struct {
	Type  string `json:"type"`
	Photo *Photo `json:"photo,omitempty"`
	Video *Video `json:"video,omitempty"` // Добавляем поле для видео
}

type Likes struct {
	Count int `json:"count"`
}

type Reposts struct {
	Count int `json:"count"`
}

type Post struct {
	ID          int          `json:"id"`
	OwnerID     int          `json:"owner_id"`
	Date        int          `json:"date"`
	Text        string       `json:"text"`
	Likes       Likes        `json:"likes,omitempty"`
	Reposts     Reposts      `json:"reposts,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"` // Добавляем поле для вложений
}

type Response struct {
	Response struct {
		Count int    `json:"count"`
		Items []Post `json:"items"`
	} `json:"response"`
}

func getWallPosts(domain string, count int, offset int, results chan<- *Response) {
	params := url.Values{}
	params.Set("domain", domain)
	params.Set("count", fmt.Sprintf("%d", count))
	params.Set("offset", fmt.Sprintf("%d", offset))
	params.Set("access_token", accessToken)
	params.Set("v", apiVersion)
	params.Set("extended", "1")
	params.Set("photo_sizes", "1") // Запрашиваем все размеры фотографий

	resp, err := http.Get(vkAPIURL + "wall.get?" + params.Encode())
	if err != nil {
		results <- nil // Отправляем nil в случае ошибки
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		results <- nil // Отправляем nil в случае ошибки
		return
	}

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		results <- nil // Отправляем nil в случае ошибки
		return
	}

	results <- &response // Отправляем результат в канал
}

// Функция для получения самого большого размера фотографии
func getLargestPhotoURL(sizes []Size) string {
	var largest Size
	for _, size := range sizes {
		if size.Type == "w" || size.Type == "z" || size.Type == "y" || size.Type == "x" || size.Type == "o" {
			if size.Width > largest.Width {
				largest = size
			}
		}
	}
	return largest.URL
}

// Функция для получения URL видео по его ID и OwnerID
func getVideoURL(video Video) string {
	return fmt.Sprintf("https://vk.com/video%d_%d", video.OwnerID, video.ID)
}

func getPostsFromMultipleGroups(groups []string) []*Response {
	var wg sync.WaitGroup
	results := make(chan *Response, len(groups)) // Канал для результатов

	for _, group := range groups {
		wg.Add(1)
		go func(g string) {
			defer wg.Done()
			getWallPosts(g, 100, 0, results) // Получаем первые 100 записей со стены
		}(group)
	}

	wg.Wait() // Ждем завершения всех горутин
	close(results)

	var allResponses []*Response
	for res := range results {
		if res != nil {
			allResponses = append(allResponses, res) // Собираем результаты
		}
	}

	return allResponses
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	responses := getPostsFromMultipleGroups(groups)

	tmpl := `
<!DOCTYPE html>
<html lang="en">
<head>
   <meta charset="UTF-8">
   <meta name="viewport" content="width=device-width, initial-scale=1.0">
   <title>VK Wall Posts</title>
</head>
<body>
   <h1>Wall Posts from VK</h1>
   <div>
       {{range .}}
           {{range .Response.Items}}
               <div>
                   <p><strong>Post ID: {{.ID}}</strong></p>
                   <p>Date: {{.Date}}</p>
                   <p>{{.Text}}</p>
                   <p>Likes: {{.Likes.Count}} | Reposts: {{.Reposts.Count}}</p>

                   {{if .Attachments}}
                       <div>
                           <h3>Attachments:</h3>
                           {{range .Attachments}}
                               {{if eq .Type "photo"}}
                                   {{if .Photo.Sizes}}
                                       <img src="{{getLargestPhotoURL .Photo.Sizes}}" alt="Photo" width="300"/>
                                   {{end}}
                               {{else if eq .Type "video"}}
                                   <h4>{{.Video.Title}}</h4>
                                   <p>{{.Video.Description}}</p>
                                   <a href="{{getVideoURL .Video}}">Watch Video</a>
                               {{end}}
                           {{end}}
                       </div>
                   {{end}}
               </div>
               <hr/>
           {{end}}
       {{end}}
   </div>
</body>
</html>`

	t := template.Must(template.New("index").Funcs(template.FuncMap{
		"getLargestPhotoURL": getLargestPhotoURL,
		"getVideoURL":        getVideoURL,
	}).Parse(tmpl))
	if err := t.Execute(w, responses); err != nil {
		http.Error(w, "Failed to render template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	http.HandleFunc("/", indexHandler)
	fmt.Println("Starting server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Failed to start server:", err)
		os.Exit(1)
	}
}
