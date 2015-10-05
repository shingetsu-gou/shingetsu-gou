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
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func threadSetup(s *http.ServeMux) {
	rtr := mux.NewRouter()

	rtr.Handle("/thread.cgi/", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newThreadCGI(w, r)
		a.printIndex()
	})))
	rtr.Handle("/thread.cgi/thread_{datfile:[0-9A-F]+)/{stamp:[0-9a-f]{32}}/s{id:\\d+}\\.{thumbnailSize:\\d+x\\d+}\\.{suffix:.*}", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newThreadCGI(w, r)
		if a != nil {
			return
		}
		a.printAttach(mux.Vars(r))
	})))
	rtr.Handle("/thread.cgi/thread_{datfile:[0-9A-F]+)/{stamp:[0-9a-f]{32}}/{id:\\d+}\\.{suffix:.*}", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newThreadCGI(w, r)
		if a != nil {
			return
		}
		a.printAttach(mux.Vars(r))
	})))
	rtr.Handle("/thread.cgi/{path:[^/]+}/?$", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newThreadCGI(w, r)
		if a == nil {
			return
		}
		a.printThread(mux.Vars(r))
	})))
	rtr.Handle("/thread.cgi/{path:([^/]+}/{id:[0-9a-f]{8}}$", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newThreadCGI(w, r)
		if a == nil {
			return
		}
		a.printThread(mux.Vars(r))
	})))
	rtr.Handle("/thread.cgi/{path:[^/]+}/p{page:[0-9]+}$", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newThreadCGI(w, r)
		if a == nil {
			return
		}
		a.printThread(mux.Vars(r))
	})))
	s.Handle("/", handlers.CompressHandler(rtr))
}

type threadCGI struct {
	*cgi
}

func newThreadCGI(w http.ResponseWriter, r *http.Request) *threadCGI {
	c := newCGI(w, r)
	r.ParseForm()

	c.host = server_name
	if c.host == "" {
		c.host = r.Host
	}
	if !c.checkVisitor() {
		c.print403("")
		return nil
	}
	return &threadCGI{
		c,
	}
}
func (t *threadCGI) printIndex() {
	if t.req.FormValue("cmd") == "post" && strings.HasPrefix(t.req.FormValue("file"), "thread_") && t.req.Method == "POST" {
		id := t.doPost()
		if id == "" {
			return
		}
		datfile := t.req.FormValue("file")
		f, _ := fileDecode(datfile)
		title := strEncode(f)
		t.print302(thread_cgi + query_separator + title + "#r" + id)
	}
	t.print404(nil, "")
}

func (t *threadCGI) setCookie(ca *cache, access string) []*http.Cookie {
	c := http.Cookie{}
	c.Expires = time.Now().Add(save_cookie)
	f, _ := fileDecode(ca.datfile)
	c.Path = thread_cgi + query_separator + strEncode(f)
	c.Name = "access"
	c.Value = strconv.FormatInt(time.Now().Unix(), 10)
	if access == "" {
		return []*http.Cookie{&c}
	} else {
		cc := http.Cookie{}
		cc.Name = "tmpaccess"
		cc.Value = access
		return []*http.Cookie{&c, &cc}
	}
}
func (t *threadCGI) printPageNavi(page string, ca *cache, path, strPath, id string) {
	first := ca.Len() / thread_page_size
	if ca.Len() == 0%thread_page_size {
		first++
	}
	s := struct {
		*DefaultVariable
		Page    string
		Cache   *cache
		Path    string
		StrPath string
		ID      string
		First   int
	}{
		t.makeDefaultVariable(),
		page,
		ca,
		path,
		strPath,
		id,
		first,
	}
	renderTemplate("page_navi", s, t.wr)
}
func (t *threadCGI) printTag(ca *cache) {
	s := struct {
		*DefaultVariable
		Cache     *cache
		Tags      []string
		Classname string
		Target    string
	}{
		t.makeDefaultVariable(),
		ca,
		tagSliceTostringSlice(ca.tags.tags),
		"tags",
		"changes",
	}
	renderTemplate("thread_tags", s, t.wr)
}

func (t *threadCGI) printThread(param map[string]string) {
	id := param["id"]
	strPath := strEncode(t.req.URL.Path)
	filePath := fileEncode("thread", t.req.URL.Path)
	ca := newCache(filePath, nil, nil)
	if id != "" && t.req.FormValue("ajax") != "" {
		t.printThreadAjax(t.req.URL.Path, id)
		return
	}
	switch {
	case ca.hasRecord():
	case t.checkGetCache():
		if t.req.FormValue("search_new_file") != "" {
			ca.standbyDirectories()
			t.unlock()
		} else {
			t.getCache(ca)
		}
	default:
		t.print404(nil, id)
		return
	}
	var access string
	var newcookie []*http.Cookie
	page := param["page"]
	nPage, err := strconv.Atoi(page)
	if err != nil {
		log.Println(err)
		return
	}
	if use_cookie && ca.Len() > 0 && id == "" && page == "" {
		cookie, err := t.req.Cookie("access")
		if err != nil {
			access = cookie.Value
		} else {
			log.Println(err)
		}
		newcookie = t.setCookie(ca, access)
	}
	rss := gateway_cgi + "/rss"
	t.header(t.req.URL.Path, rss, newcookie, false, nil)
	tags := strings.Split(strings.Trim(t.req.FormValue("tag"), "\r\n"), " \t")
	if t.isAdmin && len(tags) > 0 {
		ca.tags.addString(tags)
		ca.tags.sync()
		utl := newUserTagList()
		utl.addString(tags)
		utl.sync()
	}
	t.printTag(ca)
	var lastrec *record
	ids := ca.keys()
	if ca.Len() > 0 && page == "" && id == "" && len(ids) == 0 {
		lastrec = ca.recs[ids[len(ids)-1]]
	}
	s := struct {
		*DefaultVariable
		Path      string
		StrPath   string
		Cache     *cache
		Lastrec   *record
		threadCGI *threadCGI
	}{
		t.makeDefaultVariable(),
		t.req.URL.Path,
		strPath,
		ca,
		lastrec,
		t,
	}
	renderTemplate("thread_top", s, t.wr)
	t.printPageNavi(page, ca, t.req.URL.Path, strPath, id)
	fmt.Fprintln(t.wr, "</p>\n<dl id=\"records\">")
	var inrange []string
	switch {
	case id != "":
		inrange = ids
	case t.req.URL.Path != "":
		inrange = ids[len(ids)-thread_page_size*(nPage+1) : len(ids)-thread_page_size*nPage]
	default:
		inrange = ids[len(ids)-thread_page_size*(nPage+1):]
	}
	for _, k := range inrange {
		rec := ca.Get(k, nil)
		if (id == "" || rec.id[:8] == id) && rec.loadBody() == nil {
			t.printRecord(ca, rec, t.req.URL.Path, strPath)
		}
		rec.free()
	}
	fmt.Fprintln(t.wr, "</dl>")
	escapedPath := cgiEscape(t.req.URL.Path, false)
	escapedPath = strings.Replace(escapedPath, "  ", "&nbsp;&nbsp;", -1)
	ss := struct {
		*DefaultVariable
		Cache *cache
	}{
		t.makeDefaultVariable(),
		ca,
	}
	renderTemplate("thread_bottom", ss, t.wr)
	if ca.Len() > 0 {
		t.printPageNavi(page, ca, t.req.URL.Path, strPath, id)
		fmt.Fprintf(t.wr, "</p>")
	}
	t.printPostForm(ca)
	t.printTag(ca)
	t.removeFileForm(ca, escapedPath)
	t.footer(t.makeMenubar("bottom", rss))
}

func (t *threadCGI) printThreadAjax(path, id string) {
	strPath := strEncode(path)
	filePath := fileEncode("thread", path)
	ca := newCache(filePath, nil, nil)
	if !ca.hasRecord() {
		return
	}
	fmt.Fprintln(t.wr, "<dl>")
	for _, rec := range ca.recs {
		if id == "" || rec.id[:8] == id && rec.loadBody() == nil {
			t.printRecord(ca, rec, path, strPath)
		}
		rec.free()
	}
	fmt.Fprintln(t.wr, "<dl>")
}

func (t *threadCGI) printRecord(ca *cache, rec *record, path, strPath string) {
	thumbnailSize := ""
	var attachFile, suffix string
	var attachSize int64
	if rec.getDict("attach", "") != "" {
		attachFile = rec.attachPath("", "")
		attachSize = rec.attachSize(attachFile, "", "")
		suffix = rec.getDict("suffix", "")
		reg := regexp.MustCompile("^[0-9A-Za-z]+")
		if !reg.MatchString(suffix) {
			suffix = "txt"
		}
		typ := mime.TypeByExtension(suffix)
		if typ == "" {
			typ = "text/plain"
		}
		if isValidImage(typ, attachFile) {
			thumbnailSize = thumbnail_size
		}
	}
	body := rec.getDict("body", "")
	body = t.htmlFormat(body, thread_cgi, path, false)
	s := struct {
		*DefaultVariable
		Cache      *cache
		Rec        *record
		Sid        string
		Path       string
		StrPath    string
		AttachFile string
		AttachSize int64
		Suffix     string
		Body       string
		threadCGI  *threadCGI
		thumbnail  string
	}{
		t.makeDefaultVariable(),
		ca,
		rec,
		rec.getDict("id", "")[:8],
		path,
		strPath,
		attachFile,
		attachSize,
		suffix,
		body,
		t,
		thumbnailSize,
	}
	renderTemplate("record", s, t.wr)
}

func (t *threadCGI) printPostForm(ca *cache) {
	mimes := []string{
		".css", ".gif", ".htm", ".html", ".jpg", ".js", ".pdf", ".png", ".svg",
		".txt", ".xml",
	}
	s := struct {
		*DefaultVariable
		Cache    *cache
		Suffixes []string
		Limit    int
	}{
		t.makeDefaultVariable(),
		ca,
		mimes,
		record_limit * 3 >> 2,
	}
	renderTemplate("post_form", s, t.wr)
}

func (t *threadCGI) printAttach(param map[string]string) {
	ca := newCache(param["datfile"], nil, nil)
	typ := mime.TypeByExtension(param["suffix"])
	if typ == "" {
		typ = "text/plain"
	}
	switch {
	case ca.hasRecord():
	case t.checkGetCache():
		t.getCache(ca)
	default:
		t.print404(ca, "")
		return
	}
	rec := newRecord(ca.datfile, param["stamp"]+"_"+param["id"])
	if !rec.exists() {
		t.print404(ca, "")
		return
	}
	thumbnailSize := param["thumbnailSize"]
	suffix := param["suffix"]
	attachFile := rec.attachPath(param["suffix"], thumbnailSize)
	if thumbnail_size != "" && !isFile(attachFile) && (force_thumbnail || thumbnailSize == thumbnail_size) {
		rec.makeThumbnail(suffix, thumbnailSize)
	}
	stamp, err := strconv.ParseInt(param["stamp"], 10, 64)
	if err != nil {
		log.Println(err)
		t.print404(ca, "")
		return
	}
	if attachFile != "" {
		t.wr.Header().Set("Content-Type", typ)
		t.wr.Header().Set("Last-Modified", t.rfc822Time(stamp))
		if !isValidImage(typ, attachFile) {
			t.wr.Header().Set("Content-Disposition", "attachmentLast-Modified")
		}
		f, err := os.Open(attachFile)
		defer f.Close()
		if err != nil {
			log.Println(err)
			t.print404(ca, "")
		}
		io.Copy(t.wr, f)
	}
}
