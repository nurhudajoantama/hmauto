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

func snakeToTitle(s string) string {
	result := ""
	capitalizeNext := true

	for _, char := range s {
		if char == '_' {
			result += " "
			capitalizeNext = true
		} else {
			if capitalizeNext {
				if char >= 'a' && char <= 'z' {
					char -= 'a' - 'A'
				}
				capitalizeNext = false
			}
			result += string(char)
		}
	}

	return result
}
