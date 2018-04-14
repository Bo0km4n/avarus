package crawler

func isStartWithHTTP(link string) bool {
	if len(httpToken) <= len(link) {
		return string([]rune(link)[:len(httpToken)]) == httpToken
	}
	return false
}

func isStartWithHTTPS(link string) bool {
	if len(httpsToken) <= len(link) {
		return string([]rune(link)[:len(httpsToken)]) == httpsToken
	}
	return false
}

func isStartWithRelative(link string) bool {
	if len(relativeToken) <= len(link) {
		return string([]rune(link)[:len(relativeToken)]) == relativeToken
	}
	return false
}

func isStartWithCurrentPath(link string) bool {
	if len(currentPathToken) <= len(link) {
		return string([]rune(link)[:len(currentPathToken)]) == currentPathToken
	}
	return false
}

func isStartWithDoubleSlash(link string) bool {
	if len(doubleSlashToken) <= len(link) {
		return string([]rune(link)[:len(doubleSlashToken)]) == doubleSlashToken
	}
	return false
}
