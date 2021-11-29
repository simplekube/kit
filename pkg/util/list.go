package util

// RemoveFromListIfValid removes the provided entries if they are not
// nil. This logic results in removal of duplicates & empty entries
// if any
func RemoveFromListIfValid(given []string, remove string, more ...string) []string {
	var result = make([]string, len(given))
	var removals = make(map[string]struct{}, len(more)+1)

	if remove != "" {
		removals[remove] = struct{}{}
	}

	for _, r := range more {
		if _, found := removals[r]; !found && r != "" {
			removals[r] = struct{}{}
		}
	}
	// given order is preserved
	for _, g := range given {
		if _, found := removals[g]; !found && g != "" {
			removals[g] = struct{}{}
			result = append(result, g)
		}
	}
	return result
}

// AddToListIfValid adds the provided entries if they are not nil
// This logic results in removal of duplicates & empty entries if any
func AddToListIfValid(given []string, add string, more ...string) []string {
	var result = make([]string, len(given)+len(more)+1)
	var present = make(map[string]struct{}, len(given)+len(more)+1)

	// given order is preserved
	for _, g := range given {
		if _, found := present[g]; !found && g != "" {
			present[g] = struct{}{}
			result = append(result, g)
		}
	}
	if _, found := present[add]; !found && add != "" {
		present[add] = struct{}{}
		result = append(result, add)
	}
	for _, m := range more {
		if _, found := present[m]; !found && m != "" {
			present[m] = struct{}{}
			result = append(result, m)
		}
	}
	return result
}
