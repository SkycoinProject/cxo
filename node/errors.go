package node

import (
	"errors"
)

// common errors
var (
	ErrAlreadyListen         = errors.New("already listen")
	ErrTimeout               = errors.New("timeout")
	ErrClosed                = errors.New("closed")
	ErrNotPublic             = errors.New("not a public server")
	ErrAlreadyHaveConnection = errors.New("already have connection")
	ErrInvalidResponse       = errors.New("invalid response")
)
