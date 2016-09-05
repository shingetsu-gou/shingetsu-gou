/*
 * Copyright (c) 2015, Shinya Yagyu
 * All rights reserved.
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 * 1. Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 * 3. Neither the name of the copyright holder nor the names of its
 *    contributors may be used to endorse or promote products derived from this
 *    software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

package db

import (
	"database/sql"
	"log"
	"path"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shingetsu-gou/shingetsu-gou/cfg"
)

var tables = []string{
	`CREATE TABLE IF NOT EXISTS  "keylib" ("ID" integer not null primary key autoincrement, "Stamp" integer, "Thread" varchar(255) unique)`,
	`CREATE TABLE IF NOT EXISTS  "lookup" ("ID" integer not null primary key autoincrement, "Thread" varchar(255), "Addr" varchar(255), unique ("Addr", "Thread"))`,
	`CREATE TABLE IF NOT EXISTS  "recent" ("ID" integer not null primary key autoincrement, "Stamp" integer, "Hash" varchar(255), "Thread" varchar(255), unique ("Stamp", "Thread", "Hash"))`,
	`CREATE TABLE IF NOT EXISTS  "sugtag" ("ID" integer not null primary key autoincrement, "Thread" varchar(255), "Tag" varchar(255), unique ("Thread", "Tag"))`,
	`CREATE TABLE IF NOT EXISTS  "thread" ("ID" integer not null primary key autoincrement, "Thread" varchar(255) unique)`,
	`CREATE TABLE IF NOT EXISTS  "record" ("ID" integer not null primary key autoincrement, "Stamp" integer, "Hash" varchar(255), "Thread" varchar(255), "Body" varchar(255), "Deleted" integer, unique ("Stamp", "Hash", "Thread"))`,
	`CREATE TABLE IF NOT EXISTS  "usertag" ("ID" integer not null primary key autoincrement, "Thread" varchar(255), "Tag" varchar(255), unique ("Tag", "Thread"))`,
}

/*
sqlite in mattn-sqlite3 is copiled with serialized param.
therefore no need to lock.
https://www.sqlite.org/threadsafe.html
*/

var Mutex = &sync.RWMutex{}

var DB *sql.DB

func Setup() {
	dbpath := path.Join(cfg.RunDir, "gou.db")
	var err error
	DB, err = sql.Open("sqlite3", "file:"+dbpath+"?cache=shared&mode=rwc")
	if err != nil {
		log.Fatal(err)
	}
	tx, err := DB.Begin()
	if err != nil {
		log.Fatal(err)
	}
	for _, table := range tables {
		if _, err = DB.Exec(table); err != nil {
			log.Fatal(err)
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	_, err = DB.Exec("PRAGMA synchronous=OFF;")
	if err != nil {
		log.Fatal(err)
	}
}

func String(query string, args ...interface{}) (string, error) {
	var str string
	err := DB.QueryRow(query, args...).Scan(&str)
	return str, err
}

func Strings(query string, args ...interface{}) ([]string, error) {
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	var result []string
	for rows.Next() {
		var str string
		err = rows.Scan(&str)
		if err != nil {
			return nil, err
		}
		result = append(result, str)
	}
	return result, nil
}

func Int64(query string, args ...interface{}) (int64, error) {
	var str int64
	err := DB.QueryRow(query, args...).Scan(&str)
	return str, err
}

func Int64s(query string, args ...interface{}) ([]int64, error) {
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	var result []int64
	for rows.Next() {
		var str int64
		err = rows.Scan(&str)
		if err != nil {
			return nil, err
		}
		result = append(result, str)
	}
	return result, nil
}
