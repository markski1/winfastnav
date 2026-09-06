package apps

import (
	"testing"

	g "winfastnav/internal/globals"
)

func TestSameResources(t *testing.T) {
	left := []g.Resource{{Name: "Calculator", Filepath: `C:\Windows\System32\calc.exe`}}
	right := append([]g.Resource(nil), left...)
	if !sameResources(left, right) {
		t.Fatal("identical catalogs should compare equal")
	}
	right[0].Name = "Different"
	if sameResources(left, right) {
		t.Fatal("different catalogs should not compare equal")
	}
}

func TestFilterCachedAppsHonorsBlocklist(t *testing.T) {
	previous := g.ExecBlocklist
	g.ExecBlocklist = []string{`C:\Apps\hidden.exe`}
	t.Cleanup(func() { g.ExecBlocklist = previous })
	apps := []g.Resource{
		{Name: "Visible", Filepath: `C:\Apps\visible.exe`},
		{Name: "Hidden", Filepath: `C:\Apps\hidden.exe`},
	}
	filtered := filterCachedApps(apps)
	if len(filtered) != 1 || filtered[0].Name != "Visible" {
		t.Fatalf("filtered catalog = %#v", filtered)
	}
}

func TestCatalogRoundTripPreparesSearchFields(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	appListMu.Lock()
	previousApps := g.AppList
	appListMu.Unlock()
	catalogMu.Lock()
	previousFingerprint := catalogFingerprint
	catalogMu.Unlock()
	t.Cleanup(func() {
		appListMu.Lock()
		g.AppList = previousApps
		appListMu.Unlock()
		catalogMu.Lock()
		catalogFingerprint = previousFingerprint
		catalogMu.Unlock()
	})

	want := g.Resource{Name: "Example App", Filepath: `C:\Apps\Example.exe`}
	if err := saveCatalog(catalogFile{Version: catalogVersion, Fingerprint: 42, Apps: []g.Resource{want}}); err != nil {
		t.Fatal(err)
	}
	LoadCatalog()
	appListMu.RLock()
	defer appListMu.RUnlock()
	if len(g.AppList) != 1 || g.AppList[0].SearchName != "example app" || g.AppList[0].SearchPath != `c:\apps\example.exe` {
		t.Fatalf("loaded catalog = %#v", g.AppList)
	}
}
