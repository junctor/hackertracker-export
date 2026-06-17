package hackertracker

func CollectionNames() []string {
	out := make([]string, len(Collections))
	copy(out, Collections)
	return out
}
