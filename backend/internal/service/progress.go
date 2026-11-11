// MIVA-PROGRESS: aggregate completed attempts for the user's current week.
package service

import (
	"context"
	"time"

	"english-app/internal/model"
	"english-app/internal/repository"
)

type ProgressService struct{ repo repository.Lessons }

func (s *ProgressService) Weekly(ctx context.Context, uid string) (model.WeeklyProgress, error) {
	progress := model.WeeklyProgress{
		WeekStart: model.Monday(time.Now()),
		Days:      []string{},
		Attempts:  []model.ProgressAttempt{},
		Skills:    map[string]model.SkillProgress{},
	}
	rows, err := s.repo.Attempts(ctx, uid, progress.WeekStart)
	if err != nil {
		return progress, err
	}
	for _, skill := range []string{"vocabulary", "tenses", "reading", "listening"} {
		progress.Skills[skill] = model.SkillProgress{}
	}
	seen := map[string]bool{}
	score, total := 0, 0
	for _, row := range rows {
		attempt := row.Attempt
		if !seen[row.Day] {
			progress.Days = append(progress.Days, row.Day)
			seen[row.Day] = true
		}
		score += attempt.Score
		total += attempt.Total
		progress.Seconds += attempt.Seconds
		progress.Attempts = append(progress.Attempts, model.ProgressAttempt{Day: row.Day, Score: attempt.Score, Total: attempt.Total})
		for _, result := range attempt.Results {
			if skill, ok := progress.Skills[result.Skill]; ok {
				skill.Total++
				if result.Correct {
					skill.Correct++
				}
				progress.Skills[result.Skill] = skill
			}
		}
	}
	progress.Lessons = len(progress.Attempts)
	progress.Minutes = progress.Seconds / 60
	progress.Accuracy = model.Percent(model.Attempt{Score: score, Total: total})
	return progress, nil
}
