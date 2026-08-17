package providers

import "fmt"

type ProviderError struct {
	Provider   string
	Message    string
	StatusCode int
	Retriable  bool
}

func (e *ProviderError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s provider error (%d): %s", e.Provider, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("%s provider error: %s", e.Provider, e.Message)
}

func ToProviderMessage(err error) string {
	if err == nil {
		return ""
	}
	if pe, ok := err.(*ProviderError); ok {
		return pe.Message
	}
	return err.Error()
}
