package apps

import (
	"testing"

	g "winfastnav/internal/globals"
)

func TestExplicitAliasFindsApplication(t *testing.T) {
	appListMu.Lock()
	previousApps := g.AppList
	g.AppList = []g.Resource{
		{Name: "Visual Studio Code", Filepath: `C:\Apps\code.exe`, SearchName: "visual studio code", SearchPath: `c:\apps\code.exe`},
		{Name: "VLC", Filepath: `C:\Apps\vlc.exe`, SearchName: "vlc", SearchPath: `c:\apps\vlc.exe`},
	}
	appListMu.Unlock()
	aliasMu.Lock()
	previousAliases := aliases
	aliasMu.Unlock()
	t.Cleanup(func() {
		appListMu.Lock()
		g.AppList = previousApps
		appListMu.Unlock()
		aliasMu.Lock()
		aliases = previousAliases
		aliasMu.Unlock()
	})

	SetAliases("editor=Visual Studio Code; media=VLC")
	results := FindAppResults("editor")
	if len(results) == 0 || results[0].Name != "Visual Studio Code" {
		t.Fatalf("alias results = %#v", results)
	}
}

func TestMalformedAliasesAreIgnored(t *testing.T) {
	aliasMu.Lock()
	previousAliases := aliases
	aliasMu.Unlock()
	t.Cleanup(func() {
		aliasMu.Lock()
		aliases = previousAliases
		aliasMu.Unlock()
	})

	SetAliases("missing target; =blank; valid=Calculator")
	aliasMu.RLock()
	defer aliasMu.RUnlock()
	if len(aliases) != 1 || aliases["valid"] != "Calculator" {
		t.Fatalf("parsed aliases = %#v", aliases)
	}
}
