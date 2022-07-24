package util

func Remove(s []string, k string) []string {
	for i, v := range s {
		if v == k {
			s = append(s[:i], s[i+1:]...)
			break
		}
	}

	return s
}

func Intersects(s []string, e ...string) bool {
	for _, a := range e {
		if Contains(s, a) {
			return true
		}
	}

	return false
}

func Subset(s []string, e ...string) bool {
	for _, a := range e {
		if !Contains(s, a) {
			return false
		}
	}

	return true
}

func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}

	return false
}
