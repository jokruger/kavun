package conv

func ParseBool(s string) (bool, bool) {
	switch s {
	case "t", "T", "true", "True", "TRUE", "y", "Y", "yes", "Yes", "YES", "1":
		return true, true
	case "", "f", "F", "false", "False", "FALSE", "n", "N", "no", "No", "NO", "0":
		return false, true
	default:
		return false, false
	}
}
