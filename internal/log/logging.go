package log

import "log"

func Debug(s ...interface{}) {
	log.Println("[DEBUG]", s)
}

func Info(s ...interface{}) {
	log.Println("[INFO]", s)
}

func Warn(s ...interface{}) {
	log.Println("[WARN]", s)
}