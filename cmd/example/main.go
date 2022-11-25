package main

import (
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

func main() {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse("5 4 * * MON")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("next time:", schedule.Next(time.Now()).String())
}
