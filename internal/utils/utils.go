package utils

import (
	"bytes"
	"io"
	"strings"

	"gopkg.in/yaml.v2"
)

func StreamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}

func StreamToString(stream io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.String()
}

func SteamCopyToByte(stream io.Reader) []byte {
	buf := new(strings.Builder)
	io.Copy(buf, stream)
	return []byte(buf.String())
}

func ConvertToYAML(v interface{}) (string, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
