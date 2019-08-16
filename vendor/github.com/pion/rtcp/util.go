package rtcp

// getPadding Returns the padding required to make the length a multiple of 4
func getPadding(len int) int {
	if len%4 == 0 {
		return 0
	}
	return 4 - (len % 4)
}
