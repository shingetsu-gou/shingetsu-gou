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

package gou

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

//DatakeyTable stores cache stamp and cache datfile name pair.
type DatakeyTable struct {
	*Config
	*Global
	datakey2filekey map[int64]string
	filekey2datkey  map[string]int64
	mutex           sync.RWMutex
}

//newDatakeyTable make DataKeyTable obj.
func newDatakeyTable(cfg *Config, gl *Global) *DatakeyTable {
	d := &DatakeyTable{
		Config:          cfg,
		Global:          gl,
		datakey2filekey: make(map[int64]string),
		filekey2datkey:  make(map[string]int64),
	}
	return d
}

//loadInternal loads stamp/value from the file .
func (d *DatakeyTable) loadInternal() {
	err := eachLine(d.Datakey(), func(line string, i int) error {
		if line == "" {
			return nil
		}
		dat := strings.Split(strings.TrimSpace(line), "<>")
		stamp, err := strconv.ParseInt(dat[0], 10, 64)
		if err != nil {
			log.Println(err)
			return nil
		}
		d.setEntry(stamp, dat[1])
		return nil
	})
	if err != nil {
		log.Println(err)
	}
}

//load loads from the file, adds stamps/datfile pairs from cachelist and recentlist.
//and saves to file.
func (d *DatakeyTable) load() {
	d.loadInternal()
	for _, c := range newCacheList(d.Config, d.Global).Caches {
		d.setFromCache(c)
	}
	for _, rec := range d.RecentList.infos {
		c := newCache(rec.datfile, d.Config, d.Global)
		d.setFromCache(c)
	}
	d.save()
}

//save saves stamp<>value to the file.
func (d *DatakeyTable) save() {
	str := make([]string, len(d.datakey2filekey))
	i := 0
	d.mutex.RLock()
	for stamp, filekey := range d.datakey2filekey {
		str[i] = fmt.Sprintf("%d<>%s", stamp, filekey)
		i++
	}
	d.mutex.RUnlock()
	d.Fmutex.Lock()
	err := writeSlice(d.Datakey(), str)
	d.Fmutex.Unlock()
	if err != nil {
		log.Println(err)
	}
}

//setEntry stores stamp/value.
func (d *DatakeyTable) setEntry(stamp int64, filekey string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.datakey2filekey[stamp] = filekey
	d.filekey2datkey[filekey] = stamp
}

//setFromCache adds cache.datfile/timestamp pair if not exists.
func (d *DatakeyTable) setFromCache(ca *cache) {
	d.mutex.RLock()
	if _, exist := d.filekey2datkey[ca.Datfile]; exist {
		d.mutex.RUnlock()
		return
	}
	var firstStamp int64
	if !ca.hasRecord() {
		firstStamp = ca.recentStamp()
	} else {
		if rec := ca.loadRecords(); len(rec) > 0 {
			firstStamp = rec[rec.keys()[0]].Stamp
		}
	}
	if firstStamp == 0 {
		firstStamp = time.Now().Add(-24 * time.Hour).Unix()
	}
	for {
		if _, exist := d.datakey2filekey[firstStamp]; !exist {
			break
		}
		firstStamp++
	}
	d.mutex.RUnlock()
	d.setEntry(firstStamp, ca.Datfile)
}

//getDatKey returns stamp from filekey.
//if not found, tries to read from cache.
func (d *DatakeyTable) getDatkey(filekey string) (int64, error) {
	d.mutex.RLock()
	if v, exist := d.filekey2datkey[filekey]; exist {
		d.mutex.RUnlock()
		return v, nil
	}
	d.mutex.RUnlock()
	c := newCache(filekey, d.Config, d.Global)
	d.setFromCache(c)
	d.save()
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	if v, exist := d.filekey2datkey[filekey]; exist {
		return v, nil
	}
	return -1, errors.New(filekey + " not found")
}

//getFileKey returns value from datkey(stamp) string.
func (d *DatakeyTable) getFilekey(datkey string) (string, error) {
	nDatkey, err := strconv.ParseInt(datkey, 10, 64)
	if err != nil {
		log.Println(err)
		return "", err
	}
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	if v, exist := d.datakey2filekey[nDatkey]; exist {
		return v, nil
	}
	return "", fmt.Errorf("%s not found", datkey)
}

//makeBracketLink add links to [[hoe]] .
func (d *DatakeyTable) makeBracketLink(body, datHost, board string, table *resTable) string {
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
		file := fileEncode(result["type"], result["title"])
		datkey, err := d.getDatkey(file)
		if err != nil {
			log.Println(err)
			return body
		}
		if result["id"] == "" {
			url := fmt.Sprintf("http://%s/test/read.cgi/%s/%d/", datHost, board, datkey)
			return fmt.Sprintf("[[%s(%s)]]", result["title"], url)
		}
		ca := newCache(file, d.Config, d.Global)
		table = newResTable(ca)
		no := table.id2num[result["id"]]
		url := fmt.Sprintf("http://%s/test/read.cgi/%s/%d/%d", datHost, board, datkey, no)
		return fmt.Sprintf("[[%s(&gt;&gt;%d %s)]]", result["title"], no, url)
	})
}

//makeBody makes a dat line after stamp.
func (d *DatakeyTable) makeBody(rec *record, host, board string, table *resTable) string {
	body := rec.GetBodyValue("body", "")
	body += makeAttachLink(rec, host)
	body = makeRSSAnchor(body, table)
	body = d.makeBracketLink(body, host, board, table)
	return body
}

//makeDat makes dat lines of 2ch from cache.
func (d *DatakeyTable) makeDat(ca *cache, board, host string) []string {
	recs := ca.loadRecords()
	dat := make([]string, len(recs))
	table := newResTable(ca)

	i := 0
	for _, rec := range recs {
		err := rec.load()
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
			name, rec.GetBodyValue("main", ""), datestr2ch(rec.Stamp), d.makeBody(rec, host, board, table))
		if i == 0 {
			comment += fileDecode(ca.Datfile)
		}
		dat[i] = comment
		i++
	}

	return dat
}
