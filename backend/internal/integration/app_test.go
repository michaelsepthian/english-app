package integration

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"english-app/internal/model"
	"english-app/internal/platform/identity"
	"english-app/internal/platform/jsonutil"
	"english-app/internal/provider/ai"
	"english-app/internal/service"
)

func fixtureLesson() model.Lesson {
	l := model.Lesson{
		Title:      "A day at the café",
		Theme:      "Daily routine",
		Vocabulary: model.StarterWords[:3],
		Tense: model.Tense{
			Name:        model.Tenses[0],
			Explanation: "Simple present digunakan untuk kebiasaan, rutinitas, dan fakta umum dalam kehidupan sehari-hari.",
			Formula:     "Subject + V1 (+ s/es for he/she/it)",
			Examples: []model.Example{
				{EN: "I go to the café every day.", ID: "Saya pergi ke kafe setiap hari."},
				{EN: "She eats breakfast at seven.", ID: "Dia sarapan pukul tujuh."},
			},
		},
		Reading:   model.Reading{Title: "A small café", Passage: strings.Repeat("Every morning, Maya goes to a small café near her home. She eats breakfast and reads a book. Her friend works at the café and makes her a cup of tea. ", 2)},
		Listening: model.Listening{Title: "A morning conversation", Transcript: strings.Repeat("Hello, I would like a cup of tea and a sandwich, please. I usually visit this café before work. ", 2)},
	}
	for i := 0; i < 8; i++ {
		l.Questions = append(l.Questions, model.Question{ID: fmt.Sprintf("q%d", i), Skill: []string{"vocabulary", "tenses", "reading", "listening"}[i/2], Prompt: "Choose the correct answer for this practice question.", Options: []string{"Right answer", "Wrong answer one", "Wrong answer two", "Wrong answer three"}, Answer: 0, Explanation: "Ini jawaban yang sesuai dengan materi pada pelajaran.", SourceID: ""})
	}
	return l
}
func TestGradeAndPublicLesson(t *testing.T) {
	l := fixtureLesson()
	if err := l.Validate(); err != nil {
		t.Fatal(err)
	}
	answers := map[string]int{}
	for _, q := range l.Questions {
		answers[q.ID] = 0
	}
	answers["q2"] = 2
	a, err := model.Grade(l, answers, 92)
	if err != nil || a.Score != 7 || a.Total != 8 {
		t.Fatalf("grade: %+v %v", a, err)
	}
	if _, err = model.Grade(l, map[string]int{"q0": 0}, 0); err == nil {
		t.Fatal("accepted incomplete answers")
	}
	r := model.LessonRecord{ID: identity.NewID(), Day: "2026-11-10", Content: l, Draft: answers}
	raw := string(jsonutil.Marshal(r.Public()))
	if strings.Contains(raw, "\"answer\"") || strings.Contains(raw, "\"explanation\":\"Ini jawaban") || strings.Contains(raw, l.Listening.Transcript) {
		t.Fatal("answer key or listening transcript leaked before submission")
	}
	l.Questions[1].ID = l.Questions[0].ID
	if l.Validate() == nil {
		t.Fatal("duplicate question IDs accepted")
	}
}
func TestCalendarAndWav(t *testing.T) {
	s := model.DefaultSettings("Maya")
	s.Goal = "Belajar, setiap hari; dengan senang\nGoal baru"
	ics := model.Calendar(s, "test", "http://localhost:5180", time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC))
	for _, part := range []string{
		"DTSTART:20261110T130000Z",
		"RRULE:FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
		"TRIGGER:PT0M",
		"Belajar\\, setiap hari\\; dengan senang\\nGoal baru",
	} {
		if !strings.Contains(ics, part) {
			t.Fatalf("missing %s", part)
		}
	}
	for _, line := range strings.Split(ics, "\r\n") {
		if len(line) > 75 {
			t.Fatal("unfolded long calendar line")
		}
	}
	wav := pcmToWav([]byte{0, 1, 2, 3})
	if string(wav[:4]) != "RIFF" || binary.LittleEndian.Uint32(wav[24:28]) != 24000 || len(wav) != 48 {
		t.Fatal("invalid WAV")
	}
	if model.Monday(time.Date(2026, 11, 8, 18, 0, 0, 0, time.UTC)) != "2026-11-09" {
		t.Fatal("wrong Jakarta week boundary")
	}
}

type fakeAI struct{ generated int }

func (f *fakeAI) Generate(context.Context, model.Settings, []string) (model.Lesson, error) {
	f.generated++
	return fixtureLesson(), nil
}
func (f *fakeAI) Lookup(context.Context, string) (model.Word, error) {
	w := model.StarterWords[0]
	w.Word = "explore"
	w.V1 = "explore"
	w.V2 = "explored"
	w.V3 = "explored"
	return w, nil
}
func (f *fakeAI) Chat(context.Context, model.Settings, []model.Message, string) (string, error) {
	return "Hello! How was your day? Kamu bisa menjawab dengan satu kalimat.", nil
}
func TestOpenAIResponsesContract(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if r.URL.Path != "/responses" || body["store"] != false || body["model"] != "test-model" {
			t.Error("bad Responses request")
		}
		format := body["text"].(map[string]any)["format"].(map[string]any)
		if format["strict"] != true || format["type"] != "json_schema" {
			t.Error("schema not strict")
		}
		writeJSON(
			w,
			200,
			map[string]any{
				"status": "completed",
				"output": []any{map[string]any{"content": []any{map[string]any{"type": "output_text", "text": string(jsonutil.Marshal(fixtureLesson()))}}}},
			},
		)
	}))
	defer upstream.Close()
	ai := ai.NewAI("test-key", "test-model")
	if err := ai.ConfigureCompatible(upstream.URL, "responses", "prompt"); err != nil {
		t.Fatal(err)
	}
	l, err := ai.Generate(t.Context(), model.DefaultSettings("Maya"), nil)
	if err != nil || len(l.Questions) != 8 {
		t.Fatalf("contract: %v", err)
	}
}
func TestPostgresUserIsolationAndLearning(t *testing.T) {
	if os.Getenv("RUN_DB_TESTS") != "1" {
		t.Skip("set RUN_DB_TESTS=1 with dedicated english_app database role")
	}
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD")),
		Host:   os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT"),
		Path:   os.Getenv("DB_NAME"),
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()
	store, err := NewStore(t.Context(), u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Pool.Close()
	fake := &fakeAI{}
	server := newTestServer(store, fake, service.Config{
		AIConfigured: true,
		Registration: true,
		DailyLimit:   120,
	})
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	request := func(method, path string, body any, cookie *http.Cookie, origin string) (int, map[string]any, []*http.Cookie) {
		var raw io.Reader
		if body != nil {
			raw = bytes.NewReader(jsonutil.Marshal(body))
		}
		req, _ := http.NewRequest(method, host.URL+"/api/"+path, raw)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		r, err := host.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()
		var result map[string]any
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			_ = json.NewDecoder(r.Body).Decode(&result)
		}
		return r.StatusCode, result, r.Cookies()
	}
	origin := "http://localhost:5180"
	suffix := identity.NewID()
	var a, b *http.Cookie
	var aid, bid string
	for i := 0; i < 2; i++ {
		status, res, cookies := request(
			"POST",
			"auth/register",
			map[string]string{
				"email":    fmt.Sprintf("integration-%d-%s@example.test", i, suffix),
				"password": "Integration-password-123!",
				"name":     "Test Learner",
			},
			nil,
			origin,
		)
		if status != 200 {
			t.Fatalf("register: %d %+v", status, res)
		}
		id := res["user"].(map[string]any)["id"].(string)
		if i == 0 {
			a = cookies[0]
			aid = id
		} else {
			b = cookies[0]
			bid = id
		}
		if !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
			t.Fatal("session cookie not protected")
		}
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = store.Pool.Exec(ctx, "DELETE FROM english_app.users WHERE id=ANY($1)", []string{aid, bid})
	}()
	if status, _, _ := request("GET", "dashboard", nil, nil, ""); status != 401 {
		t.Fatal("unauthenticated access allowed")
	}
	if status, _, _ := request("POST", "lesson/generate", map[string]any{}, a, "http://evil.test"); status != 403 {
		t.Fatal("cross-site generation allowed")
	}
	// Retired model-connection routes must not remain available to signed-in users.
	for _, path := range []string{"chatgpt/authorize", "chatgpt/callback", "chatgpt/status", "chatgpt/models"} {
		code, _, _ := request("GET", path, nil, a, "")
		if code != 404 {
			t.Fatalf("retired route %s still available: %d", path, code)
		}
	}
	for _, path := range []string{"chatgpt/connect", "chatgpt/select", "chatgpt/model", "chatgpt/disconnect", "transcribe"} {
		code, _, _ := request("POST", path, map[string]any{}, a, origin)
		if code != 404 {
			t.Fatalf("retired route %s still available: %d", path, code)
		}
	}
	disabled := newTestServer(store, nil, service.Config{})
	disabledHost := httptest.NewServer(disabled.Handler())
	defer disabledHost.Close()
	originalHost := host
	host = disabledHost
	code, audioResult, _ := request("POST", "audio", map[string]string{"kind": "word", "id": model.StarterWords[0].ID, "part": "word"}, a, origin)
	host = originalHost
	if code != 200 || audioResult["mode"] != "device" || audioResult["text"] != model.StarterWords[0].Word {
		t.Fatalf("browser speech unexpectedly requires a model: %d %+v", code, audioResult)
	}
	status, res, _ := request("POST", "lesson/generate", map[string]any{}, a, origin)
	if status != 200 {
		t.Fatalf("generate %d %+v", status, res)
	}
	l := res["lesson"].(map[string]any)
	id := l["id"].(string)
	qs := l["questions"].([]any)
	if _, exists := qs[0].(map[string]any)["answer"]; exists {
		t.Fatal("answer leaked")
	}
	if _, exists := l["listening"].(map[string]any)["transcript"]; exists {
		t.Fatal("transcript leaked")
	}
	request("POST", "lesson/generate", map[string]any{}, a, origin)
	if fake.generated != 1 {
		t.Fatal("duplicate paid generation")
	}
	for _, route := range []string{"lesson/draft", "lesson/submit"} {
		method := "POST"
		if route == "lesson/draft" {
			method = "PATCH"
		}
		status, _, _ = request(method, route, map[string]any{"id": id, "answers": map[string]int{"q0": 0}}, b, origin)
		if status != 404 {
			t.Fatalf("cross-user lesson write allowed: %d", status)
		}
	}
	status, _, _ = request("POST", "audio", map[string]string{"kind": "listening", "id": id}, b, origin)
	if status != 404 {
		t.Fatal("cross-user audio leaked")
	}
	status, _, _ = request("PATCH", "lesson/draft", map[string]any{"id": id, "answers": map[string]int{"q0": 2}}, a, origin)
	if status != 200 {
		t.Fatal("draft failed")
	}
	_, res, _ = request("GET", "lesson", nil, a, "")
	if res["lesson"].(map[string]any)["draft"].(map[string]any)["q0"] != float64(2) {
		t.Fatal("draft not persisted")
	}
	status, _, _ = request("POST", "lesson/submit", map[string]any{"id": id, "answers": map[string]int{"q0": 0}}, a, origin)
	if status != 400 {
		t.Fatal("partial submission accepted")
	}
	_, _ = store.Pool.Exec(
		t.Context(),
		"UPDATE english_app.lessons SET last_tick=now()-interval '15 seconds' WHERE id=$1",
		id,
	)
	all := map[string]int{}
	for _, q := range fixtureLesson().Questions {
		all[q.ID] = 0
	}
	status, res, _ = request("POST", "lesson/submit", map[string]any{"id": id, "answers": all}, a, origin)
	if status != 200 {
		t.Fatalf("submit %d %+v", status, res)
	}
	attempt := res["lesson"].(map[string]any)["attempt"].(map[string]any)
	if attempt["score"] != float64(8) || attempt["seconds"].(float64) < 14 || attempt["listeningTranscript"] == "" {
		t.Fatal("wrong score, time or transcript")
	}
	_, res, _ = request("POST", "lesson/submit", map[string]any{"id": id, "answers": all}, a, origin)
	if res["stats"].(map[string]any)["lessons"] != float64(1) {
		t.Fatal("duplicate submission counted twice")
	}
	status, res, _ = request("POST", "chat", map[string]string{"text": "Halo, saya ingin belajar."}, a, origin)
	if status != 200 || len(res["messages"].([]any)) != 2 {
		t.Fatal("chat not persisted")
	}
	_, res, _ = request("GET", "chat", nil, b, "")
	if len(res["messages"].([]any)) != 0 {
		t.Fatal("chat leaked across users")
	}
	status, res, _ = request("POST", "words/lookup", map[string]string{"query": "explore"}, a, origin)
	if status != 200 {
		t.Fatalf("dictionary %d %+v", status, res)
	}
	p := model.StudyPlan{Vocabulary: 5, Tenses: []string{model.Tenses[0]}, Reading: 2, Listening: 2, Questions: 30}
	reqID := identity.NewID()
	status, res, _ = request("POST", "lesson/prepare", map[string]any{"plan": p, "requestId": reqID}, a, origin)
	if status != 202 {
		t.Fatalf("prepare %d %+v", status, res)
	}
	jobID := res["job"].(map[string]any)["id"].(string)
	var prepared map[string]any
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		_, res, _ = request("GET", "lesson/job?id="+jobID, nil, a, "")
		j := res["job"].(map[string]any)
		if j["status"] == "failed" {
			t.Fatalf("job failed %+v", j)
		}
		if j["status"] == "done" {
			prepared = res["lesson"].(map[string]any)
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if prepared == nil {
		t.Fatal("job not completed")
	}
	preparedID := prepared["id"].(string)
	status, res, _ = request("POST", "lesson/prepare", map[string]any{"plan": p, "requestId": reqID}, a, origin)
	if status != 200 || res["job"].(map[string]any)["id"] != jobID {
		t.Fatal("prepare not idempotent")
	}
	_, res, _ = request("GET", "lesson/job?id="+jobID, nil, b, "")
	if res["job"] != nil {
		t.Fatal("job leaked across users")
	}
	_, res, _ = request("GET", "lesson?id="+preparedID, nil, b, "")
	if res["lesson"] != nil {
		t.Fatal("lesson history leaked")
	}
	status, res, _ = request(
		"POST",
		"audio",
		map[string]string{"kind": "listening", "id": preparedID, "sourceId": "listening-2"},
		a,
		origin,
	)
	if status != 200 || res["text"] != plannedFixture(p).Listenings[1].Transcript {
		t.Fatalf("wrong listening source %d %+v", status, res)
	}
	status, res, _ = request(
		"POST",
		"reading/check",
		map[string]any{
			"lessonId":      preparedID,
			"sourceId":      "reading-1",
			"sentenceIndex": 0,
			"transcript":    "Every morning Maya goes to a small café near her home",
		},
		a,
		origin,
	)
	if status != 200 || res["check"].(map[string]any)["mode"] != "words" {
		t.Fatalf("reading check %d %+v", status, res)
	}
	status, _, _ = request(
		"POST",
		"reading/check",
		map[string]any{
			"lessonId":      preparedID,
			"sourceId":      "reading-1",
			"sentenceIndex": 0,
			"transcript":    "Every morning Maya",
		},
		b,
		origin,
	)
	if status != 404 {
		t.Fatal("reading write isolation failed")
	}
	all = map[string]int{}
	for _, q := range plannedFixture(p).Questions {
		all[q.ID] = q.Answer
	}
	all["q1"] = 1
	status, _, _ = request("PATCH", "lesson/draft", map[string]any{"id": preparedID, "answers": all}, a, origin)
	if status != 200 {
		t.Fatal("30 answer draft failed")
	}
	status, res, _ = request("POST", "lesson/submit", map[string]any{"id": preparedID, "answers": all}, a, origin)
	if status != 200 {
		t.Fatalf("30 submit %d %+v", status, res)
	}
	got := res["lesson"].(map[string]any)["attempt"].(map[string]any)
	if got["score"] != float64(29) || got["percent"] != float64(97) || got["review"] == nil {
		t.Fatalf("score or review missing %+v", got)
	}
	_, res, _ = request("GET", "lesson/history", nil, a, "")
	if len(res["lessons"].([]any)) != 2 {
		t.Fatal("multiple sessions lost")
	}
	_, res, _ = request("GET", "lesson/history", nil, b, "")
	if len(res["lessons"].([]any)) != 0 {
		t.Fatal("history leaked")
	}
	if status, _, _ = request("POST", "auth/logout", map[string]any{}, a, origin); status != 200 {
		t.Fatal("logout failed")
	}
	if status, _, _ = request("GET", "dashboard", nil, a, ""); status != 401 {
		t.Fatal("revoked session accepted")
	}
}

// pcmToWav builds test audio fixtures only; production speech is browser-based.
func pcmToWav(pcm []byte) []byte {
	b := make([]byte, 44+len(pcm))
	copy(b, "RIFF")
	binary.LittleEndian.PutUint32(b[4:], uint32(36+len(pcm)))
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1)
	binary.LittleEndian.PutUint16(b[22:], 1)
	binary.LittleEndian.PutUint32(b[24:], 24000)
	binary.LittleEndian.PutUint32(b[28:], 48000)
	binary.LittleEndian.PutUint16(b[32:], 2)
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], uint32(len(pcm)))
	copy(b[44:], pcm)
	return b
}
