package server

// GracefulRestartSignal implements a signal to be used for a graceful restart trigger.
type GracefulRestartSignal struct{}

// String is a part of os.Signal interface to represent a signal as a string.
func (g GracefulRestartSignal) String() string {
	return "GracefulRestartSignal"
}

// Signal is a part of os.Signal interface doing nothing.
func (g GracefulRestartSignal) Signal() {}
