package updatecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultEndpoint = "https://api.github.com/repos/neural-int/commiter-cli/releases/latest"
	cacheFileName   = "update-check.json"
	checkInterval   = 24 * time.Hour
)

var endpoint = defaultEndpoint
var httpClient = &http.Client{Timeout: 3 * time.Second}
var now = time.Now

type cache struct {
	CheckedAt     time.Time `json:"checked_at"`
	LatestVersion string    `json:"latest_version"`
}

type semanticVersion struct {
	parts      [3]int
	prerelease []string
}

type release struct {
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

// Check returns the latest stable release when it is newer than current.
// Errors are intentionally returned to the caller for fail-open handling.
func Check(ctx context.Context, stateDir, current string) (string, error) {
	if _, err := parseVersion(current); err != nil {
		return "", nil
	}
	path := filepath.Join(stateDir, cacheFileName)
	var saved cache
	if data, err := os.ReadFile(path); err == nil {
		if json.Unmarshal(data, &saved) == nil && now().Sub(saved.CheckedAt) < checkInterval && saved.LatestVersion != "" {
			_ = os.Chmod(path, 0o600)
			return newer(current, saved.LatestVersion), nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "commiter-update-check")
	response, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("release endpoint returned HTTP %d", response.StatusCode)
	}
	var result release
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&result); err != nil {
		return "", err
	}
	if result.Draft || result.Prerelease {
		return "", fmt.Errorf("release is not stable")
	}
	if _, err := parseVersion(result.TagName); err != nil {
		return "", err
	}
	saved = cache{CheckedAt: now().UTC(), LatestVersion: result.TagName}
	data, err := json.Marshal(saved)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return "", err
	}
	return newer(current, result.TagName), nil
}

func newer(current, latest string) string {
	c, _ := parseVersion(current)
	l, _ := parseVersion(latest)
	if compare(l, c) > 0 {
		return latest
	}
	return ""
}

var versionPattern = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

func parseVersion(value string) (semanticVersion, error) {
	var result semanticVersion
	if !versionPattern.MatchString(value) {
		return result, fmt.Errorf("invalid semantic version")
	}
	parts := strings.SplitN(strings.TrimPrefix(value, "v"), ".", 3)
	for i := range result.parts {
		part := parts[i]
		if dash := strings.IndexByte(part, '-'); dash >= 0 {
			part = part[:dash]
		}
		if plus := strings.IndexByte(part, '+'); plus >= 0 {
			part = part[:plus]
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return result, err
		}
		result.parts[i] = n
	}
	if dash := strings.IndexByte(parts[2], '-'); dash >= 0 {
		prerelease := parts[2][dash+1:]
		if plus := strings.IndexByte(prerelease, '+'); plus >= 0 {
			prerelease = prerelease[:plus]
		}
		result.prerelease = strings.Split(prerelease, ".")
	}
	return result, nil
}

func compare(left, right semanticVersion) int {
	for i := range left.parts {
		if left.parts[i] != right.parts[i] {
			if left.parts[i] > right.parts[i] {
				return 1
			}
			return -1
		}
	}
	if len(left.prerelease) == 0 && len(right.prerelease) == 0 {
		return 0
	}
	if len(left.prerelease) == 0 {
		return 1
	}
	if len(right.prerelease) == 0 {
		return -1
	}
	for i := 0; i < len(left.prerelease) && i < len(right.prerelease); i++ {
		l, r := left.prerelease[i], right.prerelease[i]
		ln, lerr := strconv.Atoi(l)
		rn, rerr := strconv.Atoi(r)
		if lerr == nil && rerr == nil && ln != rn {
			if ln > rn {
				return 1
			}
			return -1
		}
		if lerr == nil && rerr != nil {
			return -1
		}
		if lerr != nil && rerr == nil {
			return 1
		}
		if l != r {
			if l > r {
				return 1
			}
			return -1
		}
	}
	if len(left.prerelease) > len(right.prerelease) {
		return 1
	}
	if len(left.prerelease) < len(right.prerelease) {
		return -1
	}
	return 0
}
