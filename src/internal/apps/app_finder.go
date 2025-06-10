package apps

import (
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows/registry"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	g "winfastnav/internal/globals"
	"winfastnav/internal/utils"
)

func GetInstalledApps() []g.App {
	keys := []registry.Key{
		registry.LOCAL_MACHINE,
		registry.CURRENT_USER,
	}
	basePaths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}

	// We only care about programs
	skipRelease := map[string]struct{}{
		"hotfix":          {},
		"security update": {},
		"service pack":    {},
		"update":          {},
	}

	// We don't care about stuff with these substr either
	skipIfSubstr := []string{
		"speech recognition",
		"redistributable",
		"x64-based systems",
		"application verifier",
		"install",
		"unins",
		"sdk",
		"runtime",
		"rundll32.exe",
	}

	var apps []g.App

	// Somehow not found by default
	apps = append(apps, g.App{Name: "Calculator", ExecPath: "calc.exe"})

	for _, keyRoot := range keys {
		for _, basePath := range basePaths {
			k, err := registry.OpenKey(keyRoot, basePath, registry.READ)

			if err != nil {
				continue
			}

			names, err := k.ReadSubKeyNames(-1)

			_ = k.Close()

			if err != nil {
				continue
			}

			// Go through each application subkey
			for _, name := range names {
				subKey, err := registry.OpenKey(keyRoot, basePath+`\`+name, registry.READ)
				if err != nil {
					continue
				}

				// Gotta have a name
				displayName, _, err := subKey.GetStringValue("DisplayName")
				if err != nil || strings.TrimSpace(displayName) == "" {
					_ = subKey.Close()
					continue
				}

				// no system components
				if sysVal, _, err := subKey.GetIntegerValue("SystemComponent"); err == nil && sysVal > 0 {
					_ = subKey.Close()
					continue
				}

				// skip releases in skipRelease
				if rel, _, err := subKey.GetStringValue("ReleaseType"); err == nil {
					if _, bad := skipRelease[strings.ToLower(rel)]; bad {
						_ = subKey.Close()
						continue
					}
				}

				execPath, _, err := subKey.GetStringValue("DisplayIcon")

				if err != nil || len(execPath) < 1 {
					_ = subKey.Close()
					continue
				}

				// Sometimes there's a comma and extra params, clear those out
				apps = append(apps, g.App{Name: strings.TrimSpace(displayName), ExecPath: cleanExecutablePath(execPath)})
				_ = subKey.Close()
			}
		}
	}

	apps = scanStartMenu(apps)

	var cleanApps []g.App

	// remove undesirables
	for i, app := range apps {
		if !(!strings.Contains(app.ExecPath, ".exe") || utils.ContainsAny(app.ExecPath, skipIfSubstr) ||
			utils.ContainsAny(strings.ToLower(app.Name), skipIfSubstr) || utils.ContainsAny(app.ExecPath, g.ExecBlocklist)) {
			cleanApps = append(cleanApps, apps[i])
		}
	}

	// sort by name
	sort.Slice(cleanApps, func(i, j int) bool {
		return strings.ToLower(cleanApps[i].Name) < strings.ToLower(apps[j].Name)
	})

	return cleanApps
}

func cleanExecutablePath(path string) string {
	if i := strings.Index(path, ","); i != -1 {
		path = path[:i]
	}
	return strings.TrimSpace(strings.ToLower(path))
}

func resolveShortcut(path string) (string, error) {
	err := ole.CoInitialize(0)
	if err != nil {
		return "", err
	}
	defer ole.CoUninitialize()

	wshObj, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return "", err
	}
	defer wshObj.Release()

	wsh, err := wshObj.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return "", err
	}
	defer wsh.Release()

	scRaw, err := oleutil.CallMethod(wsh, "CreateShortcut", path)
	if err != nil {
		return "", err
	}
	sc := scRaw.ToIDispatch()
	defer sc.Release()

	tp, err := oleutil.GetProperty(sc, "TargetPath")
	if err != nil {
		return "", err
	}
	return tp.ToString(), nil
}

// Search for programs by grabbing .lnk's off the start menu
func scanStartMenu(currentAppList []g.App) []g.App {
	dirs := []string{
		filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs"),
		filepath.Join(os.Getenv("PROGRAMDATA"), "Microsoft", "Windows", "Start Menu", "Programs"),
	}

	for _, base := range dirs {
		err := filepath.WalkDir(base, func(p string, de fs.DirEntry, err error) error {
			if err != nil || de.IsDir() || !strings.HasSuffix(strings.ToLower(p), ".lnk") {
				return nil
			}
			target, err := resolveShortcut(p)
			if err != nil || target == "" {
				return nil
			}
			name := strings.TrimSuffix(de.Name(), ".lnk")

			// No repeats
			for _, app := range currentAppList {
				if strings.EqualFold(app.ExecPath, target) || strings.EqualFold(app.Name, name) {
					return nil
				}
			}

			currentAppList = append(currentAppList, g.App{Name: strings.TrimSpace(name), ExecPath: strings.ToLower(target)})
			return nil
		})
		if err != nil {
			return nil
		}
	}
	return currentAppList
}
