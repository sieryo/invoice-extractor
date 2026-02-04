package logger

import "log"

type Category string

const (
	Master  Category = "MASTER"
	Route   Category = "ROUTE"
	Server  Category = "SERVER"
	Service Category = "SERVICE"
)

const (
	green  = "\033[32m"
	blue   = "\033[34m"
	yellow = "\033[33m"
	red    = "\033[31m"
	reset  = "\033[0m"
)

func logPrint(color string, cat Category, msg string) {
	log.Printf("%s[%s]%s %s", color, cat, reset, msg)
}

func Info(cat Category, msg string) {
	logPrint(blue, cat, msg)
}

func Warn(cat Category, msg string) {
	logPrint(yellow, cat, msg)
}

func Error(cat Category, msg string) {
	logPrint(red, cat, msg)
}

func Success(cat Category, msg string) {
	logPrint(green, cat, msg)
}

func ServerStarted(url string, env string) {
	Success(Server, "Server running at "+yellow+url+reset)
	Info(Server, "Environment: "+env)
}
