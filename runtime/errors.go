package capwrap

import "fmt"

// Errorf returns an error tagged with the capwrap prefix. Use it in server
// implementations to signal a failure back to the caller.
func Errorf(format string, args ...any) error {
	return fmt.Errorf("capwrap: "+format, args...)
}

// WrapError annotates an error crossing the RPC boundary so callers can tell it
// came through capwrap. It returns nil when err is nil.
func WrapError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("capwrap: %w", err)
}
