//go:build !windows

package windowcontrol

type Controller struct{}

func New(string) *Controller            { return &Controller{} }
func (*Controller) BindView(any)        {}
func (*Controller) Bind() error         { return nil }
func (*Controller) Hide() error         { return nil }
func (*Controller) ShowAndFocus() error { return nil }
