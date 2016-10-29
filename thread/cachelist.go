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
	"bytes"
	"log"
	"time"

	"encoding/binary"

	"regexp"

	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/db"
	"github.com/shingetsu-gou/shingetsu-gou/record"
)

//AllCaches returns all  thread names
func AllCaches() Caches {
	r, err := db.KeyStrings("thread")
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
	r, err := db.GetPrefixs("record")
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
	err = record.ForEach(
		func(k []byte, i int) bool {
			return true
		},
		func(d *record.DB) error {
			if reg.Match([]byte(d.Body)) {
				cnt = append(cnt, d.Datfile)
			}
			return nil
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
	err := record.ForEach(
		func(k []byte, i int) bool {
			return int64(i) < int64(Len())-cfg.SaveRecord
		},
		func(d *record.DB) error {
			d.Del()
			return nil
		})
	if err != nil {
		log.Println(err)
	}
}

//RemoveRemoved removes files in removed dir if old.
func RemoveRemoved() {
	if cfg.SaveRemoved > 0 {
		return
	}
	min := time.Now().Unix() - cfg.SaveRemoved
	bmin := make([]byte, 8)
	binary.BigEndian.PutUint64(bmin, uint64(min))
	err := record.ForEach(
		func(k []byte, i int) bool {
			return bytes.Compare(k, bmin) < 0
		},
		func(d *record.DB) error {
			d.Del()
			return nil
		})
	if err != nil {
		log.Println(err)
	}
}
