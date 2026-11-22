package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

// NewLogger menulis log ke stdout dan (opsional) ke file.
func NewLogger(path string) (*log.Logger, func(), error) {
	if path == "" {
		return log.New(os.Stdout, "", log.LstdFlags), func() {}, nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
	if err != nil {
		return nil, nil, fmt.Errorf("buka file log: %w", err)
	}
	l := log.New(io.MultiWriter(os.Stdout, f), "", log.LstdFlags)
	return l, func() { f.Close() }, nil
}

// Batas panjang pesan Telegram (karakter).
const telegramMaxLen = 4096

// NotifyTelegram mengirim pesan lewat Telegram Bot API (sendMessage).
// Jika token atau chatID kosong, tidak melakukan apa-apa.
func NotifyTelegram(token, chatID, text string) error {
	if token == "" || chatID == "" {
		return nil
	}
	if r := []rune(text); len(r) > telegramMaxLen {
		text = string(r[:telegramMaxLen-3]) + "..."
	}

	form := url.Values{}
	form.Set("chat_id", chatID)
	form.Set("text", text)
	form.Set("disable_web_page_preview", "true")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm("https://api.telegram.org/bot"+token+"/sendMessage", form)
	if err != nil {
		// url.Error memuat URL lengkap (berisi token), jadi buang URL-nya.
		var uerr *url.Error
		if errors.As(err, &uerr) {
			err = uerr.Err
		}
		return fmt.Errorf("kirim telegram: %v", err)
	}
	defer resp.Body.Close()

	var res struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err := json.Unmarshal(b, &res); err != nil || !res.OK {
		desc := res.Description
		if desc == "" {
			desc = string(b)
		}
		return fmt.Errorf("telegram status %d: %s", resp.StatusCode, desc)
	}
	return nil
}
