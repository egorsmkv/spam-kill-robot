package main

import (
	"flag"
	"os"

	"github.com/egorsmkv/spam-kill-robot"
)

var (
	db, token string
	votes     int
	debug     bool
)

func init() {
	flag.StringVar(&db, "d", "", "path to the database")
	flag.StringVar(&token, "t", "", "the bot token")
	flag.IntVar(&votes, "v", 3, "number of votes for delete a spam message")
	flag.BoolVar(&debug, "dg", false, "enable debug mode")
}

func main() {
	flag.Parse()

	if db == "" || token == "" || votes < 0 {
		flag.Usage()
		os.Exit(1)
	}

	bot.Run(db, token, votes, debug)
}
