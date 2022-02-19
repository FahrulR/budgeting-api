package controllers

import (
	"log"
)

func init() {
	log.SetFlags(log.LstdFlags | log.LUTC | log.Lshortfile)
}
