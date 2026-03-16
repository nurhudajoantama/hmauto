package hmstt

func generateKey(tipe, key string) (string, bool) {
	if tipe == "" || key == "" {
		return "", false
	}

	var ok bool

	if tipe == PREFIX_SWITCH {
		ok = true
	}

	return PREFIX_HMSTT + KEY_DELIMITER + tipe + KEY_DELIMITER + key, ok
}

func canTypeChangedWithKey(tipe, key, value string) (string, bool) {
	generatedKey, ok := generateKey(tipe, key)
	if !ok {
		return "", false
	}

	if tipe == PREFIX_SWITCH {
		if value == STATE_ON || value == STATE_OFF {
			return generatedKey, true
		}
	}

	return "", false
}

