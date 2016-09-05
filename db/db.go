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
	"fmt"
	"log"
	"path"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"gopkg.in/gorp.v1"
)

var Mutex = &sync.RWMutex{}

type Keylib struct {
	ID     int64
	Time   int64
	Thread string
}

type Lookup struct {
	ID     int64
	Thread string
	Addr   string
}

type Recent struct {
	ID     int64
	Time   int64
	Hash   string
	Thread string
}
type Sugtag struct {
	ID     int64
	Thread string
	Tag    string
}
type Thread struct {
	ID     int64
	Thread string
}
type UserTag struct {
	ID     int64
	Thread string
	Tag    string
}
type Record struct {
	ID      int64
	Time    int64
	Hash    string
	Thread  string
	Body    string
	Deleted bool
}

func (r *Record) Recstr() string {
	return fmt.Sprintf("%d<>%s<>%s", r.Time, r.Hash, r.Body)
}

var Map *gorp.DbMap

func Setup() {
	// connect to db using standard Go database/sql API
	// use whatever database/sql driver you wish
	db, err := sql.Open("sqlite3", path.Join(cfg.RunDir, "gou.db"))
	if err != nil {
		log.Fatal(err)
	}

	// construct a gorp DbMap
	Map = &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}

	// add a table, setting the table name to 'posts' and
	// specifying that the Id property is an auto incrementing PK
	Map.AddTableWithName(Keylib{}, "keylib").SetKeys(true, "ID").ColMap("Thread").SetUnique(true)
	Map.AddTableWithName(Lookup{}, "lookup").SetKeys(true, "ID").SetUniqueTogether("Addr", "Thread")
	Map.AddTableWithName(Recent{}, "recent").SetKeys(true, "ID").SetUniqueTogether("Time", "Thread", "Hash")
	Map.AddTableWithName(Sugtag{}, "sugtag").SetKeys(true, "ID").SetUniqueTogether("Thread", "Tag")
	Map.AddTableWithName(Thread{}, "thread").SetKeys(true, "ID").ColMap("Thread").SetUnique(true)
	Map.AddTableWithName(Record{}, "record").SetKeys(true, "ID").SetUniqueTogether("Time", "Hash", "Thread")
	Map.AddTableWithName(UserTag{}, "usertag").SetKeys(true, "ID").SetUniqueTogether("Tag", "Thread")

	// create the table. in a production system you'd generally
	// use a migration tool, or create the tables via scripts
	err = Map.CreateTablesIfNotExists()
	if err != nil {
		log.Fatal(err)
	}
	_, err = Map.Exec("PRAGMA synchronous=OFF;")
	if err != nil {
		log.Fatal(err)
	}

}
