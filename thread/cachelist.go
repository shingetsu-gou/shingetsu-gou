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

package thread

import (
	"log"
	"time"

	"regexp"

	"github.com/boltdb/bolt"
	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/db"
	"github.com/shingetsu-gou/shingetsu-gou/record"
)

//AllCaches returns all  thread names
func AllCaches() Caches {
	var r []string
	err := db.DB.View(func(tx *bolt.Tx) error {
		var err error
		r, err = db.KeyStrings(tx, "thread")
		return err
	})
	if err != nil {
		log.Print(err)
		return nil
	}
	ca := make(Caches, len(r))
	for i, t := range r {
		ca[i] = NewCache(t)
	}
	return ca
}

//Len returns # of Caches
func Len() int {
	var r []string
	err := db.DB.View(func(tx *bolt.Tx) error {
		var err error
		r, err = db.GetPrefixs(tx, "record")
		return err
	})
	if err != nil {
		log.Print(err)
		return 0
	}

	return len(r)
}

//Search reloads records in Caches in cachelist
//and returns slice of cache which matches query.
func Search(q string) Caches {
	reg, err := regexp.Compile(q)
	if err != nil {
		log.Println(err)
		return nil
	}
	var cnt []string
	err = db.DB.View(func(tx *bolt.Tx) error {
		return record.ForEach(tx,
			func(d *record.DB) error {
				if reg.Match([]byte(d.Body)) {
					cnt = append(cnt, d.Datfile)
				}
				return nil
			})
	})
	if err != nil {
		log.Println(err)
	}
	result := make([]*Cache, len(cnt))

	for i, rr := range cnt {
		result[i] = NewCache(rr)
	}
	return result
}

//CleanRecords remove old or duplicates records for each Caches.
func CleanRecords() {
	if cfg.SaveRecord <= 0 {
		return
	}
	err := db.DB.Update(func(tx *bolt.Tx) error {
		return record.ForEach(tx, func(rec *record.DB) error {
			if rec.Head.Stamp < time.Now().Unix()-cfg.SaveRecord {
				rec.Del(tx)
			}
			return nil
		})
	})
	if err != nil {
		log.Println(err)
	}
}

//RemoveRemoved removes files in removed dir if old.
func RemoveRemoved() {
	if cfg.SaveRemoved <= 0 {
		return
	}
	err := db.DB.Update(func(tx *bolt.Tx) error {
		return record.ForEach(tx,
			func(rec *record.DB) error {
				if rec.Deleted && rec.Head.Stamp < time.Now().Unix()-cfg.SaveRemoved {
					rec.Del(tx)
				}
				return nil
			})
	})
	if err != nil {
		log.Println(err)
	}
}
