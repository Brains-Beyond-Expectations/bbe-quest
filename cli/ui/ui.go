package ui

import (
	"github.com/cqroot/prompt"
)

func CreateSelect(title string, options []string) (string, error) {
	result, err := prompt.New().Ask(title).
		Choose(options)
	if err != nil {
		return "", err
	}

	return result, nil
}

func CreateInput(title string) (string, error) {
	result, err := prompt.New().Ask(title).Input("")
	if err != nil {
		return "", err
	}

	return result, nil
}
