package plugins

import "errors"

var (
	ErrPluginNotFound     = errors.New("plugin not found")
	ErrPluginNotEnabled   = errors.New("plugin not enabled")
	ErrInvalidConfig      = errors.New("invalid plugin configuration")
	ErrSafetyLimitReached = errors.New("safety limit reached")
	ErrPluginExecution    = errors.New("plugin execution failed")
)