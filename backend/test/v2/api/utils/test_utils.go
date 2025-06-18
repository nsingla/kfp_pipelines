package test

func ParsePointersToString(s *string) string {
	if s == nil {
		return ""
	} else {
		return *s
	}
}
