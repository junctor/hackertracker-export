package export

func Summary(written []string) map[string]any {
	return map[string]any{
		"writtenFiles": len(written),
		"files":        written,
	}
}
