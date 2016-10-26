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

	"github.com/shingetsu-gou/shingetsu-gou/db"
)

//Head represents one line in updatelist/recentlist
type Head struct {
	Datfile string //cache file name
	Stamp   int64  //unixtime
	ID      string //md5(bodystr)
}

//FromRecentDB makes Head ary from recent db.
func FromRecentDB(query string, args ...interface{}) ([]*Head, error) {
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	var h []*Head
	for rows.Next() {
		r := Head{}
		var id int
		err = rows.Scan(&id, &r.Stamp, &r.ID, &r.Datfile)
		if err != nil {
			log.Print(err)
			return nil, nil
		}
		h = append(h, &r)
	}
	return h, nil
}

//Exists return true if record file exists.
func (u *Head) Exists() bool {
	r, err := db.Int64("select count(*) from record where Thread=? and Hash=? and Stamp =?", u.Datfile, u.ID, u.Stamp)
	if err != nil {
		log.Print(err)
		return false
	}
	return r > 0
}

//Removed return true if record is removed (i.e. exists.in removed path)
func (u *Head) Removed() bool {
	r, err := db.Int64("select count(*) from record where Thread=? and Hash=? and Stamp =? and Deleted=1", u.Datfile, u.ID, u.Stamp)
	if err != nil {
		log.Print(err)
		return false
	}
	return r != 0
}

//Remove moves the record file  to remove path
func (u *Head) Remove() error {
	_, err := db.DB.Exec("update record set Deleted=1 where Thread=? and Hash=? and Stamp =?", u.Datfile, u.ID, u.Stamp)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
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
