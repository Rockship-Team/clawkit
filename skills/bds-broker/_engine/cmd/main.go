package main

import (
	"fmt"
	"os"
)

const usage = `bds-cli — BDS Broker CLI for OpenClaw agents

Commands:
  init                                   Init DB and du-an/ directories

  listing list [--type TYPE] [--location LOC] [--min N] [--max N] [--limit N] [--giao-dich ban|cho_thue]
  listing get    LISTING_PATH
  listing images LISTING_PATH [SUBFOLDER]
  listing next-id LOAI_FOLDER
  listing create  LOAI_FOLDER ID TITLE LOAI_VI LOCATION ADDRESS AREA PRICE BEDROOMS DIRECTION LEGAL DESC
  listing set-field LISTING_PATH FIELD VALUE

  appt book    CUSTOMER CONTACT LISTING_PATH TITLE DATETIME [NOTE]
  appt list    [KEYWORD]
  appt update  ID STATUS

All output is JSON. BDS_DATA env var overrides the data directory.`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "init":
		db := openDB()
		initDB(db)
		db.Close()
		initDirs()
		jsonOut(map[string]interface{}{"status": "ok", "db": dbPath(), "du_an": duAnDir()})

	case "listing":
		if len(args) == 0 {
			die("subcommand required: list|get|images|next-id|create|set-field")
		}
		sub, rest := args[0], args[1:]
		switch sub {
		case "list":
			cmdListingList(rest)
		case "get":
			cmdListingGet(rest)
		case "images":
			cmdListingImages(rest)
		case "next-id":
			cmdListingNextID(rest)
		case "create":
			cmdListingCreate(rest)
		case "set-field":
			cmdListingSetField(rest)
		default:
			die("unknown listing subcommand: %s", sub)
		}

	case "appt":
		if len(args) == 0 {
			die("subcommand required: book|list|update")
		}
		sub, rest := args[0], args[1:]
		switch sub {
		case "book":
			cmdApptBook(rest)
		case "list":
			cmdApptList(rest)
		case "update":
			cmdApptUpdate(rest)
		default:
			die("unknown appt subcommand: %s", sub)
		}

	default:
		die("unknown command: %s\n%s", cmd, usage)
	}
}
