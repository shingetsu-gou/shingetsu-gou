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

package keylib

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/boltdb/bolt"
	"github.com/shingetsu-gou/shingetsu-gou/db"
	"github.com/shingetsu-gou/shingetsu-gou/mch"
	"github.com/shingetsu-gou/shingetsu-gou/recentlist"
	"github.com/shingetsu-gou/shingetsu-gou/record"
	"github.com/shingetsu-gou/shingetsu-gou/thread"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

func getThread(stamp int64) (string, error) {
	var thread string
	k := db.MustTob(stamp)
	err := db.DB.View(func(tx *bolt.Tx) error {
		_, err := db.Get(tx, "keylibST", k, &thread)
		return err
	})
	return thread, err
}

func getTime(thread string) (int64, error) {
	var stamp int64
	k := db.MustTob(thread)
	err := db.DB.View(func(tx *bolt.Tx) error {
		_, err := db.Get(tx, "keylibTS", k, &stamp)
		return err
	})
	return stamp, err
}

//Load loads from the file, adds stamps/datfile pairs from cachelist and recentlist.
//and saves to file.
func Load() {
	allCaches := thread.AllCaches()
	allRecs := recentlist.GetRecords()
	db.DB.Update(func(tx *bolt.Tx) error {
		for _, c := range allCaches {
			setFromCache(tx, c)
		}
		for _, rec := range allRecs {
			c := thread.NewCache(rec.Datfile)
			setFromCache(tx, c)
		}
		return nil
	})
}

//setEntry stores stamp/value.
func setEntry(tx *bolt.Tx, stamp int64, filekey string) {
	sb := db.MustTob(stamp)
	fb := db.MustTob(filekey)
	err := db.Put(tx, "keylibST", sb, fb)
	if err != nil {
		log.Print(err)
	}
	err = db.Put(tx, "keylibTS", fb, sb)
	if err != nil {
		log.Print(err)
	}
}

//setFromCache adds cache.datfile/timestamp pair if not exists.
func setFromCache(tx *bolt.Tx, ca *thread.Cache) {
	_, err := getTime(ca.Datfile)
	if err == nil {
		return
	}
	var firstStamp int64
	if !ca.HasRecord() {
		firstStamp = ca.RecentStamp()
	} else {
		if rec := ca.LoadRecords(record.Alive); len(rec) > 0 {
			firstStamp = rec[rec.Keys()[0]].Stamp
		}
	}
	if firstStamp == 0 {
		firstStamp = time.Now().Add(-24 * time.Hour).Unix()
	}
	for err = nil; err != nil; firstStamp++ {
		_, err = getThread(firstStamp)
	}
	setEntry(tx, firstStamp, ca.Datfile)
}

//GetDatkey returns stamp from filekey.
//if not found, tries to read from cache.
func GetDatkey(filekey string) (int64, error) {
	v, err := getTime(filekey)
	if err == nil {
		return v, nil
	}
	c := thread.NewCache(filekey)
	db.DB.Update(func(tx *bolt.Tx) error {
		setFromCache(tx, c)
		return nil
	})
	return getTime(filekey)
}

//GetFilekey returns value from datkey(stamp).
func GetFilekey(nDatkey int64) string {
	v, err := getThread(nDatkey)
	if err == nil {
		return v
	}
	return ""
}

//MakeBracketLink changes str in brackets to the html links format.
func MakeBracketLink(body, datHost, board string, table *mch.ResTable) string {
	regs := []*regexp.Regexp{
		regexp.MustCompile("^(?P<title>[^/]+)$"),
		regexp.MustCompile("^/(?P<type>[a-z]+)/(?P<title>[^/]+)$"),
		regexp.MustCompile("^(?P<title>[^/]+)/(?P<id>[0-9a-f]{8})$"),
		regexp.MustCompile("^/(?P<type>[a-z]+)/(?P<title>[^/]+)/(?P<id>[0-9a-f]{8})$"),
	}
	reg := regexp.MustCompile(`\[\[([^<>]+?)\]\]`)
	return reg.ReplaceAllStringFunc(body, func(str string) string {
		link := reg.FindStringSubmatch(str)[1]
		result := make(map[string]string)
		for _, r := range regs {
			if match := r.FindStringSubmatch(link); match != nil {
				for i, name := range r.SubexpNames() {
					result[name] = match[i]
				}
				break
			}
		}
		if result["title"] == "" {
			return result["body"]
		}
		if result["type"] == "" {
			result["type"] = "thread"
		}
		file := util.FileEncode(result["type"], result["title"])
		datkey, err := GetDatkey(file)
		if err != nil {
			log.Println(err)
			return body
		}
		if result["id"] == "" {
			url := fmt.Sprintf("http://%s/test/read.cgi/%s/%d/", datHost, board, datkey)
			return fmt.Sprintf("[[%s(%s)]]", result["title"], url)
		}
		ca := thread.NewCache(file)
		table = mch.NewResTable(ca)
		no := table.ID2num[result["id"]]
		url := fmt.Sprintf("http://%s/test/read.cgi/%s/%d/%d", datHost, board, datkey, no)
		return fmt.Sprintf("[[%s(&gt;&gt;%d %s)]]", result["title"], no, url)
	})
}

//MakeBody makes a dat body(message) line after stamp.
func MakeBody(rec *record.Record, host, board string, table *mch.ResTable) string {
	body := rec.GetBodyValue("body", "")
	body += rec.MakeAttachLink(host)
	body = table.MakeRSSAnchor(body)
	body = MakeBracketLink(body, host, board, table)
	return body
}

//MakeDat makes dat lines of 2ch from cache.
func MakeDat(ca *thread.Cache, board, host string) []string {
	recs := ca.LoadRecords(record.Alive)
	dat := make([]string, len(recs))
	table := mch.NewResTable(ca)

	i := 0
	for _, datfile := range recs.Keys() {
		rec := recs[datfile]
		err := rec.Load()
		if err != nil {
			log.Println(err)
			continue
		}
		name := rec.GetBodyValue("name", "")
		if name == "" {
			name = "名無しさん"
		}
		if rec.GetBodyValue("pubkey", "") != "" {
			name += "◆" + rec.GetBodyValue("pubkey", "")[:10]
		}
		comment := fmt.Sprintf("%s<>%s<>%s<>%s<>",
			name, rec.GetBodyValue("main", ""), util.Datestr2ch(rec.Stamp), MakeBody(rec, host, board, table))
		if i == 0 {
			comment += util.FileDecode(ca.Datfile)
		}
		dat[i] = comment
		i++
	}

	return dat
}
