package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"crypto/sha256"
	"io"
	_ "github.com/mattn/go-sqlite3"
)

var filesStatement *sql.Stmt
var locationsStatement *sql.Stmt

func hashDirFilesRecursive(dirPath string) {
	// iterate over entries in dir path
	// hash all of the files that we find there
	// save them to the database

	// Start collecting 
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			hashDirFilesRecursive(filepath.Join(dirPath, entry.Name()))	
		} else {
			fullFilePath := filepath.Join(dirPath, entry.Name())
			
			// Open (read) the file
			file, err := os.Open(fullFilePath)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()

			// Hash the contents of the file
			hasher := sha256.New()
			if _, err := io.Copy(hasher, file); err != nil {
				log.Fatal(err)
			}

			hashString := fmt.Sprintf("%x", hasher.Sum(nil))

			// Add Hash to DB
			_, err = filesStatement.Exec(hashString)
			if err != nil {
				log.Fatal(err)
			}

			// Add Filepath to DB
			_, err = locationsStatement.Exec(hashString, fullFilePath)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func main() {
	// Remove pre-existing db file
	os.Remove("./imagesorter.db")

	// Take the directory from the user as an argument
	sourceDir := os.Args[1]

	// Connect to sqlite db
	db, err := sql.Open("sqlite3", "./imagesorter.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create DB Tables
	createSQL := `
	CREATE TABLE IF NOT EXISTS files (
		hash text PRIMARY KEY
	);
	CREATE TABLE IF NOT EXISTS locations (
		hash text, 
		filepath text,
		PRIMARY KEY (hash, filepath),
		FOREIGN KEY (hash) REFERENCES files(hash)
	);
	`
	_, err = db.Exec(createSQL)
	if err != nil {
		log.Printf("%q: %s\n", err, createSQL)
		return
	}

	// Start the DB transaction
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	
	// Prepare Files Table Insert Statement
	filesStatement, err = tx.Prepare("INSERT OR IGNORE INTO files(hash) VALUES(?)")
	if err != nil {
		log.Fatal(err)
	}
	defer filesStatement.Close()


	locationsStatement, err = tx.Prepare("INSERT OR IGNORE INTO locations(hash, filepath) VALUES(?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer locationsStatement.Close()

	// Recursively Hash Files and add to DB
	hashDirFilesRecursive(sourceDir)

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	// Handle duplicates
	handleDuplicates(db)

}

func handleDuplicates(db *sql.DB) {
	hashQuery := `
	SELECT hash
	FROM locations
	GROUP BY hash
	HAVING COUNT(*) > 1;
	`

	filepathsQuery := `
	SELECT filepath
	FROM locations
	WHERE hash = ?;
	`

	rows, err := db.Query(hashQuery)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var hash string
		err = rows.Scan(&hash)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(hash)

		// Get all of the filepaths
		pathRows, err := db.Query(filepathsQuery, hash)
		if err != nil {
			log.Fatal(err)
		}
		defer pathRows.Close()

		var filepaths []string
		for pathRows.Next() {
			var filepath string
			err = pathRows.Scan(&filepath)
			if err != nil {
				log.Fatal(err)
			}
			filepaths = append(filepaths, filepath)
		}

		// Iterate over the filepaths
		// Print out an index that the user can reference
		// 1. c:\photo\myphoto.jpg

		fmt.Println("Duplicate files found:")
		for i, fileOption := range filepaths {
			fmt.Println(i+1, fileOption)
		}
		
		// Get input from user as to which one to keep
		fmt.Println("Please select which file to keep (0 to skip):")
		var selected int
		_, err = fmt.Scan(&selected)

		// Handle validation of input
		// eg. check that the index is correct (h.m. items in slice?)
		if err != nil || selected < 0 || selected > len(filepaths) {
			fmt.Println("Invalid input... skipping")
			continue
		}

		if selected == 0 {
			fmt.Println("0 selected... skipping")
			continue
		}

		// Delete the others
		for i, fileOption := range filepaths {
			if i+1 == selected {
				fmt.Printf("Selected file %s kept.\n", fileOption)
			} else {
				err = os.Remove(fileOption)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("File %s deleted.\n", fileOption)
			}
		}	
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

}
