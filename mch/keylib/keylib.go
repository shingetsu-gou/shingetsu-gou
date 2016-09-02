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
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/mch"
	"github.com/shingetsu-gou/shingetsu-gou/recentlist"
	"github.com/shingetsu-gou/shingetsu-gou/record"
	"github.com/shingetsu-gou/shingetsu-gou/thread"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//DatakeyTable stores cache stamp and cache datfile name pair.
var datakey2filekey = make(map[int64]string)
var filekey2datkey = make(map[string]int64)
var mutex sync.RWMutex

//loadInternal loads stamp/value from the file .
func loadInternal() {
	err := util.EachLine(cfg.Datakey(), func(line string, i int) error {
		if line == "" {
			return nil
		}
		dat := strings.Split(strings.TrimSpace(line), "<>")
		stamp, err := strconv.ParseInt(dat[0], 10, 64)
		if err != nil {
			log.Println(err)
			return nil
		}
		setEntry(stamp, dat[1])
		return nil
	})
	if err != nil {
		log.Println(err)
	}
}

//Load loads from the file, adds stamps/datfile pairs from cachelist and recentlist.
//and saves to file.
func Load() {
	loadInternal()
	for _, c := range thread.NewCacheList().Caches {
		setFromCache(c)
	}
	for _, rec := range recentlist.GetRecords() {
		c := thread.NewCache(rec.Datfile)
		setFromCache(c)
	}
	save()
}

//save saves stamp<>value to the file.
func save() {
	mutex.RLock()
	str := make([]string, len(datakey2filekey))
	i := 0
	for stamp, filekey := range datakey2filekey {
		str[i] = fmt.Sprintf("%d<>%s", stamp, filekey)
		i++
	}
	mutex.RUnlock()
	cfg.Fmutex.Lock()
	err := util.WriteSlice(cfg.Datakey(), str)
	cfg.Fmutex.Unlock()
	if err != nil {
		log.Println(err)
	}
}

//setEntry stores stamp/value.
func setEntry(stamp int64, filekey string) {
	mutex.Lock()
	defer mutex.Unlock()
	datakey2filekey[stamp] = filekey
	filekey2datkey[filekey] = stamp
}

//setFromCache adds cache.datfile/timestamp pair if not exists.
func setFromCache(ca *thread.Cache) {
	mutex.RLock()
	_, exist := filekey2datkey[ca.Datfile]
	mutex.RUnlock()
	if exist {
		return
	}
	var firstStamp int64
	if !ca.HasRecord() {
		firstStamp = ca.RecentStamp()
	} else {
		if rec := ca.LoadRecords(); len(rec) > 0 {
			firstStamp = rec[rec.Keys()[0]].Stamp
		}
	}
	if firstStamp == 0 {
		firstStamp = time.Now().Add(-24 * time.Hour).Unix()
	}
	for exist := true; !exist; firstStamp++ {
		mutex.RLock()
		_, exist = datakey2filekey[firstStamp]
		mutex.RUnlock()
	}
	setEntry(firstStamp, ca.Datfile)
}

//GetDatkey returns stamp from filekey.
//if not found, tries to read from cache.
func GetDatkey(filekey string) (int64, error) {
	mutex.RLock()
	if v, exist := filekey2datkey[filekey]; exist {
		mutex.RUnlock()
		return v, nil
	}
	mutex.RUnlock()
	c := thread.NewCache(filekey)
	setFromCache(c)
	save()
	mutex.RLock()
	defer mutex.RUnlock()
	if v, exist := filekey2datkey[filekey]; exist {
		return v, nil
	}
	return -1, errors.New(filekey + " not found")
}

//GetFilekey returns value from datkey(stamp).
func GetFilekey(nDatkey int64) string {
	mutex.RLock()
	defer mutex.RUnlock()
	if v, exist := datakey2filekey[nDatkey]; exist {
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
	recs := ca.LoadRecords()
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
