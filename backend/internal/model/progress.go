// MIVA-DOMAIN: learning data and pure validation rules.
package model

type AttemptRecord struct {
	Day     string
	Attempt Attempt
}
type LessonHistoryRecord struct {
	ID, Day, Title string
	Total          int
	Attempt        *Attempt
}
type LessonSummary struct {
	ID        string `json:"id"`
	Day       string `json:"day"`
	Title     string `json:"title"`
	Total     int    `json:"total"`
	Completed bool   `json:"completed"`
	Percent   int    `json:"percent"`
}
type SkillProgress struct {
	Correct int `json:"correct"`
	Total   int `json:"total"`
}
type ProgressAttempt struct {
	Day   string `json:"day"`
	Score int    `json:"score"`
	Total int    `json:"total"`
}
type WeeklyProgress struct {
	WeekStart string                   `json:"weekStart"`
	Lessons   int                      `json:"lessons"`
	Minutes   int                      `json:"minutes"`
	Seconds   int                      `json:"seconds"`
	Accuracy  int                      `json:"accuracy"`
	Days      []string                 `json:"days"`
	Attempts  []ProgressAttempt        `json:"attempts"`
	Skills    map[string]SkillProgress `json:"skills"`
}
