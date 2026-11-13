// MIVA-DOMAIN: learning data and pure validation rules.
package model

import (
	"errors"
	"fmt"
	"strings"
)

type StudyPlan struct {
	Vocabulary int      `json:"vocabulary"`
	Tenses     []string `json:"tenses"`
	Reading    int      `json:"reading"`
	Listening  int      `json:"listening"`
	Questions  int      `json:"questions"`
}

func (p StudyPlan) MinimumQuestions() int {
	n := p.Reading + p.Listening + len(p.Tenses)
	if p.Vocabulary > 0 {
		n++
	}
	return n
}

func (p StudyPlan) Validate() error {
	if p.Vocabulary < 0 || p.Vocabulary > 30 || p.Reading < 0 || p.Reading > 5 || p.Listening < 0 || p.Listening > 5 || len(p.Tenses) > 3 || p.MinimumQuestions() == 0 || p.Questions < p.MinimumQuestions() || p.Questions > 30 {
		return errors.New("Pilih materi belajar dan jumlah soal yang mencakup semua bagian. Maksimal 30 kosakata, 5 bacaan, 5 listening, 3 tenses, dan 30 soal.")
	}
	seen := map[string]bool{}
	for _, t := range p.Tenses {
		if !Contains(Tenses, t) || seen[t] {
			return errors.New("Pilihan tenses belum valid.")
		}
		seen[t] = true
	}
	return nil
}

type TenseItem struct {
	ID string `json:"id"`
	Tense
	Uses           []string  `json:"uses"`
	Rules          []string  `json:"rules"`
	CommonMistakes []Example `json:"commonMistakes"`
}

type ReadingItem struct {
	ID string `json:"id"`
	Reading
}

type ListeningItem struct {
	ID string `json:"id"`
	Listening
}

type LessonReview struct {
	Summary      string   `json:"summary"`
	Strengths    []string `json:"strengths"`
	Improvements []string `json:"improvements"`
	NextSteps    []string `json:"nextSteps"`
}

func (l Lesson) ReadingItems() []ReadingItem {
	if l.Readings != nil {
		return l.Readings
	}
	if l.Reading.Passage != "" {
		return []ReadingItem{{"reading-1", l.Reading}}
	}
	return []ReadingItem{}
}

func (l Lesson) ListeningItems() []ListeningItem {
	if l.Listenings != nil {
		return l.Listenings
	}
	if l.Listening.Transcript != "" {
		return []ListeningItem{{"listening-1", l.Listening}}
	}
	return []ListeningItem{}
}

func (l Lesson) TenseItems() []TenseItem {
	if l.Tenses != nil {
		return l.Tenses
	}
	if l.Tense.Name != "" {
		return []TenseItem{{ID: "tense-1", Tense: l.Tense}}
	}
	return []TenseItem{}
}

func (l Lesson) validatePlan() error { return l.validatePreparation(true) }

// ValidatePreparation validates accumulated parts before committing a checkpoint.
func (l Lesson) ValidatePreparation() error { return l.validatePreparation(false) }

func (l Lesson) validatePreparation(complete bool) error {
	p := l.Plan
	if p == nil {
		return errors.New("Rencana belajar belum tersedia.")
	}
	if err := p.Validate(); err != nil {
		return err
	}
	if len(l.Vocabulary) > p.Vocabulary || len(l.Tenses) > len(p.Tenses) || len(l.Readings) > p.Reading || len(l.Listenings) > p.Listening || len(l.Questions) > p.Questions {
		return errors.New("Jumlah materi melebihi rencana.")
	}
	if complete && (len(l.Title) < 3 || len(l.Vocabulary) != p.Vocabulary || len(l.Tenses) != len(p.Tenses) || len(l.Readings) != p.Reading || len(l.Listenings) != p.Listening || len(l.Questions) != p.Questions) {
		return errors.New("Jumlah materi belum sesuai rencana. Silakan coba lagi.")
	}
	sources := map[string]string{}
	if p.Vocabulary > 0 {
		sources["vocabulary"] = "vocabulary"
	}
	wordKeys := map[string]bool{}
	for _, w := range l.Vocabulary {
		key := strings.ToLower(strings.TrimSpace(w.Word))
		if wordKeys[key] {
			return errors.New("Kosakata belum cukup beragam.")
		}
		wordKeys[key] = true
		if err := w.Validate(); err != nil {
			return err
		}
		for _, example := range w.Examples {
			if strings.TrimSpace(example.EN) == "" || strings.TrimSpace(example.ID) == "" {
				return errors.New("Contoh kosakata belum lengkap.")
			}
		}
		if len(w.Examples) < 3 {
			return errors.New("Contoh kosakata belum lengkap.")
		}
	}
	for i, t := range l.Tenses {
		if t.ID != fmt.Sprintf("tense-%d", i+1) || t.Name != p.Tenses[i] || len(t.Explanation) < 200 || len(t.Examples) < 6 || len(t.Uses) < 2 || len(t.Rules) < 3 || len(t.CommonMistakes) < 2 {
			return errors.New("Penjelasan tenses belum lengkap.")
		}
		sources[t.ID] = "tenses"
	}
	for i, r := range l.Readings {
		if r.ID != fmt.Sprintf("reading-%d", i+1) || len(r.Passage) < 100 || len(r.Passage) > 5000 {
			return errors.New("Materi reading belum lengkap.")
		}
		sources[r.ID] = "reading"
	}
	for i, r := range l.Listenings {
		if r.ID != fmt.Sprintf("listening-%d", i+1) || len(r.Transcript) < 80 || len(r.Transcript) > 2500 {
			return errors.New("Materi listening belum lengkap.")
		}
		sources[r.ID] = "listening"
	}
	ids := map[string]bool{}
	covered := map[string]bool{}
	for _, q := range l.Questions {
		if q.ID == "" || ids[q.ID] || sources[q.SourceID] == "" || sources[q.SourceID] != q.Skill || len(q.Options) != 4 || q.Answer < 0 || q.Answer > 3 || len(q.Prompt) < 5 || len(q.Explanation) < 20 {
			return errors.New("Latihan belum mencakup materi dengan benar.")
		}
		opts := map[string]bool{}
		for _, o := range q.Options {
			k := strings.ToLower(strings.TrimSpace(o))
			if k == "" || opts[k] {
				return errors.New("Pilihan jawaban belum valid.")
			}
			opts[k] = true
		}
		ids[q.ID] = true
		covered[q.SourceID] = true
	}
	if !complete {
		return nil
	}
	for source := range sources {
		if !covered[source] {
			return errors.New("Ada materi yang belum masuk dalam latihan.")
		}
	}
	return nil
}

var Tenses = []string{
	"Simple Present",
	"Present Continuous",
	"Simple Past",
	"Present Perfect",
	"Simple Future",
	"Past Continuous",
	"Past Perfect",
	"Present Perfect Continuous",
	"Past Perfect Continuous",
	"Future Continuous",
	"Future Perfect",
	"Future Perfect Continuous",
}
