package apps

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows/registry"

	g "winfastnav/internal/globals"
	"winfastnav/internal/utils"
)

const appsFolderPrefix = `shell:AppsFolder\`

func GetInstalledApps() []g.Resource {
	appListMu.RLock()
	blocklist := append([]string(nil), g.ExecBlocklist...)
	appListMu.RUnlock()

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
		"teams meeting add-in",
		"teams machine-wide installer",
		"ms teams",
		"msteams",
		"unins",
		"sdk",
		"runtime",
		"rundll32.exe",
	}

	var apps []g.Resource

	apps = append(apps, calculatorResource())
	apps = scanAppsFolder(apps)
	// Windows Search presents Start Menu launchers. Index them before the registry metadata stuff
	// because these will be less likely to be shit.
	apps = scanStartMenu(apps)
	apps = scanAppPaths(apps)

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

				execPath = cleanExecutablePath(execPath)
				if execPath != "" && strings.HasSuffix(execPath, ".exe") && !hasApplication(apps, displayName, execPath) {
					apps = append(apps, g.Resource{Name: strings.TrimSpace(displayName), Filepath: execPath})
				}
				_ = subKey.Close()
			}
		}
	}
	var cleanApps []g.Resource

	// remove undesirables
	for i, app := range apps {
		if isAllowedApplication(app, skipIfSubstr, blocklist) {
			cleanApps = append(cleanApps, apps[i])
		}
	}

	// sort by name
	sort.Slice(cleanApps, func(i, j int) bool {
		return strings.ToLower(cleanApps[i].Name) < strings.ToLower(cleanApps[j].Name)
	})

	return cleanApps
}

func calculatorResource() g.Resource {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		return g.Resource{Name: "Calculator", Filepath: "calc.exe"}
	}
	return g.Resource{Name: "Calculator", Filepath: filepath.Join(systemRoot, "System32", "calc.exe")}
}

func cleanExecutablePath(path string) string {
	path = strings.TrimSpace(os.ExpandEnv(path))
	if strings.HasPrefix(path, `"`) {
		if end := strings.Index(path[1:], `"`); end >= 0 {
			path = path[1 : end+1]
		}
	} else if i := strings.Index(path, ","); i != -1 {
		path = path[:i]
	}
	return strings.ToLower(strings.Trim(strings.TrimSpace(path), `"`))
}

func isLaunchableApplication(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".exe") || isAppsFolderPath(path)
}

func isAppsFolderPath(path string) bool {
	return strings.HasPrefix(strings.ToLower(path), strings.ToLower(appsFolderPrefix)) && len(path) > len(appsFolderPrefix)
}

func isAllowedApplication(app g.Resource, skipIfSubstr, blocklist []string) bool {
	path := strings.ToLower(app.Filepath)
	return isLaunchableApplication(path) && !utils.ContainsAny(path, skipIfSubstr) &&
		!utils.ContainsAny(strings.ToLower(app.Name), skipIfSubstr) && !containsAnyFold(path, blocklist)
}

func containsAnyFold(value string, values []string) bool {
	for _, item := range values {
		if strings.Contains(value, strings.ToLower(item)) {
			return true
		}
	}
	return false
}

func resolveShortcut(path string) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

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

func scanAppPaths(currentAppList []g.Resource) []g.Resource {
	keys := []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER}
	basePaths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths`,
		`SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\App Paths`,
	}

	for _, keyRoot := range keys {
		for _, basePath := range basePaths {
			key, err := registry.OpenKey(keyRoot, basePath, registry.READ)
			if err != nil {
				continue
			}
			names, err := key.ReadSubKeyNames(-1)
			_ = key.Close()
			if err != nil {
				continue
			}
			for _, name := range names {
				entry, err := registry.OpenKey(keyRoot, basePath+`\`+name, registry.READ)
				if err != nil {
					continue
				}
				path, _, err := entry.GetStringValue("")
				_ = entry.Close()
				if err != nil {
					continue
				}
				path = cleanExecutablePath(path)
				if path == "" || !strings.HasSuffix(path, ".exe") || hasApplication(currentAppList, name, path) {
					continue
				}
				currentAppList = append(currentAppList, g.Resource{Name: strings.TrimSuffix(name, filepath.Ext(name)), Filepath: path})
			}
		}
	}
	return currentAppList
}

func scanAppsFolder(currentAppList []g.Resource) []g.Resource {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ole.CoInitialize(0); err != nil {
		return currentAppList
	}
	defer ole.CoUninitialize()

	shellObject, err := oleutil.CreateObject("Shell.Application")
	if err != nil {
		return currentAppList
	}
	defer shellObject.Release()

	shell, err := shellObject.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return currentAppList
	}
	defer shell.Release()

	namespaceRaw, err := oleutil.CallMethod(shell, "NameSpace", "shell:AppsFolder")
	if err != nil {
		return currentAppList
	}
	namespace := namespaceRaw.ToIDispatch()
	if namespace == nil {
		return currentAppList
	}
	defer namespace.Release()

	itemsRaw, err := oleutil.GetProperty(namespace, "Items")
	if err != nil {
		return currentAppList
	}
	items := itemsRaw.ToIDispatch()
	if items == nil {
		return currentAppList
	}
	defer items.Release()

	countRaw, err := oleutil.GetProperty(items, "Count")
	if err != nil {
		return currentAppList
	}
	count, ok := oleInteger(countRaw)
	_ = countRaw.Clear()
	if !ok {
		return currentAppList
	}

	for index := 0; index < count; index++ {
		itemRaw, err := oleutil.CallMethod(items, "Item", index)
		if err != nil {
			continue
		}
		item := itemRaw.ToIDispatch()
		if item == nil {
			continue
		}
		name := oleStringProperty(item, "Name")
		appID := oleStringProperty(item, "Path")
		item.Release()

		path := appsFolderPrefix + appID
		if name == "" || appID == "" || hasApplication(currentAppList, name, path) {
			continue
		}
		currentAppList = append(currentAppList, g.Resource{Name: name, Filepath: path})
	}
	return currentAppList
}

func oleInteger(value *ole.VARIANT) (int, bool) {
	switch number := value.Value().(type) {
	case int:
		return number, true
	case int32:
		return int(number), true
	case uint32:
		return int(number), true
	case int64:
		return int(number), true
	case uint64:
		return int(number), true
	default:
		return 0, false
	}
}

func oleStringProperty(item *ole.IDispatch, name string) string {
	value, err := oleutil.GetProperty(item, name)
	if err != nil {
		return ""
	}
	defer value.Clear()
	return strings.TrimSpace(value.ToString())
}

// Search for programs by grabbing .lnk's off the start menu
func scanStartMenu(currentAppList []g.Resource) []g.Resource {
	for _, base := range startMenuDirectories() {
		err := filepath.WalkDir(base, func(p string, de fs.DirEntry, err error) error {
			if err != nil || de.IsDir() || !strings.HasSuffix(strings.ToLower(p), ".lnk") {
				return nil
			}
			target, err := resolveShortcut(p)
			if err != nil || target == "" {
				return nil
			}
			target = cleanExecutablePath(target)
			if !strings.HasSuffix(target, ".exe") {
				return nil
			}
			name := strings.TrimSuffix(de.Name(), ".lnk")

			if hasApplication(currentAppList, name, target) {
				return nil
			}

			currentAppList = append(currentAppList, g.Resource{Name: strings.TrimSpace(name), Filepath: target})
			return nil
		})
		if err != nil {
			continue
		}
	}
	return currentAppList
}

func startMenuDirectories() []string {
	return []string{
		filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs"),
		filepath.Join(os.Getenv("PROGRAMDATA"), "Microsoft", "Windows", "Start Menu", "Programs"),
	}
}

func hasApplication(apps []g.Resource, name, path string) bool {
	for _, app := range apps {
		if strings.EqualFold(app.Filepath, path) || strings.EqualFold(app.Name, strings.TrimSuffix(name, filepath.Ext(name))) {
			return true
		}
	}
	return false
}
