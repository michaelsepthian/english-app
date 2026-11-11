// MIVA-DOMAIN: learning data and pure validation rules.
package model

import (
	"errors"
	"strings"
	"time"
)

type Tense struct {
	Name        string    `json:"name"`
	Explanation string    `json:"explanation"`
	Formula     string    `json:"formula"`
	Examples    []Example `json:"examples"`
}

type Reading struct {
	Title   string `json:"title"`
	Passage string `json:"passage"`
}

type Listening struct {
	Title      string `json:"title"`
	Transcript string `json:"transcript"`
}

type Question struct {
	ID          string   `json:"id"`
	Skill       string   `json:"skill"`
	Prompt      string   `json:"prompt"`
	Options     []string `json:"options"`
	Answer      int      `json:"answer"`
	Explanation string   `json:"explanation"`
	SourceID    string   `json:"sourceId,omitempty"`
}

type Lesson struct {
	Title      string          `json:"title"`
	Theme      string          `json:"theme"`
	Vocabulary []Word          `json:"vocabulary"`
	Tense      Tense           `json:"tense"`
	Reading    Reading         `json:"reading"`
	Listening  Listening       `json:"listening"`
	Questions  []Question      `json:"questions"`
	Plan       *StudyPlan      `json:"plan,omitempty"`
	Tenses     []TenseItem     `json:"tenses,omitempty"`
	Readings   []ReadingItem   `json:"readings,omitempty"`
	Listenings []ListeningItem `json:"listenings,omitempty"`
}

type Result struct {
	ID          string `json:"id"`
	Chosen      int    `json:"chosen"`
	Answer      int    `json:"answer"`
	Correct     bool   `json:"correct"`
	Explanation string `json:"explanation"`
	Skill       string `json:"skill"`
}

type Attempt struct {
	Score                int               `json:"score"`
	Total                int               `json:"total"`
	Seconds              int               `json:"seconds"`
	Results              []Result          `json:"results"`
	ListeningTranscript  string            `json:"listeningTranscript"`
	ListeningTranscripts map[string]string `json:"listeningTranscripts,omitempty"`
	Percent              int               `json:"percent"`
	Review               *LessonReview     `json:"review,omitempty"`
}

type LessonRecord struct {
	ID       string
	Day      string
	Content  Lesson
	Draft    map[string]int
	Seconds  int
	LastTick *time.Time
	Attempt  *Attempt
}

func (r LessonRecord) Public() map[string]any {
	qs := make([]map[string]any, 0, len(r.Content.Questions))
	for _, q := range r.Content.Questions {
		qs = append(qs, map[string]any{"id": q.ID, "skill": q.Skill, "prompt": q.Prompt, "options": q.Options, "sourceId": q.SourceID})
	}
	listenings := []map[string]string{}
	for _, item := range r.Content.ListeningItems() {
		listenings = append(listenings, map[string]string{"id": item.ID, "title": item.Title})
	}
	return map[string]any{
		"id":         r.ID,
		"day":        r.Day,
		"title":      r.Content.Title,
		"theme":      r.Content.Theme,
		"vocabulary": r.Content.Vocabulary,
		"tense":      r.Content.Tense,
		"reading":    r.Content.Reading,
		"listening":  map[string]string{"title": r.Content.Listening.Title},
		"tenses":     r.Content.TenseItems(),
		"readings":   r.Content.ReadingItems(),
		"listenings": listenings,
		"plan":       r.Content.Plan,
		"questions":  qs,
		"draft":      r.Draft,
		"attempt":    r.Attempt,
	}
}

func (l Lesson) Validate() error {
	if l.Plan != nil {
		return l.validatePlan()
	}
	if len(l.Title) < 3 || len(l.Title) > 160 || len(l.Theme) > 160 || len(l.Vocabulary) < 3 || len(l.Vocabulary) > 5 || !Contains(Tenses, l.Tense.Name) || len(l.Tense.Explanation) < 30 || len(l.Tense.Examples) < 2 || len(l.Reading.Passage) < 150 || len(l.Reading.Passage) > 5000 || len(l.Listening.Transcript) < 80 || len(l.Listening.Transcript) > 2500 || len(l.Questions) < 8 || len(l.Questions) > 12 {
		return errors.New("Pelajaran AI belum lengkap. Coba lagi.")
	}
	for _, w := range l.Vocabulary {
		if err := w.Validate(); err != nil {
			return err
		}
	}
	seen := map[string]bool{}
	skills := map[string]int{}
	for _, q := range l.Questions {
		if len(q.ID) == 0 || len(q.ID) > 30 || seen[q.ID] || !Contains([]string{"vocabulary", "tenses", "reading", "listening"}, q.Skill) || len(q.Options) != 4 || q.Answer < 0 || q.Answer > 3 || len(q.Prompt) < 5 || len(q.Explanation) < 5 {
			return errors.New("Soal AI belum valid. Coba lagi.")
		}
		opts := map[string]bool{}
		for _, o := range q.Options {
			v := strings.ToLower(strings.TrimSpace(o))
			if v == "" || opts[v] {
				return errors.New("Pilihan jawaban AI belum valid.")
			}
			opts[v] = true
		}
		seen[q.ID] = true
		skills[q.Skill]++
	}
	for _, s := range []string{"vocabulary", "tenses", "reading", "listening"} {
		if skills[s] < 1 {
			return errors.New("Ada bagian latihan AI yang belum tersedia.")
		}
	}
	return nil
}
