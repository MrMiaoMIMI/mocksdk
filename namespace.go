package mocksdk

import "strings"

func NormalizeNamespace(namespace string) string {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return DefaultNamespace
	}
	return namespace
}

func normalizedNamespace(namespace string) string {
	return NormalizeNamespace(namespace)
}
