//go:build darwin

package main

/*
#cgo LDFLAGS: -framework UniformTypeIdentifiers
*/
import "C"

// Wails v2.13 needs this framework for its native save dialog on current SDKs.
