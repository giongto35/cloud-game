package resampler

func Linear(dst, src []int16) {
	nSrc, nDst := len(src), len(dst)
	if nSrc < 2 || nDst < 2 {
		return
	}

	srcPairs, dstPairs := nSrc>>1, nDst>>1

	// replicate single pair input or output
	if srcPairs == 1 || dstPairs == 1 {
		for i := 0; i < dstPairs; i++ {
			dst[i*2], dst[i*2+1] = src[0], src[1]
		}
		return
	}

	ratio := ((srcPairs - 1) << 16) / (dstPairs - 1)
	lastSrc := nSrc - 2

	// interpolate all pairs except the last
	for i, pos := 0, 0; i < dstPairs-1; i, pos = i+1, pos+ratio {
		idx := (pos >> 16) << 1
		di := i << 1
		frac := int32(pos & 0xFFFF)
		l0, r0 := int32(src[idx]), int32(src[idx+1])

		// L = L0 + (L1-L0)*frac
		dst[di] = int16(l0 + ((int32(src[idx+2])-l0)*frac)>>16)
		// R = R0 + (R1-R0)*frac
		dst[di+1] = int16(r0 + ((int32(src[idx+3])-r0)*frac)>>16)
	}

	// last output pair = last input pair (avoids precision loss at the edge)
	lastDst := (dstPairs - 1) << 1
	dst[lastDst], dst[lastDst+1] = src[lastSrc], src[lastSrc+1]
}

func Nearest(dst, src []int16) {
	nSrc, nDst := len(src), len(dst)
	if nSrc < 2 || nDst < 2 {
		return
	}

	srcPairs, dstPairs := nSrc>>1, nDst>>1

	if srcPairs == 1 || dstPairs == 1 {
		for i := 0; i < dstPairs; i++ {
			dst[i*2], dst[i*2+1] = src[0], src[1]
		}
		return
	}

	ratio := (srcPairs << 16) / dstPairs

	for i, pos := 0, 0; i < dstPairs; i, pos = i+1, pos+ratio {
		si := (pos >> 16) << 1
		di := i << 1
		dst[di], dst[di+1] = src[si], src[si+1]
	}
}
