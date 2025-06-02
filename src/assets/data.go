package assets

type App struct {
	Name     string
	ExecPath string
}

var (
	AppList       []App
	ExecBlocklist []string
)
