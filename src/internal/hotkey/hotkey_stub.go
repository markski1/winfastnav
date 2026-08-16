//go:build !windows

package hotkey

type Listener struct{}

func Start(func()) (*Listener, error) { return &Listener{}, nil }
func (*Listener) Stop()               {}
