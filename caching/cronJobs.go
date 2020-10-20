package smached

import (
	"log"
	"time"
)

func initCronJobs() {

	duration, _ := time.ParseDuration(config.MaxTtl)
	ticker := time.NewTicker(duration)
	for _ = range ticker.C {
		log.Println("Clearing cache by expiration.")
		EvictByExpirationTime()
	}
}
