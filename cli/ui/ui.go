package ui

import (
	"github.com/cqroot/prompt"
)

func CreateModal(title string, options []string) (string, error) {
	val1, err := prompt.New().Ask(title).
		Choose(options)
	if err != nil {
		return "", err
	}

	return val1, nil
}
