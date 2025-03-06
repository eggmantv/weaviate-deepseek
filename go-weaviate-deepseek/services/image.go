package services

import (
	"bytes"
	"encoding/base64"
	"go-weaviate-deepseek/ext"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// ExtractTextFromImage imgBase64OrPath: image base64 or image path
func ExtractTextFromImage(imgBase64OrPath string, isBase64 bool) (string, error) {
	f := imgBase64OrPath
	var err error
	if isBase64 {
		f, err = saveImageFrom64(imgBase64OrPath)
		if err != nil {
			return "", err
		}
		defer os.Remove(f)
	}

	targetFile := "/tmp/" + ext.GenGlobalID()
	_, err = exec.Command("tesseract", f, targetFile, "-l", "chi_sim+eng").Output()
	if err != nil {
		return "", nil
	}
	defer os.Remove(targetFile)
	b, err := ioutil.ReadFile(targetFile + ".txt")
	if err != nil {
		return "", nil
	}
	return string(b), nil
}

func saveImageFrom64(b64 string) (string, error) {
	// in case has data prefix
	seg := strings.Split(b64, ",")
	data := b64
	if len(seg) > 1 {
		data = seg[1]
	}
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	r := bytes.NewReader(b)
	im, err := png.Decode(r)
	if err != nil {
		return "", err
	}

	filep := "/tmp/" + ext.GenGlobalID()
	f, err := os.OpenFile(filep, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return "", err
	}
	err = png.Encode(f, im)
	if err != nil {
		return "", err
	}
	f.Close()
	return filep, nil
}

func ReadImageTo64(filep string, withMimeType bool) (string, error) {
	b, err := ioutil.ReadFile(filep)
	if err != nil {
		return "", err
	}
	mimeType := http.DetectContentType(b)
	res := base64.StdEncoding.EncodeToString(b)
	if withMimeType {
		switch mimeType {
		case "image/jpeg":
			res = "data:image/jpeg;base64," + res
		case "image/png":
			res = "data:image/png;base64," + res
		}
	}
	return res, nil
}
