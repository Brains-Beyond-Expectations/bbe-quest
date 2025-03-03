package ui_service

import (
	"github.com/cqroot/prompt"
	"github.com/cqroot/prompt/multichoose"
)

type UiService struct{}

func (uiService UiService) CreateSelect(title string, options []string) (string, error) {
	result, err := prompt.New().Ask(title).
		Choose(options)
	if err != nil {
		return "", err
	}

	return result, nil
}

func (uiService UiService) CreateInput(title string, suggestion string) (string, error) {
	result, err := prompt.New().Ask(title).Input(suggestion)
	if err != nil {
		return "", err
	}

	return result, nil
}

func (uiService UiService) CreateMultiChoose(title string, options []string, defaultIndex []int) ([]string, error) {
	result, err := prompt.New().Ask(title).
		MultiChoose(options, multichoose.WithDefaultIndexes(0, defaultIndex))
	if err != nil {
		return nil, err
	}

	return result, nil
}
