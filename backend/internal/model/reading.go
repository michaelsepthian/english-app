// MIVA-DOMAIN: learning data and pure validation rules.
package model

type ReadingWord struct {
	Word     string  `json:"word"`
	Heard    string  `json:"heard,omitempty"`
	Status   string  `json:"status"`
	Accuracy float64 `json:"accuracy,omitempty"`
}

type ReadingCheck struct {
	ID           string        `json:"id"`
	SourceID     string        `json:"sourceId"`
	Mode         string        `json:"mode"`
	Label        string        `json:"label"`
	Score        float64       `json:"score"`
	Accuracy     float64       `json:"accuracy,omitempty"`
	Fluency      float64       `json:"fluency,omitempty"`
	Completeness float64       `json:"completeness,omitempty"`
	Reference    string        `json:"reference"`
	Transcript   string        `json:"transcript"`
	Words        []ReadingWord `json:"words"`
	Feedback     string        `json:"feedback"`
}
