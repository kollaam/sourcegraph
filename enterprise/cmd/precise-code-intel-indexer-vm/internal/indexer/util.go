package indexer

import (
	"os"
	"sort"
)

func fileExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// TODO - rename
func prefixKeys(v string, s []string) []string {
	var q []string
	for _, x := range s {
		q = append(q, v, x)
	}

	return q
}

func sortKeys(m map[string]string) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	return keys
}
