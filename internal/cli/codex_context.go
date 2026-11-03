package cli

import (
	"crypto/sha256"
	"github.com/natsuki0413/commiter-cli/internal/historyattribution"
	"regexp"
	"time"
)

type codexCycleKey struct{}
type codexCycle struct {
	started              time.Time
	attributionCollected bool
	attributionKey       [sha256.Size]byte
	attribution          historyattribution.Result
}

var inlineLogSecret = regexp.MustCompile(`(?i)(?:api[_-]?key|access[_-]?token|password|passwd|client[_-]?secret|authorization)\s*[=:]\s*\S+`)
