package ext

import (
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/bwmarrin/snowflake"
	"github.com/gofrs/uuid"
)

type M map[string]interface{}

var GlobalID *snowflake.Node

func init() {
	var err error
	GlobalID, err = snowflake.NewNode(1)
	if err != nil {
		panic(err)
	}
}

func MergeM(m1 M, m2 M) M {
	merged := make(M)
	for k, v := range m1 {
		merged[k] = v
	}
	for key, value := range m2 {
		merged[key] = value
	}
	return merged
}

func ToA(b []byte) []string {
	a := make([]string, 0)
	_ = json.Unmarshal(b, &a)
	return a
}

func ToB(obj interface{}) []byte {
	b, _ := json.Marshal(obj)
	return b
}

func ToM(b []byte) M {
	m := M{}
	_ = json.Unmarshal(b, &m)
	return m
}

func ToMA(b []byte) []M {
	m := make([]M, 0)
	_ = json.Unmarshal(b, &m)
	return m
}

var reNewlinesChar = regexp.MustCompile(`\n+`)

func Oneline(str string) string {
	r := reNewlinesChar.ReplaceAllString(str, "\n")
	return strings.ReplaceAll(r, "\n", "")
}

func GenUUID() string {
	u, _ := uuid.NewV4()
	return strings.Replace(u.String(), "-", "", -1)
}

func GenGlobalID() string {
	return GlobalID.Generate().String()
}

func ToBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func FromBase64(s string) string {
	res, _ := base64.StdEncoding.DecodeString(s)
	return string(res)
}
