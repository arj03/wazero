//go:build wazero_nointerp

package wazero

// interpreterEngine is nil under wazero_nointerp, which drops the interpreter from the
// binary. A runtime that would pick it panics instead: NewRuntimeConfigInterpreter, or
// NewRuntimeConfig on a platform the compiler does not support.
var interpreterEngine newEngine
