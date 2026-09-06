package apps

import (
	"strings"
	"sync"
)

var (
	aliasMu sync.RWMutex
	aliases map[string]string
)

func SetAliases(value string) {
	parsed := make(map[string]string)
	for entry := range strings.FieldsFuncSeq(value, func(r rune) bool { return r == ';' || r == '\n' }) {
		alias, target, ok := strings.Cut(entry, "=")
		alias = normalizeAlias(alias)
		target = strings.TrimSpace(target)
		if ok && alias != "" && target != "" {
			parsed[alias] = target
		}
	}
	aliasMu.Lock()
	aliases = parsed
	aliasMu.Unlock()
}

func aliasTarget(query string) string {
	aliasMu.RLock()
	target := aliases[normalizeAlias(query)]
	aliasMu.RUnlock()
	return target
}

func normalizeAlias(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}
