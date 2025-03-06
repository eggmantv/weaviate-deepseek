package services

import (
	"go-weaviate-deepseek/ext"

	"github.com/sirupsen/logrus"
)

func l() *logrus.Entry {
	return ext.LF("services")
}
