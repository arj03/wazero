//go:build !wazero_nointerp

package wazero

import "github.com/tetratelabs/wazero/internal/engine/interpreter"

// interpreterEngine backs engineKindInterpreter: NewRuntimeConfigInterpreter, and
// NewRuntimeConfig on a platform the compiler does not support.
var interpreterEngine newEngine = interpreter.NewEngine
