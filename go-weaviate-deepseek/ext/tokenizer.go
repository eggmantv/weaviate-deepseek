package ext

import "github.com/tiktoken-go/tokenizer"

var TokenCodec tokenizer.Codec

func init() {
	var err error
	TokenCodec, err = tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		panic(err)
	}
}

func TokenLen(s string) int {
	toS, _, _ := TokenCodec.Encode(s)
	return len(toS)
}
