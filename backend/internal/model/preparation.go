package model

import "encoding/json"

const GeneratorVersion = 1
const PreparationAttempts = 3

// PreparationHooks keep checkpoint payloads internal to the worker.
type PreparationHooks struct {
	JobID         string
	Checkpoints   map[string]json.RawMessage
	BeforeAttempt func(key, stage string, progress, attempt int) error
	Save          func(key string, payload json.RawMessage, progress int) error
}
