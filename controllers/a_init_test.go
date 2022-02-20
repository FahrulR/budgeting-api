package controllers

import (
	"bytes"
	"encoding/json"
	"log"
)

func init() {
	log.SetFlags(log.LstdFlags | log.LUTC | log.Lshortfile)
}

func parsePayload(p interface{}) *bytes.Buffer {
	data, _ := json.Marshal(p)
	return bytes.NewBuffer(data)
}
