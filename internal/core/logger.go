package core

import (
	"log"

	"github.com/julien-sobczak/the-notewriter/pkg/resync"
)

var (
	// Lazy-load and ensure a single read
	loggerOnce      resync.Once
	loggerSingleton *Logger
)

type VerboseLevel int

const (
	VerboseOff VerboseLevel = iota
	VerboseInfo
	VerboseDebug
	VerboseTrace
)

func CurrentLogger() *Logger {
	loggerOnce.Do(func() {
		loggerSingleton = NewLogger()
	})
	return loggerSingleton
}

type Logger struct {
	verbose VerboseLevel
}

func NewLogger() *Logger {
	return &Logger{
		verbose: VerboseOff,
	}
}

// SetVerboseLevel overrides the default verbose level
func (l *Logger) SetVerboseLevel(level VerboseLevel) *Logger {
	l.verbose = level
	return l
}

func (l *Logger) Fatal(v ...any) {
	log.Fatalln(v...)
}
func (l *Logger) Fatalf(format string, v ...any) {
	log.Fatalf(format, v...)
}

func (l *Logger) Warn(v ...any) {
	log.Println(v...)
}
func (l *Logger) Warnf(format string, v ...any) {
	log.Printf(format, v...)
}

func (l *Logger) Info(v ...any) {
	if l.verbose >= VerboseInfo {
		log.Println(v...)
	}
}
func (l *Logger) Infof(format string, v ...any) {
	if l.verbose >= VerboseInfo {
		log.Printf(format, v...)
	}
}

func (l *Logger) Debug(v ...any) {
	if l.verbose >= VerboseDebug {
		log.Println(v...)
	}
}
func (l *Logger) Debugf(format string, v ...any) {
	if l.verbose >= VerboseDebug {
		log.Printf(format, v...)
	}
}

func (l *Logger) Trace(v ...any) {
	if l.verbose >= VerboseTrace {
		log.Println(v...)
	}
}
func (l *Logger) Tracef(format string, v ...any) {
	if l.verbose >= VerboseTrace {
		log.Printf(format, v...)
	}
}
