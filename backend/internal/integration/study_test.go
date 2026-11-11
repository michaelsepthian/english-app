package integration

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"english-app/internal/model"
	"english-app/internal/platform/jsonutil"
	"english-app/internal/provider/ai"
	"english-app/internal/provider/speech"
	"english-app/internal/service"
)

func plannedFixture(p model.StudyPlan) model.Lesson {
	old := fixtureLesson()
	l := model.Lesson{
		Title:      "English for your next conversation",
		Theme:      "Daily conversation",
		Plan:       &p,
		Vocabulary: []model.Word{},
		Tenses:     []model.TenseItem{},
		Readings:   []model.ReadingItem{},
		Listenings: []model.ListeningItem{},
		Questions:  []model.Question{},
	}
	for i := 0; i < p.Vocabulary; i++ {
		w := model.StarterWords[i%len(model.StarterWords)]
		if i >= len(model.StarterWords) {
			w.Word = fmt.Sprintf("practice%d", i)
		}
		w.Examples = []model.Example{
			{EN: w.Example, ID: w.Translation},
			{EN: "I practice this word at work.", ID: "Saya berlatih kata ini di tempat kerja."},
			{EN: "We learn something new every day.", ID: "Kami belajar sesuatu yang baru setiap hari."},
		}
		l.Vocabulary = append(l.Vocabulary, w)
	}
	sources := []model.Question{}
	if p.Vocabulary > 0 {
		sources = append(sources, model.Question{Skill: "vocabulary", SourceID: "vocabulary"})
	}
	for i, name := range p.Tenses {
		t := old.Tense
		t.Name = name
		t.Explanation = strings.Repeat(old.Tense.Explanation+" ", 4)
		t.Examples = append(append(append([]model.Example{}, t.Examples...), t.Examples...), t.Examples...)
		id := fmt.Sprintf("tense-%d", i+1)
		l.Tenses = append(l.Tenses, model.TenseItem{
			ID:    id,
			Tense: t,
			Uses:  []string{"Kebiasaan sehari-hari", "Fakta umum"},
			Rules: []string{
				"Gunakan do atau does dalam pertanyaan.",
				"Gunakan bentuk dasar setelah does.",
				"Tambahkan s untuk he, she, dan it.",
			},
			CommonMistakes: []model.Example{
				{EN: "She go to work.", ID: "She goes to work. Tambahkan s pada verb."},
				{EN: "Does she goes?", ID: "Does she go? Gunakan bentuk dasar setelah does."},
			},
		})
		sources = append(sources, model.Question{Skill: "tenses", SourceID: id})
	}
	for i := 0; i < p.Reading; i++ {
		id := fmt.Sprintf("reading-%d", i+1)
		l.Readings = append(l.Readings, model.ReadingItem{ID: id, Reading: old.Reading})
		sources = append(sources, model.Question{Skill: "reading", SourceID: id})
	}
	for i := 0; i < p.Listening; i++ {
		id := fmt.Sprintf("listening-%d", i+1)
		l.Listenings = append(l.Listenings, model.ListeningItem{ID: id, Listening: old.Listening})
		sources = append(sources, model.Question{Skill: "listening", SourceID: id})
	}
	for i := 0; i < p.Questions; i++ {
		q := old.Questions[i%len(old.Questions)]
		q.ID = fmt.Sprintf("q%d", i+1)
		q.Skill = sources[i%len(sources)].Skill
		q.SourceID = sources[i%len(sources)].SourceID
		l.Questions = append(l.Questions, q)
	}
	return l
}
func (f *fakeAI) GeneratePlan(ctx context.Context, s model.Settings, p model.StudyPlan, progress func(string, int) error) (model.Lesson, error) {
	if err := progress("Menyiapkan sesi pilihanmu", 20); err != nil {
		return model.Lesson{}, err
	}
	return plannedFixture(p), nil
}
func (f *fakeAI) Review(context.Context, model.Lesson, model.Attempt, string) (model.LessonReview, error) {
	return model.LessonReview{
		Summary:      "Kamu sudah memahami materi sesi ini. Pertahankan kebiasaan belajarmu.",
		Strengths:    []string{"Sudah mencoba semua bagian latihan."},
		Improvements: []string{"Ulangi kata yang masih sulit."},
		NextSteps:    []string{"Gunakan satu kalimat baru dalam percakapan.", "Dengarkan lagi contoh yang sudah dipelajari."},
	}, nil
}
func TestPlannedLessonBoundsCoverageAndScoring(t *testing.T) {
	p := model.StudyPlan{Vocabulary: 30, Tenses: model.Tenses[:3], Reading: 5, Listening: 5, Questions: 30}
	l := plannedFixture(p)
	if err := l.Validate(); err != nil {
		t.Fatal(err)
	}
	answers := map[string]int{}
	for i, q := range l.Questions {
		answers[q.ID] = 0
		if i%3 == 0 {
			answers[q.ID] = 1
		}
	}
	a, err := model.Grade(l, answers, 42)
	if err != nil || a.Score != 20 || a.Percent != 67 || len(a.Results) != 30 || len(a.ListeningTranscripts) != 5 {
		t.Fatalf("bad grade %+v %v", a, err)
	}
	public := string(jsonutil.Marshal((model.LessonRecord{Content: l}).Public()))
	if strings.Contains(public, "\"answer\"") || strings.Contains(public, l.Listenings[4].Transcript) {
		t.Fatal("answer or transcript leaked")
	}
	l.Questions[0].SourceID = "absent"
	if l.Validate() == nil {
		t.Fatal("unrelated question accepted")
	}
	for _, bad := range []model.StudyPlan{
		{Questions: 1},
		{Listening: 6, Questions: 30},
		{Reading: 1, Questions: 31},
		{Reading: 2, Questions: 1},
		{Vocabulary: 31, Questions: 30},
		{Tenses: []string{model.Tenses[0], model.Tenses[0]}, Questions: 2},
	} {
		if bad.Validate() == nil {
			t.Fatalf("invalid plan accepted %+v", bad)
		}
	}
}
func TestWordMatchingAndAudioValidation(t *testing.T) {
	exact := service.CompareWords("I go to work every day.", "i go to work every day")
	if exact.Score != 100 || exact.Mode != "words" {
		t.Fatalf("bad exact %+v", exact)
	}
	partial := service.CompareWords("I go to work every day.", "I walk to work today")
	if partial.Score >= 100 || partial.Score <= 0 {
		t.Fatalf("bad partial %+v", partial)
	}
	wav := make([]byte, 32044)
	copy(wav, "RIFF")
	binary.LittleEndian.PutUint32(wav[4:8], uint32(len(wav)-8))
	copy(wav[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(wav[16:20], 16)
	binary.LittleEndian.PutUint16(wav[20:22], 1)
	binary.LittleEndian.PutUint16(wav[22:24], 1)
	binary.LittleEndian.PutUint32(wav[24:28], 16000)
	binary.LittleEndian.PutUint32(wav[28:32], 32000)
	binary.LittleEndian.PutUint16(wav[32:34], 2)
	binary.LittleEndian.PutUint16(wav[34:36], 16)
	copy(wav[36:], "data")
	binary.LittleEndian.PutUint32(wav[40:44], 32000)
	if err := speech.ValidateWav(wav); err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint32(wav[24:28], 24000)
	if speech.ValidateWav(wav) == nil {
		t.Fatal("bad sample rate accepted")
	}
}

func TestStagedPlanIncludesGoalAndExactCounts(t *testing.T) {
	p := model.StudyPlan{Vocabulary: 5, Tenses: []string{model.Tenses[0]}, Reading: 1, Listening: 2, Questions: 30}
	fixture := plannedFixture(p)
	step := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct{ Role, Content string }
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || len(body.Messages) < 2 {
			t.Error("bad request")
			http.Error(w, "bad", 400)
			return
		}
		if !strings.Contains(body.Messages[1].Content, "practice workplace introductions") {
			t.Error("learner goal missing from staged request")
		}
		var out any
		switch step {
		case 0:
			out = map[string]any{"words": fixture.Vocabulary}
		case 1:
			out = fixture.Tenses[0]
		case 2:
			out = fixture.Readings[0]
		case 3, 4:
			out = fixture.Listenings[step-3]
		default:
			offset := (step - 5) * 10
			out = map[string]any{"questions": fixture.Questions[offset : offset+10]}
		}
		step++
		completion(w, string(jsonutil.Marshal(out)), "stop")
	}))
	defer upstream.Close()
	ai := ai.NewAI("test-key", "test-model")
	if err := ai.ConfigureCompatible(upstream.URL, "chat_completions", "prompt"); err != nil {
		t.Fatal(err)
	}
	settings := model.DefaultSettings("Test")
	settings.Goal = "practice workplace introductions"
	callbacks := 0
	lesson, err := ai.GeneratePlan(t.Context(), settings, p, func(stage string, progress int) error {
		callbacks++
		if stage == "" || progress < 0 || progress > 100 {
			t.Error("bad progress")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if step != 8 || callbacks != 8 || len(lesson.Questions) != 30 || len(lesson.Listenings) != 2 {
		t.Fatal("wrong stage counts")
	}
}

func TestAcousticAssessmentRequiresProviderScores(t *testing.T) {
	wav := pcmToWav(make([]byte, 32000))
	binary.LittleEndian.PutUint32(wav[24:28], 16000)
	binary.LittleEndian.PutUint32(wav[28:32], 32000)
	valid := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Ocp-Apim-Subscription-Key") != "speech-test" || r.Header.Get("Pronunciation-Assessment") == "" {
			t.Error("assessment headers missing")
		}
		if !valid {
			writeJSON(
				w,
				200,
				map[string]any{"RecognitionStatus": "Success", "NBest": []any{map[string]any{"Display": "Hello"}}},
			)
			return
		}
		writeJSON(
			w,
			200,
			map[string]any{
				"RecognitionStatus": "Success",
				"NBest": []any{map[string]any{
					"Display":                 "Hello",
					"PronunciationAssessment": map[string]any{"AccuracyScore": 85, "FluencyScore": 80, "CompletenessScore": 100, "PronScore": 84},
					"Words": []any{map[string]any{
						"Word":                    "Hello",
						"PronunciationAssessment": map[string]any{"AccuracyScore": 85, "ErrorType": "None"},
					}},
				}},
			},
		)
	}))
	defer upstream.Close()
	newSpeech := speech.NewSpeechAssessment
	speech := &speech.SpeechAssessment{Endpoint: upstream.URL, Key: "speech-test", HTTP: upstream.Client()}
	if _, err := speech.Assess(t.Context(), "Hello", wav); err == nil {
		t.Fatal("invented acoustic scores from transcript")
	}
	valid = true
	result, err := speech.Assess(t.Context(), "Hello", wav)
	if err != nil || result.Mode != "pronunciation" || result.Score != 84 || result.Words[0].Status != "correct" {
		t.Fatalf("assessment failed %+v %v", result, err)
	}
	if _, err := newSpeech("https://evil.test", "secret"); err == nil {
		t.Fatal("untrusted speech endpoint accepted")
	}
}
