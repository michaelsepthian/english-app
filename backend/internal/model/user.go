// MIVA-DOMAIN: learning data and pure validation rules.
package model

import (
	"errors"
	"strings"
	"time"
)

type Settings struct {
	Name          string `json:"name"`
	Level         string `json:"level"`
	Goal          string `json:"goal"`
	WeeklyMinutes int    `json:"weeklyMinutes"`
	WeeklyLessons int    `json:"weeklyLessons"`
	Reminder      string `json:"reminder"`
	Days          []int  `json:"days"`
	Tense         string `json:"tense"`
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Settings
}

func DefaultSettings(name string) Settings {
	return Settings{
		Name:          name,
		Level:         "A1",
		Goal:          "Bisa mengobrol tentang kegiatan sehari-hari dengan percaya diri.",
		WeeklyMinutes: 75,
		WeeklyLessons: 5,
		Reminder:      "20:00",
		Days:          []int{0, 1, 2, 3, 4},
		Tense:         Tenses[0],
	}
}

func (s Settings) Validate() error {
	if len(strings.TrimSpace(s.Name)) < 2 || len(s.Name) > 120 || len(strings.TrimSpace(s.Goal)) < 5 || len(s.Goal) > 600 || s.WeeklyMinutes < 15 || s.WeeklyMinutes > 1200 || s.WeeklyLessons < 1 || s.WeeklyLessons > 7 {
		return errors.New("Periksa nama, goal, dan target mingguan.")
	}
	if !Contains([]string{"A1", "A2", "B1", "B2"}, s.Level) || !Contains(Tenses, s.Tense) {
		return errors.New("Level atau tenses belum valid.")
	}
	if _, err := time.Parse("15:04", s.Reminder); err != nil || len(s.Reminder) != 5 {
		return errors.New("Jadwal belum valid.")
	}
	seen := map[int]bool{}
	for _, v := range s.Days {
		if v < 0 || v > 6 || seen[v] {
			return errors.New("Hari belajar belum valid.")
		}
		seen[v] = true
	}
	if len(seen) == 0 || len(seen) > 7 {
		return errors.New("Pilih setidaknya satu hari belajar.")
	}
	return nil
}
