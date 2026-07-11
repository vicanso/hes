package hes

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// FileConverter rewrites captured source paths (e.g. strip build prefixes).
type FileConverter func(file string) string

var (
	defaultCaller atomic.Bool

	converterMu   sync.RWMutex
	fileConverter FileConverter
)

// SetDefaultCaller enables or disables automatic caller capture for New/Wrap.
func SetDefaultCaller(enabled bool) {
	defaultCaller.Store(enabled)
}

// SetFileConverter sets a path rewriter used when capturing callers.
// Pass nil to clear.
func SetFileConverter(fn FileConverter) {
	converterMu.Lock()
	defer converterMu.Unlock()
	fileConverter = fn
}

// captureCaller is like runtime.Caller after applying the file converter.
// skip is relative to captureCaller itself (0 = captureCaller).
func captureCaller(skip int) (file string, line int, ok bool) {
	_, file, line, ok = runtime.Caller(skip)
	if !ok {
		return "", 0, false
	}
	converterMu.RLock()
	fn := fileConverter
	converterMu.RUnlock()
	if fn != nil {
		file = fn(file)
	}
	return file, line, true
}

// setCaller records the call site. skip is frames above setCaller
// (0 = setCaller's direct caller).
func (e *Error) setCaller(skip int) {
	if e == nil {
		return
	}
	// Frames: captureCaller=0, setCaller=1, setCaller caller=2+skip.
	file, line, ok := captureCaller(skip + 2)
	if !ok {
		return
	}
	e.File = file
	e.Line = line
}
