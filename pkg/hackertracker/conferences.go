package hackertracker

func CollectionNames() []string {
	out := make([]string, len(collections))
	copy(out, collections[:])
	return out
}
