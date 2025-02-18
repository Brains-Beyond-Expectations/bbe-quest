package interfaces

type UiServiceInterface interface {
	CreateSelect(title string, options []string) (string, error)
	CreateInput(title string, suggestion string) (string, error)
	CreateMultiChoose(title string, options []string, defaultIndex []int) ([]string, error)
}
