package util

func MinInt(x int, y int) int {
	if x < y {
		return x
	}
	return y
}

func ContainsString(sslice []string, s string) bool {
	for _, ss := range sslice {
		if ss == s {
			return true
		}
	}

	return false
}
