package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gorilla/websocket"
)

const (
	botToken = "8262089909:AAERsogW0R37xzVznBaskE3EWLZzGGRkv8o" // <-- o'zingizning tokenni yozing
	wsURL    = "wss://back-ai.ccenter.uz/ws/1cee6529-91c6-4b5f-b170-9b930134021f"
)

type WSContent struct {
	Citations     []string       `json:"citations"`
	ImagesURL     []string       `json:"images_url"`
	Location      []Location     `json:"location"`
	Organizations []Organization `json:"organizations"`
	Text          string         `json:"text"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Organization struct {
	Name     string   `json:"name"`
	Address  string   `json:"address"`
	Location Location `json:"location"`
}

type WSMessage struct {
	Content *WSContent `json:"content,omitempty"`
	Text    string     `json:"text,omitempty"`
	Status  string     `json:"status,omitempty"`
}

func askWebSocket(question string) (string, error) {
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return "", fmt.Errorf("WebSocketga ulanishda xatolik: %v", err)
	}
	defer conn.Close()

	req := map[string]string{"message": question}
	reqData, _ := json.Marshal(req)
	if err := conn.WriteMessage(websocket.TextMessage, reqData); err != nil {
		return "", fmt.Errorf("yozishda xatolik: %v", err)
	}

	var fullMsg WSMessage
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return "", fmt.Errorf("javobni o‘qishda xatolik: %v", err)
		}

		if !bytes.HasPrefix(msg, []byte("{")) {
			continue
		}

		var temp WSMessage
		if err := json.Unmarshal(msg, &temp); err != nil {
			continue
		}

		if temp.Content != nil {
			fullMsg = temp
		}

		if temp.Status == "end" {
			break
		}
	}

	return formatResponse(fullMsg.Content), nil
}

func formatResponse(content *WSContent) string {
	if content == nil {
		return "❌ Javob topilmadi."
	}

	var b strings.Builder

	if content.Text != "" {
		b.WriteString(content.Text)
	}

	if len(content.Organizations) > 0 {
		b.WriteString("🏛 *Tashkilotlar:*\n")
		for _, org := range content.Organizations {
			b.WriteString(fmt.Sprintf("- %s — %s\n", org.Name, org.Address))
		}
		b.WriteString("\n")
	}

	if len(content.Location) > 0 {
		for _, loc := range content.Location {
			mapURL := fmt.Sprintf("https://www.google.com/maps?q=%f,%f", loc.Latitude, loc.Longitude)
			b.WriteString(fmt.Sprintf("📍 *Manzil:* [Xaritada ko‘rish](%s)\n", mapURL))
		}
		b.WriteString("\n")
	}

	if len(content.Citations) > 0 {
		b.WriteString("🔗 *Manbalar:*\n")
		for _, c := range content.Citations {
			b.WriteString(fmt.Sprintf("- [%s](%s)\n", c, c))
		}
	}

	return b.String()
}

func main() {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Botni yaratishda xatolik: %v", err)
	}
	log.Printf("✅ Bot ishga tushdi: %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil || update.Message.Text == "" {
			continue
		}

		userMsg := update.Message.Text
		log.Printf("[%s]: %s", update.Message.From.UserName, userMsg)

		answer, err := askWebSocket(userMsg)
		if err != nil {
			answer = "⚠️ Xatolik: " + err.Error()
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, answer)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
	}
}
