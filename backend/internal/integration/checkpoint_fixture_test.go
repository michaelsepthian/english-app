package integration

import (
	"context"
	"english-app/internal/model"
)

func (f *fakeAI) GeneratePlanCheckpointed(ctx context.Context, s model.Settings, p model.StudyPlan, h model.PreparationHooks) (model.Lesson, error) {
	if h.BeforeAttempt != nil {
		if err := h.BeforeAttempt("fixture", "Menyiapkan sesi pilihanmu", 20, 1); err != nil {
			return model.Lesson{}, err
		}
	}
	return plannedFixture(p), nil
}
