package planning

import "strings"

// CommandSafetyText prevents common, approved source-file suffixes from being
// mistaken for JWTs. It is used only for scanning, never as the transmitted text.
// Other path components and all non-path text remain subject to normal checks.
func CommandSafetyText(command string, approvedPaths []string) string {
	for _, path := range approvedPaths {
		sourcePath := false
		for _, ext := range []string{".go", ".js", ".jsx", ".ts", ".tsx", ".css", ".scss", ".py", ".rs"} {
			if strings.HasSuffix(path, ext) {
				sourcePath = true
				break
			}
		}
		if !sourcePath {
			continue
		}
		normalized := path
		for _, marker := range []string{".test.", ".spec.", ".module.", ".min.", ".d.ts"} {
			normalized = strings.ReplaceAll(normalized, marker, strings.ReplaceAll(marker, ".", "/"))
		}
		if normalized != path {
			command = strings.ReplaceAll(command, path, normalized)
		}
	}
	return command
}
