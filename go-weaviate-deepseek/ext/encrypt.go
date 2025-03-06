package ext

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
)

var EncryptSecret = []byte("um4vQSyIw49nnr3rKxXpSHbanTgUi5gK")

func Encrypt(str string) ([]byte, error) {
	iv := make([]byte, aes.BlockSize)
	bPlaintext := pkcs5Padding([]byte(str), aes.BlockSize, len(str))
	block, _ := aes.NewCipher(EncryptSecret)
	ciphertext := make([]byte, len(bPlaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, bPlaintext)
	// hex.EncodeToString(ciphertext)
	res := base64.StdEncoding.EncodeToString(ciphertext)
	return []byte(res), nil
}

func Decrypt(str string) ([]byte, error) {
	encryptedString, _ := base64.StdEncoding.DecodeString(str)

	// Decrypt the string
	block, err := aes.NewCipher(EncryptSecret)
	if err != nil {
		return nil, err
	}
	iv := make([]byte, aes.BlockSize)
	stream := cipher.NewCBCDecrypter(block, iv)
	decryptedString := make([]byte, len(encryptedString))
	stream.CryptBlocks(decryptedString, encryptedString)
	decryptedString = pkcs5UnPadding(decryptedString)
	return decryptedString, nil
}

func pkcs5Padding(ciphertext []byte, blockSize int, after int) []byte {
	padding := (blockSize - len(ciphertext)%blockSize)
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func pkcs5UnPadding(data []byte) []byte {
	length := len(data)
	unpadding := int(data[length-1])
	return data[:(length - unpadding)]
}
