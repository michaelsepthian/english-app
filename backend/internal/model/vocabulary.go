// MIVA-DOMAIN: learning data and pure validation rules.
package model

import (
	"errors"
)

type Word struct {
	ID            string    `json:"id,omitempty"`
	Word          string    `json:"word"`
	Meaning       string    `json:"meaning"`
	Pronunciation string    `json:"pronunciation"`
	PartOfSpeech  string    `json:"partOfSpeech"`
	V1            string    `json:"v1"`
	V2            string    `json:"v2"`
	V3            string    `json:"v3"`
	Example       string    `json:"example"`
	Translation   string    `json:"translation"`
	Source        string    `json:"source,omitempty"`
	Examples      []Example `json:"examples,omitempty"`
}

type Example struct {
	EN string `json:"en"`
	ID string `json:"id"`
}

func (w Word) Validate() error {
	if len(w.Word) == 0 || len(w.Word) > 100 || len(w.Meaning) == 0 || len(w.Meaning) > 600 || len(w.Example) == 0 || len(w.Example) > 700 || len(w.Translation) == 0 || len(w.V1) > 100 || len(w.V2) > 100 || len(w.V3) > 100 {
		return errors.New("Entri kamus AI belum lengkap.")
	}
	if w.PartOfSpeech == "verb" && (w.V1 == "" || w.V2 == "" || w.V3 == "") {
		return errors.New("Bentuk kata kerja AI belum lengkap.")
	}
	return nil
}

var StarterWords = []Word{
	{
		ID:           "starter-go",
		Word:         "go",
		Meaning:      "pergi",
		PartOfSpeech: "verb",
		V1:           "go",
		V2:           "went",
		V3:           "gone",
		Example:      "I go to work by train.",
		Translation:  "Saya pergi bekerja naik kereta.",
		Source:       "starter",
	},
	{
		ID:           "starter-eat",
		Word:         "eat",
		Meaning:      "makan",
		PartOfSpeech: "verb",
		V1:           "eat",
		V2:           "ate",
		V3:           "eaten",
		Example:      "She has eaten breakfast.",
		Translation:  "Dia sudah sarapan.",
		Source:       "starter",
	},
	{
		ID:           "starter-speak",
		Word:         "speak",
		Meaning:      "berbicara",
		PartOfSpeech: "verb",
		V1:           "speak",
		V2:           "spoke",
		V3:           "spoken",
		Example:      "We spoke English yesterday.",
		Translation:  "Kami berbicara bahasa Inggris kemarin.",
		Source:       "starter",
	},
	{
		ID:           "starter-write",
		Word:         "write",
		Meaning:      "menulis",
		PartOfSpeech: "verb",
		V1:           "write",
		V2:           "wrote",
		V3:           "written",
		Example:      "I have written a short story.",
		Translation:  "Saya sudah menulis cerita pendek.",
		Source:       "starter",
	},
	{
		ID:            "starter-read",
		Word:          "read",
		Meaning:       "membaca",
		Pronunciation: "V1 /riːd/; V2 & V3 /red/",
		PartOfSpeech:  "verb",
		V1:            "read",
		V2:            "read",
		V3:            "read",
		Example:       "I read a book every evening.",
		Translation:   "Saya membaca buku setiap malam.",
		Source:        "starter",
	},
	{
		ID:           "starter-take",
		Word:         "take",
		Meaning:      "mengambil",
		PartOfSpeech: "verb",
		V1:           "take",
		V2:           "took",
		V3:           "taken",
		Example:      "He took a photo.",
		Translation:  "Dia mengambil foto.",
		Source:       "starter",
	},
	{
		ID:           "starter-make",
		Word:         "make",
		Meaning:      "membuat",
		PartOfSpeech: "verb",
		V1:           "make",
		V2:           "made",
		V3:           "made",
		Example:      "We made dinner together.",
		Translation:  "Kami membuat makan malam bersama.",
		Source:       "starter",
	},
	{
		ID:           "starter-learn",
		Word:         "learn",
		Meaning:      "belajar",
		PartOfSpeech: "verb",
		V1:           "learn",
		V2:           "learned",
		V3:           "learned",
		Example:      "I learn something new every day.",
		Translation:  "Saya belajar hal baru setiap hari.",
		Source:       "starter",
	},
	{
		ID:           "starter-drink",
		Word:         "drink",
		Meaning:      "minum",
		PartOfSpeech: "verb",
		V1:           "drink",
		V2:           "drank",
		V3:           "drunk",
		Example:      "She drank a glass of water.",
		Translation:  "Dia minum segelas air.",
		Source:       "starter",
	},
	{
		ID:           "starter-see",
		Word:         "see",
		Meaning:      "melihat",
		PartOfSpeech: "verb",
		V1:           "see",
		V2:           "saw",
		V3:           "seen",
		Example:      "I have seen that film.",
		Translation:  "Saya sudah menonton film itu.",
		Source:       "starter",
	},
	{
		ID:           "starter-buy",
		Word:         "buy",
		Meaning:      "membeli",
		PartOfSpeech: "verb",
		V1:           "buy",
		V2:           "bought",
		V3:           "bought",
		Example:      "I bought a new notebook.",
		Translation:  "Saya membeli buku catatan baru.",
		Source:       "starter",
	},
	{
		ID:           "starter-work",
		Word:         "work",
		Meaning:      "bekerja",
		PartOfSpeech: "verb",
		V1:           "work",
		V2:           "worked",
		V3:           "worked",
		Example:      "He worked from home yesterday.",
		Translation:  "Dia bekerja dari rumah kemarin.",
		Source:       "starter",
	},
}
