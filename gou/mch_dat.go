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
	"net"
	"regexp"
	"strconv"
	"time"
)

func datestr2ch(epochStr string) string {
	epoch, err := strconv.ParseInt(epochStr, 10, 64)
	if err != nil {
		log.Println(err)
		return ""
	}
	t := time.Unix(epoch, 0)
	d := t.Format("2006/01/02(%s) 15:04:05.99")
	wdays := []string{"日", "月", "火", "水", "木", "金", "土"}
	return fmt.Sprintf(d, wdays[t.Weekday()])
}

type resTable struct {
	id2num map[string]int
	num2id map[int]string
}

func newResTable(ca *cache) *resTable {
	r := &resTable{
		make(map[string]int),
		make(map[int]string),
	}
	ca.load()
	for i, k := range ca.keys() {
		rec := ca.get(k, nil)
		r.num2id[i+1] = rec.id[:8]
		r.id2num[rec.id[:8]] = i + 1
	}
	return r
}

func makeDat(ca *cache, host string, board string) []string {
	dat := make([]string, len(ca.keys()))
	table := newResTable(ca)

	for i, k := range ca.keys() {
		rec := ca.get(k, nil)
		err := rec.load()
		if err != nil {
			log.Println(err)
		}
		name := rec.Get("name", "")
		if name == "" {
			name = "名無しさん"
		}
		if rec.Get("pubkey", "") != "" {
			name += "◆" + rec.Get("pubkey", "")[:10]
		}
		comment := fmt.Sprintf("%s<>%s<>%s<>%s<>",
			name, rec.Get("main", ""), datestr2ch(rec.Get("stamp", "")), makeBody(rec, host, board, table))
		if i == 0 {
			comment += fileDecode(ca.datfile)
		}
		comment += "\n"
		dat[i] = comment
		rec.free()
	}

	return dat
}

func makeBody(rec *record, host, board string, table *resTable) string {
	var datHost, sakuHost string
	if serverName != "" {
		datHost = serverName
		sakuHost = serverName
	} else {
		tcp, err := net.ResolveTCPAddr("tcp", host)
		if err != nil {
			log.Println(err)
			return ""
		}
		tcp.Port = datPort
		datHost = tcp.String()
		tcp.Port = defaultPort
		sakuHost = tcp.String()
	}
	body := makeAttachLink(rec, sakuHost)
	body = makeRssAnchor(body, table)
	body = makeBracketLink(body, datHost, board, table)
	return body
}

func makeAttachLink(rec *record, sakuHost string) string {
	if rec.Get("attach", "") != "" {
		return rec.Get("body", "")
	}
	url := fmt.Sprintf("http://%s/thread.cgi/%s/%s/%d/%s", sakuHost, rec.datfile, rec.id, rec.stamp, rec.Get("suffix", "txt"))
	return rec.Get("body", "") + "<br><br>[Attached]<br>" + url
}

func makeRssAnchor(body string, table *resTable) string {
	reg := regexp.MustCompile("&gt;&gt;([0-9a-f]{8})")
	return reg.ReplaceAllStringFunc(body, func(id string) string {
		no := table.id2num[id]
		return strconv.Itoa(no)
	})
}

func makeBracketLink(body string, datHost, board string, table *resTable) string {
	reg := regexp.MustCompile("\\[\\[([^<>]+?)\\]\\]")
	return reg.ReplaceAllStringFunc(body, func(link string) string {
		regs := []*regexp.Regexp{
			regexp.MustCompile("^(?P<title>[^/]+)$"),
			regexp.MustCompile("^/(?P<type>[a-z]+)/(?P<title>[^/]+)$"),
			regexp.MustCompile("^(?P<title>[^/]+)/(?P<id>[0-9a-f]{8})$"),
			regexp.MustCompile("^/(?P<type>[a-z]+)/(?P<title>[^/]+)/(?P<id>[0-9a-f]{8})$"),
		}
		var title, typ, id string
		for _, r := range regs {
			if match := r.FindStringSubmatch(link); match != nil {
				result := make(map[string]string)
				for i, name := range r.SubexpNames() {
					result[name] = match[i]
				}
				title = result["title"]
				typ = result["type"]
				id = result["id"]
				break
			}
		}
		if title == "" {
			return body
		}
		if typ != "" {
			typ = "thread"
		}
		file := fileEncode(typ, title)
		datkey, err := dkTable.getDatkey(file)
		if err != nil {
			log.Println(err)
			return body
		}
		if id != "" {
			url := fmt.Sprintf("http://%s/test/read.cgi/%s/%d/", datHost, board, datkey)
			return fmt.Sprintf("[[%s(%s)]]", title, url)
		}
		ca := newCache(file)
		table = newResTable(ca)
		no := table.id2num[id]
		url := fmt.Sprintf("http://%s/test/read.cgi/%s/%d/%d", datHost, board, datkey, no)
		return fmt.Sprintf("[[%s(&gt;&gt;%s %s)]]", title, id, url)
	})
}
