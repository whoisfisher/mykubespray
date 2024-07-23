package utils

import (
	"bytes"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io"
	"log"
	"strings"
)

type LogEntry struct {
	Message string // 日志消息
	IsError bool   // 是否为错误日志
}

func DecodeBytes(data []byte, decoder *encoding.Decoder) (string, error) {
	reader := transform.NewReader(bytes.NewReader(data), decoder)
	decodedData, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(decodedData), nil
}

func DecodeGBK(data []byte) (string, error) {
	return DecodeBytes(data, simplifiedchinese.GBK.NewDecoder())
}

func GetDistribution(executor *SSHExecutor) (string, error) {
	output, err := executor.ExecuteShortCommand("cat /etc/os-release")
	if err != nil {
		log.Printf("Get distribution failed: %s", err.Error())
		return "", err
	}
	res := parseOSRelease(output)
	return res, nil
}

func parseOSRelease(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ID=") {
			return strings.TrimPrefix(line, "ID=")
		}
	}
	return "Unknown"
}
