package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// cmdChannel sends files / messages to the bot runtime's chat channel.
//
//	sme-cli channel send-file <path> --chat-id <id> [--caption TEXT]
//	sme-cli channel send-message <text> --chat-id <id>
//
// Uses Telegram Bot API directly (token read from ~/.openclaw/openclaw.json).
// Other channels (Discord, Zalo) are not yet wired.
func cmdChannel(args []string) {
	if len(args) == 0 {
		errOut("usage: channel send-file|send-message")
		return
	}
	switch args[0] {
	case "send-file":
		channelSendFile(args[1:])
	case "send-message":
		channelSendMessage(args[1:])
	default:
		errOut("unknown channel command: " + args[0])
	}
}

type channelFlags struct {
	Path    string
	Message string
	ChatID  string
	Caption string
}

func parseChannelFlags(args []string, positionalKey string) channelFlags {
	var f channelFlags
	var positional []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--chat-id":
			if i+1 < len(args) {
				f.ChatID = args[i+1]
				i++
			}
		case "--caption":
			if i+1 < len(args) {
				f.Caption = args[i+1]
				i++
			}
		default:
			positional = append(positional, args[i])
		}
	}
	if len(positional) > 0 {
		switch positionalKey {
		case "path":
			f.Path = positional[0]
		case "message":
			f.Message = strings.Join(positional, " ")
		}
	}
	return f
}

func channelSendFile(args []string) {
	f := parseChannelFlags(args, "path")
	if f.Path == "" || f.ChatID == "" {
		errOut("usage: channel send-file <path> --chat-id <id> [--caption TEXT]")
	}
	info, err := os.Stat(f.Path)
	if err != nil {
		errOut("file not found: " + f.Path)
	}
	if info.Size() == 0 {
		errOut("file is empty: " + f.Path)
	}

	token, err := readTelegramBotToken()
	if err != nil {
		errOut(err.Error())
	}
	fh, err := os.Open(f.Path)
	if err != nil {
		errOut("open file: " + err.Error())
	}
	defer fh.Close()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("chat_id", f.ChatID)
	if f.Caption != "" {
		_ = mw.WriteField("caption", f.Caption)
	}
	part, err := mw.CreateFormFile("document", filepath.Base(f.Path))
	if err != nil {
		errOut("form file: " + err.Error())
	}
	if _, err := io.Copy(part, fh); err != nil {
		errOut("copy file: " + err.Error())
	}
	mw.Close()

	url := "https://api.telegram.org/bot" + token + "/sendDocument"
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		errOut("build request: " + err.Error())
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		errOut("telegram: " + err.Error())
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		errOut(fmt.Sprintf("telegram HTTP %d: %s", resp.StatusCode, string(raw)))
	}
	var parsed struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int64 `json:"message_id"`
			Document  struct {
				FileName string `json:"file_name"`
				FileSize int64  `json:"file_size"`
			} `json:"document"`
		} `json:"result"`
	}
	_ = json.Unmarshal(raw, &parsed)
	if !parsed.OK {
		errOut("telegram rejected: " + string(raw))
	}
	okOut(map[string]interface{}{
		"message_id": parsed.Result.MessageID,
		"file_name":  parsed.Result.Document.FileName,
		"file_size":  parsed.Result.Document.FileSize,
		"chat_id":    f.ChatID,
	})
}

func channelSendMessage(args []string) {
	f := parseChannelFlags(args, "message")
	if f.Message == "" || f.ChatID == "" {
		errOut("usage: channel send-message <text> --chat-id <id>")
	}
	token, err := readTelegramBotToken()
	if err != nil {
		errOut(err.Error())
	}
	body, err := json.Marshal(map[string]string{"chat_id": f.ChatID, "text": f.Message})
	if err != nil {
		errOut("encode body: " + err.Error())
	}
	url := "https://api.telegram.org/bot" + token + "/sendMessage"
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		errOut("build request: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		errOut("telegram: " + err.Error())
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		errOut(fmt.Sprintf("telegram HTTP %d: %s", resp.StatusCode, string(raw)))
	}
	var parsed struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
	}
	_ = json.Unmarshal(raw, &parsed)
	if !parsed.OK {
		errOut("telegram rejected: " + string(raw))
	}
	okOut(map[string]interface{}{
		"message_id": parsed.Result.MessageID,
		"chat_id":    f.ChatID,
	})
}

// readTelegramBotToken reads the Telegram bot token from the runtime's
// openclaw.json. Tries $OPENCLAW_HOME then the conventional ~/.openclaw.
func readTelegramBotToken() (string, error) {
	paths := []string{}
	if h := os.Getenv("OPENCLAW_HOME"); h != "" {
		paths = append(paths, filepath.Join(h, "openclaw.json"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".openclaw", "openclaw.json"))
	}
	var lastErr error
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			lastErr = err
			continue
		}
		var cfg struct {
			Channels struct {
				Telegram struct {
					BotToken string `json:"botToken"`
				} `json:"telegram"`
			} `json:"channels"`
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			lastErr = err
			continue
		}
		if cfg.Channels.Telegram.BotToken != "" {
			return cfg.Channels.Telegram.BotToken, nil
		}
	}
	if lastErr != nil {
		return "", fmt.Errorf("telegram bot token not found in openclaw.json: %w", lastErr)
	}
	return "", fmt.Errorf("telegram bot token not found in openclaw.json (checked: %v)", paths)
}
