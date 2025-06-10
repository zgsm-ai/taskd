package utils

import "fmt"

/*
 * HTTP error wrapper structure
 * @param code HTTP status code
 * @param err Original error object
 */
type HttpError struct {
	code int
	err  error
}

/*
 * Get HTTP status code
 * @return int HTTP status code
 */
func (e *HttpError) Code() int {
	return e.code
}

/*
 * Implement error interface, return error message
 * @return string Error message
 */
func (e *HttpError) Error() string {
	return e.err.Error()
}

/*
 * Get original error object
 * @return error Original error object
 */
func (e *HttpError) Origin() error {
	return e.err
}

/*
 * Create new HTTP error
 * @param code HTTP status code
 * @param msg Error message
 * @return *HttpError HTTP error object
 */
func NewHttpError(code int, msg string) *HttpError {
	return &HttpError{
		code: code,
		err:  fmt.Errorf("%s", msg),
	}
}

/*
 * Re-wrap error as HTTP error
 * @param code HTTP status code
 * @param err Original error
 * @return *HttpError HTTP error object
 */
func RethrowError(code int, err error) *HttpError {
	return &HttpError{
		code: code,
		err:  err,
	}
}
