package historyattribution

type historyFetchModeContext struct{}
type fetchConfig struct {
	threadPage, turnPage, itemPage int
	itemsView                      string
}

func historyFetchConfig(mode string) (fetchConfig, error) {
	switch mode {
	case "paged20":
		return fetchConfig{20, 20, 20, "notLoaded"}, nil
	case "paged500":
		return fetchConfig{500, 500, 500, "notLoaded"}, nil
	case "pipeline100x1", "pipeline100x4", "parallel100", "parallel100x4", "parallel100x8", "paged100":
		return fetchConfig{100, 100, 100, "notLoaded"}, nil
	case "full5":
		return fetchConfig{100, 5, 0, "full"}, nil
	default:
		return fetchConfig{}, failure("arguments", "invalid_history_fetch_mode")
	}
}

func parallelWorkers(mode string) int {
	switch mode {
	case "pipeline100x1":
		return 1
	case "parallel100":
		return 2
	case "pipeline100x4", "parallel100x4":
		return 4
	case "parallel100x8":
		return 8
	}
	return 0
}
