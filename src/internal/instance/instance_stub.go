//go:build !windows

package instance

type Guard struct{}

func Acquire(string) (*Guard, bool, error) { return &Guard{}, true, nil }
func (*Guard) Release()                    {}
