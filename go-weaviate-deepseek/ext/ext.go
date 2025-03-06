package ext

import (
	"regexp"
)

var cnRe = regexp.MustCompile(`\p{Han}`)

func HasChinese(text string) bool {
	return cnRe.MatchString(text)
}
