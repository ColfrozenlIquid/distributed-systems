package vshttp

import "fmt"

type HTTPError struct {
	Status  StatusCode
	Message string
	Err     error
}

func Error(status StatusCode, message string) *HTTPError {
	return &HTTPError{
		Status:  status,
		Message: message,
	}
}

func WrapError(status StatusCode, message string, err error) *HTTPError {
	return &HTTPError{
		Status:  status,
		Message: message,
		Err:     err,
	}
}

func (e *HTTPError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *HTTPError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
