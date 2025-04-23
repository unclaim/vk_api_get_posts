package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	apiURL      = "https://api.vk.com/method/wall.get"
	apiVersion  = "5.199"                                                                   // Актуальная версия VK API
	accessToken = "acf650ffacf650ffacf650ffbcafd9bacbaacf6acf650ffc4f19bc3a40adcc9a36dea49" // Замените на ваш токен
	groupID     = "smeyaka"                                                                 // Замените на короткое имя вашего сообщества
	count       = 10                                                                        // Количество записей, которые необходимо получить
	offset      = 0                                                                         // Смещение для выборки записей
)

type Video struct {
	ID       int    `json:"id"`
	OwnerID  int    `json:"owner_id"`
	Title    string `json:"title"`
	Duration int    `json:"duration"`
}

type Attachment struct {
	Type  string `json:"type"`
	Video Video  `json:"video"`
}

type Response struct {
	Response struct {
		Count int `json:"count"`
		Items []struct {
			ID          int          `json:"id"`
			Text        string       `json:"text"` // Текст записи
			Date        int64        `json:"date"` // Дата создания записи
			OwnerID     int          `json:"owner_id"`
			Attachments []Attachment `json:"attachments,omitempty"` // Вложения записи
		} `json:"items"`
	} `json:"response"`
}

func main() {
	url := fmt.Sprintf("%s?domain=%s&count=%d&offset=%d&access_token=%s&v=%s", apiURL, groupID, count, offset, accessToken, apiVersion)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching data:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: received non-200 response code")
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}

	fmt.Printf("Total posts: %d\n", response.Response.Count)
	for _, item := range response.Response.Items {
		for _, attachment := range item.Attachments {
			if attachment.Type == "video" {
				videoURL := fmt.Sprintf("https://vk.com/video%d_%d", attachment.Video.OwnerID, attachment.Video.ID)
				fmt.Printf("Video URL: %s\n", videoURL)
			}
		}

	}
}
