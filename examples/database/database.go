// Copyright (C) 2016 Kale Blankenship. All rights reserved.
// This software may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details

// +build ignore

package main

import (
	"database/sql"
	"io/ioutil"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/vcabbage/trivialt"
)

func main() {
	// Create or open a sqlite database
	db, err := sql.Open("sqlite3", "tftp.db")
	if err != nil {
		log.Fatal(err)
	}

	// Create a simple table to hold the ip and sent log data from
	// the client.
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tftplogs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        ip TEXT,
        log TEXT
    );`)
	if err != nil {
		log.Fatal(err)
	}

	// Create a new server listening on port 6900, all interfaces
	server, err := trivialt.NewServer(":6900")
	if err != nil {
		log.Fatal(err)
	}

	// Set the server's write handler, read requests will be rejeccted
	server.WriteHandler(&tftpDB{db})

	// Start the server, if it fails error will be printed by log.Fatal
	log.Fatal(server.ListenAndServe())
}

// tftpDB embeds a *sql.DB and implements the trivialt.ReadHandler
// interface.
type tftpDB struct {
	*sql.DB
}

func (db *tftpDB) ReceiveTFTP(w trivialt.WriteRequest) {
	// Get the file size
	size, err := w.Size()

	// We're choosing to only store logs that are less than 1MB.
	// An error indicates no size was recieved.
	if err != nil || size > 1024*1024 {
		// Send a "disk full" error.
		w.WriteError(trivialt.ErrCodeDiskFull, "File too large or no size sent")
		return
	}

	// Read the data from the client into memory
	data, err := ioutil.ReadAll(w)
	if err != nil {
		log.Println(err)
		return
	}

	// Insert the IP address of the client and the data into the database
	res, err := db.Exec("INSERT INTO tftplogs (ip, log) VALUES (?, ?)", w.Addr().IP.String(), string(data))
	if err != nil {
		log.Println(err)
		return
	}

	// Log a message with the details
	id, _ := res.LastInsertId()
	log.Printf("Inserted %d bytes of data from %s. (ID=%d)", len(data), w.Addr().IP, id)
}
