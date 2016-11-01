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

package record

import (
	"crypto/md5"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/shingetsu-gou/shingetsu-gou/db"
)

//Head represents one line in updatelist/recentlist
type Head struct {
	Datfile string //cache file name
	Stamp   int64  //unixtime
	ID      string //md5(bodystr)
}

//ToKey returns key for db.
func (u *Head) ToKey() []byte {
	return db.ToKey(u.Datfile, u.Stamp, u.ID)
}

//Exists return true if record file exists.
func (u *Head) Exists() bool {
	var r bool
	err := db.DB.View(func(tx *bolt.Tx) error {
		var err error
		r, err = db.HasKey(tx, "record", u.ToKey())
		return err
	})
	if err != nil {
		log.Print(err)
		return false
	}
	return r
}

//Remove moves the record file  to remove path
func (u *Head) Remove() error {
	var d *DB
	err := db.DB.Update(func(tx *bolt.Tx) error {
		var err error
		d, err = GetFromDB(tx, u)
		if err != nil {
			return err
		}
		d.Deleted = true
		return d.Put(tx)
	})
	if err != nil {
		log.Print(err)
	}
	return err
}

//Hash returns md5 of Head.
func (u *Head) Hash() [16]byte {
	return md5.Sum([]byte(u.Recstr()))
}

//Recstr returns one line of update/recentlist file.
func (u *Head) Recstr() string {
	return fmt.Sprintf("%d<>%s<>%s", u.Stamp, u.ID, u.Datfile)
}

//Idstr returns real file name of the record file.
func (u *Head) Idstr() string {
	return fmt.Sprintf("%d_%s", u.Stamp, u.ID)
}

//ParseHeadResponse parses response of head and returns Head map.
func ParseHeadResponse(res []string, datfile string) map[string]*Head {
	m := make(map[string]*Head)
	for _, line := range res {
		strs := strings.Split(strings.TrimRight(line, "\n\r"), "<>")
		if len(strs) < 2 {
			err := errors.New("illegal format")
			log.Println(err)
			return nil
		}
		u := &Head{
			ID:      strs[1],
			Datfile: datfile,
		}
		var err error
		u.Stamp, err = strconv.ParseInt(strs[0], 10, 64)
		if err != nil {
			log.Println(err)
			return nil
		}
		m[u.Idstr()] = u
	}
	return m
}
