package conf

import "os"

var Env string
var TIKA_HOST string

const (
	AuthHeaderKey    = "X_KEY"
	AuthHeaderSecret = "xxx"
	WebAPIPort       = ":5012"
	WebAPIPrdSelfURL = "https://eggman.tv" + WebAPIPort

	WebAPISecret = "xxx"

	AliDeepSeekAPIKey  = "sk-xxx"
	AliDeepSeeKBaseUrl = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	APITypeAliDeepSeeK = "AliDeepSeeK"

	AliDeepSeekModelName = "deepseek-v3" // "deepseek-r1"
)

func init() {
	TIKA_HOST = os.Getenv("TIKA_HOST")
	if len(TIKA_HOST) == 0 {
		TIKA_HOST = "http://localhost:9998"
	}
}

func Parse(e string) {
	Env = e

	if IsPrd() {
	} else {
	}
}

func IsPrd() bool {
	if os.Getenv("ENV") == "production" {
		return true
	}
	return Env == "production"
}
