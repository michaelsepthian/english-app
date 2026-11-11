// MIVA-DOMAIN: learning data and pure validation rules.
package model

import (
	"errors"
	"math"
)

func Grade(l Lesson, answers map[string]int, seconds int) (Attempt, error) {
	a := Attempt{
		Total:               len(l.Questions),
		Seconds:             seconds,
		Results:             []Result{},
		ListeningTranscript: l.Listening.Transcript,
	}
	if len(answers) != len(l.Questions) {
		return a, errors.New("Jawab semua soal terlebih dahulu.")
	}
	for _, q := range l.Questions {
		n, ok := answers[q.ID]
		if !ok || n < 0 || n > 3 {
			return a, errors.New("Jawaban tidak valid.")
		}
		correct := n == q.Answer
		if correct {
			a.Score++
		}
		a.Results = append(a.Results, Result{q.ID, n, q.Answer, correct, q.Explanation, q.Skill})
	}
	a.Percent = Percent(a)
	a.ListeningTranscripts = map[string]string{}
	for _, item := range l.ListeningItems() {
		a.ListeningTranscripts[item.ID] = item.Transcript
	}
	return a, nil
}

func Percent(a Attempt) int {
	if a.Total == 0 {
		return 0
	}
	return int(math.Round(float64(a.Score) * 100 / float64(a.Total)))
}
