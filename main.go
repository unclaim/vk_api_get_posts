package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

const (
	postURL        = "https://api.vk.com/method/wall.post"
	postAPIVersion = "5.199"
	vkAPIURL       = "https://api.vk.com/method/"
	apiVersion     = "5.199"
	accessToken    = "acf650ffacf650ffacf650ffbcafd9bacbaacf6acf650ffc4f19bc3a40adcc9a36dea49" // Замените на ваш токен
)

var (
	groups     = []string{"smeyaka"}                                                                                                                                                                                                            // Замените на ваши группы
	groupToken = "vk1.a.BLBOWDBNwA04A42cucM23VVw8GaEJsyCOUQhxyx5f5C_6C_NMuBEiKFp6RcRZdg3drYbbV-xjyWyRhuR0OKiZoOYqdWSnk2SYgYe4-3e_vXdFILixc0CE5eZOo-GHwTG1cl9Sdwc3lqPPuez5M_frkGc1R7iYakjG_0Yx27uGOOqVbMyl7uNslQpu2m07dZisJPTASpldGEVWfbCHHWYmQ" // Замените на ваш токен
	ownerID    = int64(-230198608)                                                                                                                                                                                                              // Замените на ID вашего сообщества (отрицательное значение для сообществ)
)

// Структуры для работы с API
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

// Функция для получения всех постов со стены группы с учетом пагинации.
func getAllWallPosts(domain string) ([]Post, error) {
	var allPosts []Post
	offset := 0

	for {
		params := url.Values{}
		params.Set("domain", domain)
		params.Set("count", "100")
		params.Set("offset", fmt.Sprintf("%d", offset))
		params.Set("access_token", accessToken)
		params.Set("v", apiVersion)

		resp, err := http.Get(vkAPIURL + "wall.get?" + params.Encode())
		if err != nil || resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to get wall posts: %w", err)
		}

		var response Response
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("failed to parse response JSON: %w", err)
		}

		allPosts = append(allPosts, response.Response.Items...)

		if len(response.Response.Items) < 100 { // Если меньше 100 постов, значит это последний запрос.
			break
		}

		offset += 100 // Увеличиваем смещение для следующего запроса.
	}

	return allPosts, nil
}

// Функция для публикации поста на стене ВКонтакте.
func postToWall(ownerID int64, message string, attachments []string) error {
	data := url.Values{}
	data.Set("access_token", groupToken) // Используем токен группы для публикации поста
	data.Set("v", postAPIVersion)
	data.Set("owner_id", fmt.Sprintf("%d", ownerID))
	data.Set("message", message)
	data.Set("from_group", "1")

	if len(attachments) > 0 {
		data.Set("attachments", url.QueryEscape(attachments[0])) // Присоединяем вложения (если есть)
		for _, attachment := range attachments[1:] {
			data.Add("attachments", url.QueryEscape(attachment))
		}
	}

	resp, err := http.PostForm(postURL, data)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Response struct{ PostID int64 }
		Error    *struct {
			ErrorCode     int
			ErrorMsg      string
			RequestParams []struct{ Key, Value string }
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response JSON: %w", err)
	}

	if result.Error != nil {
		fmt.Printf("Ошибка при публикации поста ID: %d - vk api error %d: %s\n",
			ownerID,
			result.Error.ErrorCode,
			result.Error.ErrorMsg,
		)
		return nil // Игнорируем ошибку и продолжаем публикацию следующих постов.
	}

	fmt.Printf("Пост успешно опубликован! ID поста: %d\n", result.Response.PostID)
	return nil
}

// Функция для получения постов из нескольких групп и их публикации.
func publishPostsFromGroups(groups []string) error {
	for _, group := range groups {
		posts, err := getAllWallPosts(group)
		if err != nil {
			return fmt.Errorf("ошибка при получении постов из группы %s: %w", group, err)
		}

		for _, post := range posts {
			message := post.Text

			var attachments []string

			for _, attachment := range post.Attachments {
				if attachment.Type == "photo" {
					attachments = append(attachments, fmt.Sprintf("%d_%d", attachment.Photo.OwnerID, attachment.Photo.ID))
				} else if attachment.Type == "video" {
					attachments = append(attachments, fmt.Sprintf("%d_%d", attachment.Video.OwnerID, attachment.Video.ID))
				}
			}

			// Выводим информацию о посте и его вложениях в терминал.
			fmt.Printf("Пост ID: %d\nТекст: %s\nВложения: %+v\n\n", post.ID, message, attachments)

			if err := postToWall(ownerID, message, attachments); err != nil {
				fmt.Printf("Ошибка при публикации поста ID %d: %s\n", post.ID, err)
			}
		}
	}

	return nil
}

// Обработчик HTTP-запросов.
func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Готов к публикации постов. Перейдите по адресу /publish чтобы опубликовать.")
}

// Обработчик для публикации постов.
func publishHandler(w http.ResponseWriter, r *http.Request) {

	if err := publishPostsFromGroups(groups); err != nil {
		http.Error(w, "Ошибка при публикации постов: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, "Все посты успешно опубликованы!")
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/publish", publishHandler)

	fmt.Println("Starting server on :8080...")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Failed to start server:", err)
		os.Exit(1)
	}
}
