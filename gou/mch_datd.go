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
	"bytes"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

//mchSetup setups handlers for 2ch interface.
func mchSetup(s *loggingServeMux) {
	log.Println("start 2ch interface")
	dataKeyTable.load()
	rtr := mux.NewRouter()

	registToRouter(rtr, "/2ch/", boardApp)
	registToRouter(rtr, "/2ch/dat/{datkey:[^\\.]+}.dat", threadApp)
	registToRouter(rtr, "/2ch/{board:[^/]+}/subject.txt", subjectApp)
	registToRouter(rtr, "/2ch/subject.txt", subjectApp)
	registToRouter(rtr, "/2ch/{board:[^/]+}/head.txt", headApp)
	registToRouter(rtr, "/2ch/head.txt", headApp)
	s.Handle("/2ch/", handlers.CompressHandler(rtr))

	s.registCompressHandler("/test/bbs.cgi", postCommentApp)

}

//boardApp just calls boardApp(), only print title.
func boardApp(w http.ResponseWriter, r *http.Request) {
	a := newMchCGI(w, r)
	if a == nil {
		return
	}
	a.boardApp()
}

//threadApp renders dat files(record data) in the thread.
func threadApp(w http.ResponseWriter, r *http.Request) {
	a := newMchCGI(w, r)
	if a == nil {
		return
	}
	m := mux.Vars(r)
	board := m["board"]
	if board == "" {
		board = "2ch"
	}
	a.threadApp(board, m["datkey"])
}

//subjectApp renders time-subject lines of the thread.
func subjectApp(w http.ResponseWriter, r *http.Request) {
	a := newMchCGI(w, r)
	if a == nil {
		return
	}
	m := mux.Vars(r)
	a.subjectApp(m["board"])
}

//postCommentApp posts one record to the thread.
func postCommentApp(w http.ResponseWriter, r *http.Request) {
	a := newMchCGI(w, r)
	if a == nil {
		return
	}
	a.postCommentApp()
}

//headApp just renders motd.
func headApp(w http.ResponseWriter, r *http.Request) {
	a := newMchCGI(w, r)
	if a == nil {
		return
	}
	a.headApp()
}

//mchCGI is a class for renderring pages of 2ch interface .
type mchCGI struct {
	*cgi
	updateCounter map[string]int
}

//newMchCGI returns mchCGI obj if visitor  is allowed.
//if not allowed print 403.
func newMchCGI(w http.ResponseWriter, r *http.Request) *mchCGI {
	c := newCGI(w, r)
	if c == nil || !c.checkVisitor() {
		w.WriteHeader(403)
		fmt.Fprintf(w, "403 Forbidden")
		return nil
	}

	m := &mchCGI{
		cgi:           c,
		updateCounter: make(map[string]int),
	}
	return m
}

//counterIsUpdate countup threadKey counter and returns true
//for each 4 times.
func (m *mchCGI) counterIsUpdate(threadKey string) bool {
	updateCount := 4
	m.updateCounter[threadKey]++
	m.updateCounter[threadKey] %= updateCount
	return m.updateCounter[threadKey] == 0
}

//serveContent serves str as content with name=name(only used suffix to determine
//data type),time=t after converted cp932. ServeContent is used to make clients possible
//to use range request.
func (m *mchCGI) serveContent(name string, t time.Time, str string) {
	br := bytes.NewReader([]byte(toSJIS(str)))
	http.ServeContent(m.wr, m.req, name, t, br)
}

//boardApp just renders title stripped from url.
func (m *mchCGI) boardApp() {
	l := m.req.FormValue("Accept-Language")
	if l == "" {
		l = "ja"
	}
	message := searchMessage(l)
	m.wr.Header().Set("Content-Type", "text/html; charset=Shift_JIS")
	board := escape(getBoard(m.path))
	text := ""
	if board != "" {
		text = fmt.Sprintf("%s - %s - %s", message["logo"], message["description"], board)
	} else {
		text = fmt.Sprintf("%s - %s", message["logo"], message["description"])
	}

	htmlStr := fmt.Sprintf(
		`<!DOCTYPE html>
        <html><head>
        <meta http-equiv="content-type" content="text/html; charset=Shift_JIS">
        <title>%s</title>
        <meta name="description" content="%s">
        </head><body>
        <h1>%s</h1>
        </body></html>`,
		text, text, text)
	m.serveContent("a.html", time.Time{}, htmlStr)
}

//threadApp load cache specified in the url and returns dat file
//listing records. if cache len=0 or for each refering the cache 4 times
//reloads cache fron network.
func (m *mchCGI) threadApp(board, datkey string) {
	m.wr.Header().Set("Content-Type", "text/plain; charset=Shift_JIS")
	key, err := dataKeyTable.getFilekey(datkey)
	if err != nil {
		m.wr.WriteHeader(404)
		fmt.Fprintf(m.wr, "404 Not Found")
	}
	data := newCache(key)
	data.load()

	if m.checkGetCache() {
		if data.Exists() || data.Len() == 0 {
			data.search(nil)
		} else {
			if m.counterIsUpdate(key) {
				data.search(nil) //can use goroutine
			}
		}
	}

	if !data.Exists() {
		m.wr.WriteHeader(404)
		fmt.Fprintf(m.wr, "404 Not Found")
	}
	thread := makeDat(data, board, m.req.Host)
	str := strings.Join(thread, "\n")
	m.serveContent("a.txt", time.Unix(data.stamp, 0), str)
}

//makeSubjectCachelist returns caches in all cache and in recentlist sorted by recent stamp.
//if board is specified,  returns caches whose tagstr=board.
func (m *mchCGI) makeSubjectCachelist(board string) []*cache {
	cl := newCacheList()
	seen := make([]string, cl.Len())
	for i, c := range cl.Caches {
		seen[i] = c.Datfile
	}
	for _, rec := range recentList.infos {
		if !hasString(seen, rec.datfile) {
			seen = append(seen, rec.datfile)
			c := newCache(rec.datfile)
			c.RecentStamp = rec.stamp
			cl.append(c)
		}
	}
	var result []*cache
	for _, c := range cl.Caches {
		if c.Typee == "thread" {
			result = append(result, c)
		}
	}
	sort.Sort(sort.Reverse(sortByRecentStamp{result}))
	if board == "" {
		return result
	}
	var result2 []*cache
	for _, c := range result {
		if m.hasTag(c, board) {
			result2 = append(result2, c)
		}
	}
	return result2
}

//hasTab adds tags in sugtag to cache and returns true if one of tags ==  board.
func (m *mchCGI) hasTag(c *cache, board string) bool {
	if tl := suggestedTagTable.get(c.Datfile, nil); tl != nil {
		c.tags.Tags = append(c.tags.Tags, tl.Tags...)
	}
	return hasString(c.tags.getTagstrSlice(), board)
}

//subjectApp makes list of records title from caches whose tag is same as one stripped from url.
func (m *mchCGI) subjectApp(board string) {
	var boardEncoded, boardName string
	if board != "" {
		boardEncoded = strDecode(board)
	}
	if boardEncoded != "" {
		boardName = fileDecode("dummy_" + boardEncoded)
	}
	subject, lastStamp := m.makeSubject(boardName)
	m.wr.Header().Set("Content-Type", "text/plain; charset=Shift_JIS")
	m.serveContent("a.txt", time.Unix(lastStamp, 0), strings.Join(subject, "\n"))
}

//makeSubject makes subject.txt(list of records title) from caches with tag=board.
func (m *mchCGI) makeSubject(board string) ([]string, int64) {
	loadFromNet := m.checkGetCache()
	var subjects []string
	cl := m.makeSubjectCachelist(board)
	var lastStamp int64
	for _, c := range cl {
		if !loadFromNet && c.Len() == 0 {
			continue
		}
		if lastStamp < c.stamp {
			lastStamp = c.stamp
		}
		key, err := dataKeyTable.getDatkey(c.Datfile)
		if err != nil {
			log.Println(err)
			continue
		}
		titleStr := fileDecode(c.Datfile)
		if titleStr == "" {
			continue
		}
		titleStr = strings.Trim(titleStr, "\r\n")
		subjects = append(subjects, fmt.Sprintf("%d.dat<>%s (%d)",
			key, titleStr, c.Len()))
	}
	return subjects, lastStamp
}

//headApp renders motd(terms of service).
func (m *mchCGI) headApp() {
	m.wr.Header().Set("Content-Type", "text/plain; charset=Shift_JIS")
	var body string
	err := eachLine(motd, func(line string, i int) error {
		line = strings.TrimSpace(line)
		body += line + "<br>\n"
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	m.serveContent("a.txt", time.Time{}, body)
}
