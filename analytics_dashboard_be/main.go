package main

import (
	"log"

	"analytics-dashboard-be/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
