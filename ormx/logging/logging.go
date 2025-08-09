package logging

import internal "go-ormx/ormx/internal/logging"

// Re-export Logger and helpers for external packages/examples

type Logger = internal.Logger
type LogLevel = internal.LogLevel
type LogField = internal.LogField

var (
	String   = internal.String
	Int      = internal.Int
	Int64    = internal.Int64
	Float64  = internal.Float64
	Bool     = internal.Bool
	Duration = internal.Duration
	Any      = internal.Any
)

const (
	Silent LogLevel = internal.Silent
	Error  LogLevel = internal.Error
	Warn   LogLevel = internal.Warn
	Info   LogLevel = internal.Info
)

type LoggerConfig = internal.LoggerConfig

func NewDBLogger(l Logger, cfg LoggerConfig) *internal.DBLogger { return internal.NewDBLogger(l, cfg) }
