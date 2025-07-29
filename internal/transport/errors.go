package transport

import "errors"

var (
	// ErrStreamableHTTPNotImplemented はStreamable HTTPがまだ実装されていないことを示すエラー
	ErrStreamableHTTPNotImplemented = errors.New("streamable HTTP transport is not yet implemented")
)
