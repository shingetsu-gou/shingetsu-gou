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

package suggest

import (
	"log"

	"github.com/boltdb/bolt"
	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/db"
	"github.com/shingetsu-gou/shingetsu-gou/record"
	"github.com/shingetsu-gou/shingetsu-gou/tag"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//Get returns copy of Slice associated with datfile or returns def if not exists.
func Get(datfile string, def tag.Slice) tag.Slice {
	var r []string
	err := db.DB.View(func(tx *bolt.Tx) error {
		var err error
		r, err = db.MapKeys(tx, "sugtag", []byte(datfile))
		return err
	})
	if err != nil {
		log.Print(err, datfile)
		return def
	}
	tags := make([]*tag.Tag, len(r))
	for i, rr := range r {
		tags[i] = &tag.Tag{
			Tagstr: rr,
		}
	}
	if len(tags) > cfg.TagSize {
		tags = tags[:cfg.TagSize]
	}
	return tag.Slice(tags)
}

//keys return datfile names of Sugtaglist.
func keys() []string {
	var r []string
	err := db.DB.View(func(tx *bolt.Tx) error {
		var err error
		r, err = db.KeyStrings(tx, "sugtag")
		return err
	})
	if err != nil {
		log.Print(err)
		return nil
	}
	return r
}

//AddString adds tags to datfile from tagstrings.
func AddString(tx *bolt.Tx, datfile string, vals []string) {
	for _, v := range vals {
		if !tag.IsOK(v) {
			continue
		}
		if err := db.PutMap(tx, "sugtag", []byte(datfile), v); err != nil {
			log.Print(err)
		}
	}
}

//HasTagstr return true if one of tags has tagstr
func HasTagstr(datfile string, tagstr string) bool {
	var r bool
	err := db.DB.View(func(tx *bolt.Tx) error {
		r = db.HasVal(tx, "sugtag", []byte(datfile), tagstr)
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	return r
}

//String return tagstr string of datfile.
func String(datfile string) string {
	ts := Get(datfile, nil)
	if ts == nil {
		return ""
	}
	return ts.String()
}

//Prune removes Sugtaglists which are not listed in recs,
//or truncates its size to tagsize if listed.
func Prune(recs []*record.Head) {
	tmp := keys()
	for _, r := range recs {
		if l := util.FindString(tmp, r.Datfile); l != -1 {
			tmp = append(tmp[:l], tmp[l+1:]...)
		}
	}
	err := db.DB.Update(func(tx *bolt.Tx) error {
		for _, datfile := range tmp {
			err := db.Del(tx, "sugtag", []byte(datfile))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}
}
