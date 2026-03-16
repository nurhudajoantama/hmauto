package hmstt

func canTypeChangedWithKey(tipe, key, value string) bool {
	if tipe == "" || key == "" {
		return false
	}

	if tipe == PREFIX_SWITCH {
		return value == STATE_ON || value == STATE_OFF
	}

	return false
}
