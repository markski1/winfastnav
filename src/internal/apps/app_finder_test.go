package apps

import "testing"

func TestCleanExecutablePathRemovesQuotedIconIndex(t *testing.T) {
	path := cleanExecutablePath(`"C:\Users\example\AppData\Local\Discord\app-1.0\Discord.exe",0`)
	want := `c:\users\example\appdata\local\discord\app-1.0\discord.exe`
	if path != want {
		t.Fatalf("cleanExecutablePath() = %q, want %q", path, want)
	}
}
