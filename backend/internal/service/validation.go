// MIVA-SERVICE: learning workflows, account rules and progress.
package service

import (
	"regexp"

	"english-app/internal/platform/apperror"
)

var idPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func validateAnswers(id string, answers map[string]int) error {
	if !idPattern.MatchString(id) || len(answers) > 30 {
		return apperror.NewError(400, "Jawaban belum valid.")
	}
	return nil
}
