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
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/axgle/mahonia"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func mchSetup(s *http.ServeMux) {
	log.Println("start 2ch interface")
	dkTable.load()
	rtr := mux.NewRouter()

	rtr.Handle("/2ch/{board:[^/]+}/$", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newMchCGI(w, r)
		if a == nil {
			return
		}
		a.boardApp()
	})))
	rtr.Handle("/2ch/(board:[^/]+}/dat/{datkey:([^.]+}\\.dat", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newMchCGI(w, r)
		if a == nil {
			return
		}
		m := mux.Vars(r)
		a.threadApp(m["board"], m["datkey"])
	})))
	rtr.Handle("/2ch/(board:[^/]+}/subject\\.txt", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newMchCGI(w, r)
		if a == nil {
			return
		}
		m := mux.Vars(r)
		a.subjectApp(m["board"])
	})))
	rtr.Handle("/2ch/test/bbs\\.cgi", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newMchCGI(w, r)
		if a == nil {
			return
		}
		a.postCommentApp()
	})))
	rtr.Handle("/2ch/(board:[^/]+}/head\\.txt$", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newMchCGI(w, r)
		if a == nil {
			return
		}
		a.headApp()
	})))
	rtr.Handle("/2ch/", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		br := bytes.NewReader([]byte("404 Not Found"))
		http.ServeContent(w, r, "a.txt", time.Time{}, br)
	})))
}

type mchCGI struct {
	*cgi
	mutex         sync.Mutex
	updateCounter map[string]int
}

func newMchCGI(w http.ResponseWriter, r *http.Request) *mchCGI {
	c := newCGI(w, r)
	r.ParseForm()
	isopen := c.isAdmin || c.isFriend || c.isVisitor
	logRequest(r)
	if !isopen {
		w.WriteHeader(403)
		br := bytes.NewReader([]byte("403 Forbidden"))
		http.ServeContent(w, r, "a.txt", time.Time{}, br)
		return nil
	}

	m := &mchCGI{}
	m.cgi = c
	m.updateCounter = make(map[string]int)
	return m
}

func (m *mchCGI) checkGetCache() bool {
	if !m.isFriend && !m.isAdmin {
		return false
	}
	agent := m.getCP932("User-Agent")
	reg, err := regexp.Compile(robot)
	if err != nil {
		log.Println(err)
		return true
	}
	if reg.MatchString(agent) {
		return false
	}
	return true
}

func (m *mchCGI) counterIsUpdate(threadKey string) bool {
	updateCount := 4
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.updateCounter[threadKey]++
	m.updateCounter[threadKey] %= updateCount
	return m.updateCounter[threadKey] == updateCount
}
func (m *mchCGI) serveContent(name string, t time.Time, str string) {
	str = mahonia.NewEncoder("cp932").ConvertString(str)
	br := bytes.NewReader([]byte(str))
	http.ServeContent(m.wr, m.req, name, t, br)
}
func (m *mchCGI) boardApp() {
	l := m.req.FormValue("Accept-Language")
	if l == "" {
		l = "ja"
	}
	message := searchMessage(l)
	m.wr.Header().Set("Content-Type", "text/html; charset=Shift_JIS")
	m.wr.WriteHeader(200)
	board := escape(getBoard(m.req.URL.Path))
	text := ""
	if board != "" {
		text = fmt.Sprintf("%s - %s - %s", message["logo"], message["description"], message["board"])
	} else {
		text = fmt.Sprintf("%s - %s", message["logo"], message["description"])
	}

	htmlStr := fmt.Sprintf(`
        <!DOCTYPE html>
        <html><head>
        <meta http-equiv="content-type" content="text/html; charset=Shift_JIS">
        <title>{text}</title>
        <meta name="description" content="{text}">
        </head><body>
        <h1>{text}</h1>
        </body></html>`, text)
	m.serveContent("a.html", time.Time{}, htmlStr)
}

func (m *mchCGI) threadApp(board, datkey string) {
	m.wr.Header().Set("Content-Type", "text/plain; charset=Shift_JIS")
	key, err := dkTable.getFilekey(datkey)
	if err != nil {
		m.wr.WriteHeader(404)
		m.serveContent("a.txt", time.Time{}, "404 Not Found")
	}
	data := newCache(key, nil, nil)
	data.load()

	if m.checkGetCache() {
		if data.exists() || data.Len() == 0 {
			data.search(nil, nil)
		} else {
			if m.counterIsUpdate(key) {
				go data.search(nil, nil)
			}
		}
	}

	if !data.exists() {
		m.wr.WriteHeader(404)
		m.serveContent("a.txt", time.Time{}, "404 Not Found")
	}
	thread := makeDat(data, m.req.URL.Host, board)
	m.wr.WriteHeader(200)
	str := strings.Join(thread, "\n")
	m.serveContent("a.txt", time.Unix(data.stamp, 0), str)
}

func (m *mchCGI) makeSubjectCachelist(board string) []*cache {
	rl := newRecentList()
	cl := newCacheList()
	seen := make([]string, cl.Len())
	for i, c := range cl.caches {
		seen[i] = c.datfile
	}
	for _, rec := range rl.tiedlist {
		if !hasString(stringSlice(seen), rec.datfile) {
			seen = append(seen, rec.datfile)
			c := newCache(rec.datfile, nil, nil)
			c.recentStamp = rec.stamp
			cl.append(c)
		}
	}
	result := make([]*cache, 0)
	for _, c := range cl.caches {
		if c.typee == "thread" {
			result = append(result, c)
		}
	}
	sort.Sort(sort.Reverse(sortByRecentStamp{result}))
	if board == "" {
		return result
	}
	result2 := make([]*cache, 0)
	sugtags := newSuggestedTagTable()
	for _, c := range result {
		if m.hasTag(c, board, sugtags) {
			result2 = append(result2, c)

		}
	}
	return result2
}

func (m *mchCGI) hasTag(c *cache, board string, sugtag *suggestedTagTable) bool {
	tags := c.tags
	if tl := sugtag.Get(c.datfile, nil); tl != nil {
		tags.tags = append(tags.tags, tl.tags...)
	}
	return hasString(stringSlice(tagSliceTostringSlice(tags.tags)), board)
}

func (m *mchCGI) subjectApp(board string) {
	reg := regexp.MustCompile("2ch_(\\S+)")
	if strings.HasPrefix(board, "2ch") && !reg.MatchString(board) {
		m.wr.WriteHeader(404)
		m.wr.Header().Set("Content-Type", "text/plain")
		m.serveContent("a.txt", time.Time{}, "404 Not Found")
		return
	}
	ma := reg.FindStringSubmatch(board)
	var boardEncoded, boardName string
	if ma != nil {
		boardEncoded = strDecode(ma[1])
	}
	if boardEncoded != "" {
		boardName, _ = fileDecode("dummy_" + boardEncoded)
	}
	subject, lastStamp := m.makeSubject(boardName)
	m.wr.WriteHeader(200)
	m.wr.Header().Set("Content-Type", "text/plain; charset=Shift_JIS")
	m.serveContent("a.txt", time.Unix(lastStamp, 0), strings.Join(subject, "\n"))
}

func (m *mchCGI) makeSubject(board string) ([]string, int64) {
	loadFromNet := m.checkGetCache()
	subjects := make([]string, 0)
	cl := m.makeSubjectCachelist(board)
	var lastStamp int64
	for _, c := range cl {
		if !loadFromNet && len(c.recs) == 0 {
			continue
		}
		if lastStamp < c.stamp {
			lastStamp = c.stamp
		}
		key, err := dkTable.getDatkey(c.datfile)
		if err != nil {
			log.Println(err)
			continue
		}
		titleStr, _ := fileDecode(c.datfile)
		if titleStr != "" {
			titleStr = strings.Replace(titleStr, "\n", "", -1)
		}
		subjects = append(subjects, fmt.Sprintf("%s.dat<>%s (%d)\n",
			key, titleStr, len(c.recs)))
	}
	return subjects, lastStamp
}

func (m *mchCGI) headApp() {
	m.wr.Header().Set("Content-Type", "text/plain; charset=Shift_JIS")
	var body string
	eachLine(motd, func(line string, i int) error {
		line = strings.TrimSpace(line)
		body += line + "<br>\n"
		return nil
	})
	m.serveContent("a.txt", time.Time{}, body)
}
