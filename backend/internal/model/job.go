// MIVA-DOMAIN: learning data and pure validation rules.
package model

type StudyJob struct {
	ID               string    `json:"id"`
	RequestID        string    `json:"-"`
	Status           string    `json:"status"`
	Stage            string    `json:"stage"`
	Progress         int       `json:"progress"`
	Error            string    `json:"error"`
	LessonID         *string   `json:"lessonId"`
	Plan             StudyPlan `json:"plan"`
	Settings         *Settings `json:"-"`
	GeneratorVersion int       `json:"-"`
	CanResume        bool      `json:"canResume"`
	Attempt          int       `json:"attempt"`
	MaxAttempts      int       `json:"maxAttempts"`
	Goal             string    `json:"goal"`
	Level            string    `json:"level"`
}
