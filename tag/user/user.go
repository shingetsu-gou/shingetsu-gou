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

package user

import (
	"log"

	"github.com/boltdb/bolt"
	"github.com/shingetsu-gou/shingetsu-gou/db"
	"github.com/shingetsu-gou/shingetsu-gou/tag"
)

//String  returns string form of usertags.
func String(thread string) string {
	tags := GetByThread(thread)
	return tags.String()
}

//Len  returns # of usertags.
func Len(thread string) int {
	var r map[string]struct{}
	err := db.DB.View(func(tx *bolt.Tx) error {
		var errr error
		r, errr = db.GetMap(tx, "usertag", []byte(thread))
		return errr
	})
	if err != nil {
		return 0
	}
	return len(r)
}

//Has returns true if thread has the tag.
func Has(thread string, tag ...string) bool {
	rr := false
	err := db.DB.View(func(tx *bolt.Tx) error {
		for _, t := range tag {
			if db.HasVal(tx, "usertag", []byte(thread), t) {
				rr = true
				return nil
			}
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	return rr
}

//Get tags from the disk and returns Slice.
func Get() tag.Slice {
	var r []string
	err := db.DB.View(func(tx *bolt.Tx) error {
		var err error
		r, err = db.KeyStrings(tx, "usertagTag")
		return err
	})
	if err != nil {
		return nil
	}
	return tag.NewSlice(r)
}

//GetStrings gets thread tags from the disk
func GetStrings(thread string) []string {
	var r []string
	err := db.DB.View(func(tx *bolt.Tx) error {
		var err error
		r, err = db.MapKeys(tx, "usertag", []byte(thread))
		return err
	})
	if err != nil {
		return nil
	}
	return r
}

//GetByThread gets thread tags from the disk
func GetByThread(thread string) tag.Slice {
	r := GetStrings(thread)
	return tag.NewSlice(r)
}

//Add saves tag strings.
func Add(thread string, tag []string) {
	err := db.DB.Update(func(tx *bolt.Tx) error {
		return AddTX(tx, thread, tag)
	})
	if err != nil {
		log.Println(err)
	}
}

//AddTX saves tag strings.
func AddTX(tx *bolt.Tx, thread string, tag []string) error {
	for _, t := range tag {
		if err := db.PutMap(tx, "usertag", []byte(thread), t); err != nil {
			return err
		}
		if err := db.PutMap(tx, "usertagTag", []byte(t), thread); err != nil {
			return err
		}
	}
	return nil
}

//Set remove all tags and saves tag strings.
func Set(thread string, tag []string) {
	err := db.DB.Update(func(tx *bolt.Tx) error {
		ts, err := db.GetMap(tx, "usertag", []byte(thread))
		if err != nil {
			log.Println(err)
		}
		for t := range ts {
			err = db.DelMap(tx, "usertagTag", []byte(t), thread)
			if err != nil {
				return err
			}
		}
		if err := db.Del(tx, "usertag", []byte(thread)); err != nil {
			log.Println(err)
		}
		return AddTX(tx, thread, tag)
	})
	if err != nil {
		log.Print(err)
	}
}
