// MIVA-SERVICE: learning workflows, account rules and progress.
package service

import (
	"math"
	"regexp"
	"strings"
	"unicode"

	"english-app/internal/model"
	"english-app/internal/platform/identity"
)

func readingSentences(text string) []string {
	parts := regexp.MustCompile(`[^.!?]+(?:[.!?]+|$)`).FindAllString(text, -1)
	out := []string{}
	for _, s := range parts {
		v := strings.TrimSpace(s)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func wordTokens(text string) []string {
	return strings.FieldsFunc(strings.ToLower(strings.ReplaceAll(text, "’", "'")), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '\'' })
}

func CompareWords(reference, transcript string) model.ReadingCheck {
	a, b := wordTokens(reference), wordTokens(transcript)
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
		dp[i][0] = i
	}
	for j := 0; j <= m; j++ {
		dp[0][j] = j
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			dp[i][j] = min(dp[i-1][j]+1, dp[i][j-1]+1, dp[i-1][j-1]+cost)
		}
	}
	words := []model.ReadingWord{}
	for i, j := n, m; i > 0 || j > 0; {
		if i > 0 && j > 0 && a[i-1] == b[j-1] && dp[i][j] == dp[i-1][j-1] {
			words = append(words, model.ReadingWord{Word: a[i-1], Heard: b[j-1], Status: "correct"})
			i--
			j--
		} else if i > 0 && j > 0 && dp[i][j] == dp[i-1][j-1]+1 {
			words = append(words, model.ReadingWord{Word: a[i-1], Heard: b[j-1], Status: "changed"})
			i--
			j--
		} else if i > 0 && dp[i][j] == dp[i-1][j]+1 {
			words = append(words, model.ReadingWord{Word: a[i-1], Status: "missing"})
			i--
		} else {
			words = append(words, model.ReadingWord{Word: b[j-1], Heard: b[j-1], Status: "extra"})
			j--
		}
	}
	for i, j := 0, len(words)-1; i < j; i, j = i+1, j-1 {
		words[i], words[j] = words[j], words[i]
	}
	score := 0.
	if n > 0 {
		score = math.Max(0, 100*(1-float64(dp[n][m])/float64(n)))
	}
	return model.ReadingCheck{
		ID:         identity.NewID(),
		Mode:       "words",
		Label:      "Kecocokan kata",
		Score:      math.Round(score),
		Reference:  reference,
		Transcript: transcript,
		Words:      words,
		Feedback:   "Ini perbandingan kata yang terbaca oleh browser, bukan skor bunyi pengucapan. Dengarkan contoh, lalu ulangi kata yang berbeda atau terlewat.",
	}
}
