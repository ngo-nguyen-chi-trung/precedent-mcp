package model

import (
	"errors"
	"fmt"

	"github.com/ngo/precedent_mcp/internal/env"
)

type Error struct {
	Code    env.AssetErrorCode
	Message string
	Data    any
}

func (e *Error) Error() string {
	return e.Message
}

type SkillErrorData struct {
	Name string
}

func ErrSkillNotFound(name string) error {
	return &Error{
		Code:    env.AssetErrorSkillNotFound,
		Message: fmt.Sprintf("skill %s not found", name),
		Data:    SkillErrorData{Name: name},
	}
}

type ExamplesErrorData struct {
	Name string
}

func ErrExamplesNotFound(name string) error {
	return &Error{
		Code:    env.AssetErrorExamplesNotFound,
		Message: fmt.Sprintf("examples %s not found", name),
		Data:    ExamplesErrorData{Name: name},
	}
}

func ErrInvalidMetadata(name string, err error) error {
	return &Error{
		Code:    env.AssetErrorInvalidMetadata,
		Message: fmt.Sprintf("invalid metadata: %v", err),
		Data:    SkillErrorData{Name: name},
	}
}

func ErrSkillNameMismatch(name, dir string) error {
	return &Error{
		Code:    env.AssetErrorSkillNameMismatch,
		Message: fmt.Sprintf("skill name %q does not match directory %q", name, dir),
		Data:    SkillErrorData{Name: dir},
	}
}

func ErrDescriptionLength(name string, n int) error {
	return &Error{
		Code:    env.AssetErrorDescriptionLength,
		Message: fmt.Sprintf("description must be 1 to %d characters, got %d", env.LimitSkillDescription, n),
		Data:    SkillErrorData{Name: name},
	}
}

func HasCode(err error, code env.AssetErrorCode) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == code
}
