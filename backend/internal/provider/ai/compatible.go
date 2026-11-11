// MIVA-PROVIDER: external API-key and speech adapters.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
	"english-app/internal/platform/jsonutil"
)

func (a *OpenAI) ConfigureCompatible(baseURL, format, jsonMode string) error {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return apperror.NewError(500, "AI_BASE_URL belum valid.")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "[::1]" || u.Hostname() == "::1")) {
		return apperror.NewError(500, "Gunakan HTTPS untuk alamat API eksternal.")
	}
	if format != "chat_completions" && format != "responses" {
		return apperror.NewError(500, "AI_API_FORMAT harus chat_completions atau responses.")
	}
	if !model.Contains([]string{"prompt", "json_object", "json_schema"}, jsonMode) {
		return apperror.NewError(500, "AI_JSON_MODE belum valid.")
	}
	if a.HTTP != nil {
		client := *a.HTTP
		client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
		a.HTTP = &client
	}
	a.BaseURL = u.String()
	a.Format = format
	a.JSONMode = jsonMode
	return nil
}

func (a *OpenAI) compatibleResponse(ctx context.Context, instructions string, input any, schema map[string]any, name string, max int) (string, error) {
	if schema != nil {
		instructions += "\nReturn ONLY one valid JSON object, without Markdown fences or commentary. It must match this JSON schema exactly: " + string(jsonutil.Marshal(schema))
	}
	messages := []map[string]string{{"role": "system", "content": instructions}}
	switch v := input.(type) {
	case string:
		messages = append(messages, map[string]string{"role": "user", "content": v})
	case []map[string]string:
		messages = append(messages, v...)
	default:
		return "", apperror.NewError(500, "Format konteks AI belum valid.")
	}
	body := map[string]any{"model": a.Model, "messages": messages, "max_tokens": max, "stream": false}
	if schema != nil {
		switch a.JSONMode {
		case "json_object":
			body["response_format"] = map[string]any{"type": "json_object"}
		case "json_schema":
			body["response_format"] = map[string]any{
				"type":        "json_schema",
				"json_schema": map[string]any{"name": name, "strict": true, "schema": schema},
			}
		}
	}
	raw, err := a.request(ctx, "/chat/completions", "application/json", bytes.NewReader(jsonutil.Marshal(body)))
	if err != nil {
		return "", err
	}
	var response struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content json.RawMessage `json:"content"`
				Refusal string          `json:"refusal"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(raw, &response) != nil || len(response.Choices) == 0 {
		return "", apperror.NewError(502, "Penyedia AI belum memberikan jawaban yang valid.")
	}
	choice := response.Choices[0]
	if choice.FinishReason != "stop" {
		return "", apperror.NewError(502, "Jawaban AI terhenti atau belum lengkap. Coba lagi; materi belum disimpan.")
	}
	if choice.Message.Refusal != "" {
		return "", apperror.NewError(502, "AI belum dapat memenuhi permintaan ini. Coba topik belajar lain.")
	}
	var text string
	if json.Unmarshal(choice.Message.Content, &text) != nil {
		var parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if json.Unmarshal(choice.Message.Content, &parts) != nil {
			return "", apperror.NewError(502, "Format jawaban penyedia AI belum didukung.")
		}
		for _, p := range parts {
			if p.Type == "text" {
				text += p.Text
			}
		}
	}
	text = strings.TrimSpace(text)
	if schema != nil && strings.HasPrefix(text, "```") {
		line := strings.IndexByte(text, '\n')
		if line >= 0 && strings.HasSuffix(text, "```") {
			text = strings.TrimSpace(strings.TrimSuffix(text[line+1:], "```"))
		}
	}
	if text == "" {
		return "", apperror.NewError(502, "AI belum menghasilkan jawaban. Coba lagi.")
	}
	if schema != nil && !json.Valid([]byte(text)) {
		return "", apperror.NewError(502, "Materi AI belum berbentuk JSON yang valid. Coba lagi atau pilih model lain.")
	}
	return text, nil
}

func (a *OpenAI) Probe(ctx context.Context) (string, error) {
	return a.response(
		ctx,
		"You are a connection checker. Reply exactly OK, without explanation.",
		"Connection test",
		nil,
		"",
		16,
	)
}
