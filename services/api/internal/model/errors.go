package model

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("username already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenNotFound      = errors.New("token not found")
	ErrTokenExpired       = errors.New("token expired")
	ErrForbidden          = errors.New("forbidden")
	ErrChatNotFound       = errors.New("chat not found")
	ErrChatAlreadyExists  = errors.New("direct chat already exists")
	ErrMessageNotFound    = errors.New("message not found")
)
