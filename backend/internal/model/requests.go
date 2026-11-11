// MIVA-DOMAIN: learning data and pure validation rules.
package model

// Service inputs are transport-independent. Audio is never serialized into history.
type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}
type PrepareRequest struct {
	Plan      StudyPlan `json:"plan"`
	RequestID string    `json:"requestId"`
}
type AnswerRequest struct {
	ID      string         `json:"id"`
	Answers map[string]int `json:"answers"`
}
type HeartbeatRequest struct {
	ID     string `json:"id"`
	Active bool   `json:"active"`
}
type AudioRequest struct {
	Kind         string `json:"kind"`
	ID           string `json:"id"`
	Part         string `json:"part"`
	SourceID     string `json:"sourceId"`
	ExampleIndex int    `json:"exampleIndex"`
}
type ReadingRequest struct {
	LessonID      string `json:"lessonId"`
	SourceID      string `json:"sourceId"`
	SentenceIndex int    `json:"sentenceIndex"`
	Transcript    string `json:"transcript"`
	Audio         []byte `json:"-"`
}
type IDRequest struct {
	ID string `json:"id"`
}
type LookupRequest struct {
	Query string `json:"query"`
}
type ChatRequest struct {
	Text string `json:"text"`
}
