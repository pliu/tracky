package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var sampleNotes = []string{
	"Had a productive morning meeting",
	"Finished the quarterly report",
	"Reviewed pull requests",
	"Fixed a critical bug in production",
	"Standup notes: discussed blockers",
	"Lunch with the team",
	"Brainstormed new feature ideas",
	"Updated documentation",
	"Deployed new version to staging",
	"Code review session",
	"Worked on performance optimization",
	"Customer feedback review",
	"Sprint planning completed",
	"Refactored authentication module",
	"Database migration successful",
	"Added unit tests for new feature",
	"Attended product demo",
	"Fixed UI alignment issues",
	"Researched new technologies",
	"Weekly sync with stakeholders",
}

func main() {
	db, err := sql.Open("sqlite3", "./tracky.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Get default notebook for user 1
	var notebookID int
	err = db.QueryRow("SELECT id FROM notebooks WHERE user_id = 1 AND name = 'Default'").Scan(&notebookID)
	if err != nil {
		log.Fatalf("Could not find default notebook for user 1: %v", err)
	}

	fmt.Printf("Found default notebook ID: %d\n", notebookID)

	// Insert notes for the past year
	now := time.Now()
	oneYearAgo := now.AddDate(-1, 0, 0)
	inserted := 0

	for day := oneYearAgo; day.Before(now); day = day.AddDate(0, 0, 1) {
		// Random number of notes per day (0-3)
		numNotes := rand.Intn(4)
		for i := 0; i < numNotes; i++ {
			// Random time during the day
			hour := rand.Intn(14) + 8 // 8 AM to 10 PM
			minute := rand.Intn(60)
			noteTime := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())

			// Random note content
			content := sampleNotes[rand.Intn(len(sampleNotes))]

			_, err := db.Exec(
				"INSERT INTO notes (user_id, notebook_id, content, created_at) VALUES (?, ?, ?, ?)",
				1, notebookID, content, noteTime,
			)
			if err != nil {
				log.Printf("Error inserting note: %v", err)
				continue
			}
			inserted++
		}
	}

	fmt.Printf("Inserted %d notes for user 1 over the past year\n", inserted)
}
