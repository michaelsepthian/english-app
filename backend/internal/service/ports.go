// Package service owns application rules. Dependencies are interfaces; no SQL or HTTP transport belongs here.
package service

import (
	"context"

	"english-app/internal/model"
)

type AI interface {
	Generate(context.Context, model.Settings, []string) (model.Lesson, error)
	Lookup(context.Context, string) (model.Word, error)
	Chat(context.Context, model.Settings, []model.Message, string) (string, error)
}
type Speech interface {
	Assess(context.Context, string, []byte) (model.ReadingCheck, error)
}
