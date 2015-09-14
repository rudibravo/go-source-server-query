package sourceserver

func DecodeString(b []byte, currIndex int) (int, string) {
	var value []byte
	for ; b[currIndex] != 0; currIndex++ {
		value = append(value, b[currIndex])
	}
	currIndex++
	return currIndex, string(value)
}

func DecodeInt32(b []byte, currIndex int) (int, int32) {
	return currIndex+4, int32(b[currIndex]) << 24 | int32(b[currIndex + 1]) << 16 | int32(b[currIndex + 2]) << 8 | int32(b[currIndex + 3])
}

func DecodeInt32LittleEndian(b []byte, currIndex int) (int, int32) {
	return currIndex+4, int32(b[currIndex]) | int32(b[currIndex + 1]) << 8 | int32(b[currIndex + 2]) << 16 | int32(b[currIndex + 3]) << 24
}