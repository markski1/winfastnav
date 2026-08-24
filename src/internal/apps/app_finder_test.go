package apps

import (
	"path/filepath"
	"testing"

	g "winfastnav/internal/globals"
)

func TestCleanExecutablePathRemovesQuotedIconIndex(t *testing.T) {
	path := cleanExecutablePath(`"C:\Users\example\AppData\Local\Discord\app-1.0\Discord.exe",0`)
	want := `c:\users\example\appdata\local\discord\app-1.0\discord.exe`
	if path != want {
		t.Fatalf("cleanExecutablePath() = %q, want %q", path, want)
	}
}

func TestStartMenuEntryWinsOverRegistryMetadata(t *testing.T) {
	resources := []g.Resource{{
		Name:     "Example Launcher",
		Filepath: `C:\Program Files\Example\launcher.exe`,
	}}

	if !hasApplication(resources, "Example Launcher", `C:\Program Files\Example\uninstall.exe`) {
		t.Fatal("a Start Menu application should prevent lower-priority registry metadata from replacing it")
	}
}

func TestAppsFolderPathsAreLaunchable(t *testing.T) {
	path := appsFolderPrefix + "Example.Package_123!App"
	if !isLaunchableApplication(path) {
		t.Fatalf("isLaunchableApplication(%q) = false, want true", path)
	}
	if !isAppsFolderPath(path) {
		t.Fatalf("isAppsFolderPath(%q) = false, want true", path)
	}
}

func TestCalculatorResource(t *testing.T) {
	calculator := calculatorResource()
	if calculator.Name != "Calculator" || filepath.Base(calculator.Filepath) != "calc.exe" {
		t.Fatalf("calculatorResource() = %#v", calculator)
	}
}

func TestAllowedApplication(t *testing.T) {
	skip := []string{"updater"}
	tests := []struct {
		name string
		app  g.Resource
		want bool
	}{
		{name: "executable", app: g.Resource{Name: "Example", Filepath: `C:\Apps\example.exe`}, want: true},
		{name: "registered app", app: g.Resource{Name: "Example", Filepath: appsFolderPrefix + "Example.Package_123!App"}, want: true},
		{name: "non-application", app: g.Resource{Name: "Example", Filepath: `C:\Files\example.txt`}, want: false},
		{name: "filtered name", app: g.Resource{Name: "Example Updater", Filepath: `C:\Apps\example.exe`}, want: false},
	}

	for _, test := range tests {
		if got := isAllowedApplication(test.app, skip, nil); got != test.want {
			t.Errorf("%s: isAllowedApplication() = %t, want %t", test.name, got, test.want)
		}
	}
}

func TestInstalledAppsAreDiscovered(t *testing.T) {
	if len(GetInstalledApps()) == 0 {
		t.Fatal("no applications were discovered")
	}
}
