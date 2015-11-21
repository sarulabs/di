package di

// stringSliceContains checks if a slice of string contains a given element
func stringSliceContains(arr []string, s string) bool {
	for _, elt := range arr {
		if s == elt {
			return true
		}
	}
	return false
}
