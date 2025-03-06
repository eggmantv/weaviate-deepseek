package services

import (
	"go-weaviate-deepseek/ext"
	"go-weaviate-deepseek/ext/weaviatelib"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

type ChunkAttr struct {
	Chunk       string    `json:"chunk"`
	ChunkTokens int       `json:"chunk_tokens"`
	ChunkLength int       `json:"chunk_length"`
	TextVector  []float32 `json:"text_vector"`
}

const (
	CHUNK_SIZE          = 500
	CHUNK_DELIMITTER_CN = "。"
	CHUNK_DELIMITTER_EN = "."
)

var RE_CHUNK_SPLIT_DELIMITTER = regexp.MustCompile(`\.\s|。`)
var RE_CHUNK_SPLIT_COMMA = regexp.MustCompile(`\,|，`)
var RE_CHUNK_SPACE = regexp.MustCompile(`\s+`)
var RE_CHUNK_NEWLINE = regexp.MustCompile(`(\n\s*)+`)

// var RE_CHUNK_EMPTY = regexp.MustCompile(`^\s*$`)

// func l() *logrus.Entry {
// 	return ext.LF("chunk")
// }

func subChunkSplit(splits []string, chunkSize int, reg *regexp.Regexp, res []string) []string {
	for _, partSplit := range splits {
		partToken, _, _ := ext.TokenCodec.Encode(partSplit)
		if len(partToken) > CHUNK_SIZE {
			s := reg.Split(partSplit, -1)

			for _, innerChunk := range s {
				innerToken, _, _ := ext.TokenCodec.Encode(innerChunk)
				if len(innerToken) > CHUNK_SIZE {
					res = append(res, subChunkSplit([]string{innerChunk}, chunkSize, RE_CHUNK_SPACE, res)...)
				} else {
					res = append(res, innerChunk)
				}
			}
		} else {
			res = append(res, partSplit)
		}
	}

	return res
}

func ChunkSplit(text string, chunkSize int) []*ChunkAttr {
	content := RE_CHUNK_SPACE.ReplaceAllString(text, " ")
	isChinese := ext.HasChinese(content)

	chunks := make([]*ChunkAttr, 0)

	contentTokensLength := ext.TokenLen(content)

	if contentTokensLength > chunkSize {
		split := RE_CHUNK_SPLIT_DELIMITTER.Split(content, -1)
		// 因为whisper有时生成的文字会连续很长没有句号，导致split有问题, 所以如果超过，就用逗号来分割
		newSplit := make([]string, 0)
		newSplit = subChunkSplit(split, chunkSize, RE_CHUNK_SPLIT_COMMA, newSplit)

		chunkText := ""
		for _, ns := range newSplit {
			sentence := strings.TrimSpace(ns)
			sentenceTokensLength := ext.TokenLen(sentence)
			chunkTextTokensLength := ext.TokenLen(chunkText)
			if chunkTextTokensLength+sentenceTokensLength > chunkSize {
				if chunkTextTokensLength > 0 {
					chunks = append(chunks, &ChunkAttr{
						Chunk:       chunkText,
						ChunkTokens: chunkTextTokensLength,
						ChunkLength: len(chunkText),
					})
				}
				chunkText = ""
			}

			if len(sentence) > 0 {
				if strings.HasSuffix(sentence, CHUNK_DELIMITTER_CN) || strings.HasSuffix(sentence, CHUNK_DELIMITTER_EN) {
					chunkText += sentence
				} else {
					if isChinese {
						chunkText += sentence + CHUNK_DELIMITTER_CN
					} else {
						chunkText += sentence + CHUNK_DELIMITTER_EN
					}
				}
			}
		}
		chunkTextTokensLength := ext.TokenLen(chunkText)
		if chunkTextTokensLength > 0 {
			chunks = append(chunks, &ChunkAttr{
				Chunk:       strings.TrimSpace(chunkText),
				ChunkTokens: chunkTextTokensLength,
				ChunkLength: len(chunkText),
			})
		}
	} else {
		if contentTokensLength > 0 {
			chunks = append(chunks, &ChunkAttr{
				Chunk:       strings.TrimSpace(content),
				ChunkTokens: contentTokensLength,
				ChunkLength: len(text),
			})
		}
	}

	return chunks
}

func (ca *ChunkAttr) CalVector() error {
	textVector, err := weaviatelib.VectorizerFunc(ca.Chunk)
	if err != nil {
		return err
	}
	ca.TextVector = textVector
	return nil
}

func (ca *ChunkAttr) Save(clsName string, addiAttrs ext.M) error {
	id := uuid.NewString()
	text := ca.Chunk
	textVector := ca.TextVector
	attrs := ext.MergeM(ext.M{"captions": text}, addiAttrs)
	_, err := weaviatelib.Create(clsName, id, attrs, textVector)
	if err != nil {
		return err
		// l().Println("ToVector save err:", err)
	}
	l().Printf("ToVector text: %s, vector size: %v", text, len(textVector))
	return nil
}
