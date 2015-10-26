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
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"
)

//datastr2ch unixtime str ecpochStr to the certain format string.
//e.g. 2006/01/02(日) 15:04:05.99
func datestr2ch(epoch int64) string {
	t := time.Unix(epoch, 0)
	d := t.Format("2006/01/02(%s) 15:04:05.99")
	wdays := []string{"日", "月", "火", "水", "木", "金", "土"}
	return fmt.Sprintf(d, wdays[t.Weekday()])
}

//resTable maps id[:8] and its number.
type resTable struct {
	id2num map[string]int
	num2id []string
}

//newResTable creates ane returns a resTable maps instance.
func newResTable(ca *cache) *resTable {
	r := &resTable{
		make(map[string]int),
		make([]string, ca.Len()+1),
	}
	ca.load()
	for i, k := range ca.keys() {
		rec := ca.get(k, nil)
		r.num2id[i+1] = rec.ID[:8]
		r.id2num[rec.ID[:8]] = i + 1
	}
	return r
}

//makeDat makes dat lines of 2ch from cache.
func makeDat(ca *cache, board, host string) []string {
	dat := make([]string, len(ca.keys()))
	table := newResTable(ca)

	for i, k := range ca.keys() {
		rec := ca.get(k, nil)
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
			name, rec.GetBodyValue("main", ""), datestr2ch(rec.Stamp), makeBody(rec, host, board, table))
		if i == 0 {
			comment += fileDecode(ca.Datfile)
		}
		dat[i] = comment
	}

	return dat
}

//makeBody makes a dat line after stamp.
func makeBody(rec *record, host, board string, table *resTable) string {
	body := rec.GetBodyValue("body", "")
	body += makeAttachLink(rec, host)
	body = makeRSSAnchor(body, table)
	body = makeBracketLink(body, host, board, table)
	return body
}

//makeAttachLink makes and returns attached file link.
func makeAttachLink(rec *record, sakuHost string) string {
	if rec.GetBodyValue("attach", "") == "" {
		return ""
	}
	url := fmt.Sprintf("http://%s/thread.cgi/%s/%s/%d.%s",
		sakuHost, rec.datfile, rec.ID, rec.Stamp, rec.GetBodyValue("suffix", "txt"))
	return "<br><br>[Attached]<br>" + url
}

//makeRSSAnchor replace id to the record number.
func makeRSSAnchor(body string, table *resTable) string {
	reg := regexp.MustCompile("&gt;&gt;([0-9a-f]{8})")
	return reg.ReplaceAllStringFunc(body, func(str string) string {
		id := reg.FindStringSubmatch(str)[1]
		no := table.id2num[id]
		return "&gt;&gt;" + strconv.Itoa(no)
	})
}

//makeBracketLink add links to [[hoe]] .
func makeBracketLink(body, datHost, board string, table *resTable) string {
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
		datkey, err := dataKeyTable.getDatkey(file)
		if err != nil {
			log.Println(err)
			return body
		}
		if result["id"] == "" {
			url := fmt.Sprintf("http://%s/test/read.cgi/%s/%d/", datHost, board, datkey)
			return fmt.Sprintf("[[%s(%s)]]", result["title"], url)
		}
		ca := newCache(file)
		table = newResTable(ca)
		no := table.id2num[result["id"]]
		url := fmt.Sprintf("http://%s/test/read.cgi/%s/%d/%d", datHost, board, datkey, no)
		return fmt.Sprintf("[[%s(&gt;&gt;%d %s)]]", result["title"], no, url)
	})
}
