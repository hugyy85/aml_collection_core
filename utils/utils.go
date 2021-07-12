package utils


// Contains указывает, содержится ли x в a.
func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func GetDefaultString(fieldVal string, defaultVal string) string {
	if fieldVal == "" {
		return defaultVal
	} else {
		return fieldVal
	}
}
