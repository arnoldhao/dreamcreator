package skills

import "errors"

var (
	ErrInvalidSkill   = errors.New("invalid skill")
	ErrSkillNotFound  = errors.New("skill not found")
	ErrNotImplemented = errors.New("not implemented")
)
