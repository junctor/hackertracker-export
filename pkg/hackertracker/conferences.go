package hackertracker

import "slices"

func CollectionNames() []string {
	return slices.Clone(collections[:])
}
