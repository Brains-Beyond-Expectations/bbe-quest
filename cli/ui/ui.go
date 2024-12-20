package ui

import (
	"github.com/cqroot/prompt"
	"github.com/cqroot/prompt/multichoose"
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

func CreateMultiChoose(title string, options []string, defaultIndex []int) ([]string, error) {
	result, err := prompt.New().Ask(title).
		MultiChoose(options, multichoose.WithDefaultIndexes(0, defaultIndex))
	if err != nil {
		return nil, err
	}

	return result, nil
}
