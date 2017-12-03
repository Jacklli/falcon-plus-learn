package g

import log "github.com/Sirupsen/logrus" // Logrus is a structured logger for Go (golang), completely API compatible with the standard library logger.

func InitLog(level string) (err error) {
	switch level {
	case "info":
		log.SetLevel(log.InfoLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	default:
		log.Fatal("log conf only allow [info, debug, warn], please check your confguire")
	}
	return
}
