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
	"html"
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

	registToRouter(rtr, "/thread.cgi/", printIndex)

	reg := "/thread.cgi/thread_{datfile:[0-9A-F]+)/{stamp:[0-9a-f]{32}}/s{id:\\d+}\\.{thumbnailSize:\\d+x\\d+}\\.{suffix:.*}"
	registToRouter(rtr, reg, printAttach)

	reg = "/thread.cgi/thread_{datfile:[0-9A-F]+)/{stamp:[0-9a-f]{32}}/{id:\\d+}\\.{suffix:.*}"
	registToRouter(rtr, reg, printAttach)

	reg = "/thread.cgi/{path:[^/]+}/?$"
	registToRouter(rtr, reg, printThread)

	reg = "/thread.cgi/{path:([^/]+}/{id:[0-9a-f]{8}}$"
	registToRouter(rtr, reg, printThread)

	reg = "/thread.cgi/{path:[^/]+}/p{page:[0-9]+}$"
	registToRouter(rtr, reg, printThread)

	s.Handle("/", handlers.CompressHandler(rtr))
}

func printIndex(w http.ResponseWriter, r *http.Request) {
	if a := newThreadCGI(w, r); a != nil {
		a.printIndex()
	}
}

func printAttach(w http.ResponseWriter, r *http.Request) {
	if a := newThreadCGI(w, r); a != nil {
		m := mux.Vars(r)
		a.printAttach(m["datfile"], m["stamp"], m["id"], m["thumbnailSize"], m["suffix"])
	}
}
func printThread(w http.ResponseWriter, r *http.Request) {
	if a := newThreadCGI(w, r); a != nil {
		m := mux.Vars(r)
		a.printThread(m["path"], m["id"], m["page"])
	}
}

type threadCGI struct {
	*cgi
}

func newThreadCGI(w http.ResponseWriter, r *http.Request) *threadCGI {
	c := newCGI(w, r)

	if c == nil || !c.checkVisitor() {
		c.print403("")
		return nil
	}

	c.host = serverName
	if c.host == "" {
		c.host = r.Host
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
		title := strEncode(fileDecode(datfile))
		t.print302(threadURL + querySeparator + title + "#r" + id)
	}
	t.print404(nil, "")
}

func (t *threadCGI) setCookie(ca *cache, access string) []*http.Cookie {
	c := http.Cookie{}
	c.Expires = time.Now().Add(saveCookie)
	c.Path = threadURL + querySeparator + strEncode(fileDecode(ca.Datfile))
	c.Name = "access"
	c.Value = strconv.FormatInt(time.Now().Unix(), 10)
	if access == "" {
		return []*http.Cookie{&c}
	}
	cc := http.Cookie{}
	cc.Name = "tmpaccess"
	cc.Value = access
	return []*http.Cookie{&c, &cc}
}
func (t *threadCGI) printPageNavi(page int, ca *cache, id string) {
	first := ca.Len() / threadPageSize
	if ca.Len() == 0%threadPageSize {
		first++
	}
	pages := make([]int, first)
	for i := 0; i < first; i++ {
		pages[i] = i
	}
	s := struct {
		Page           int
		Cache          *cache
		Path           string
		ID             string
		First          int
		ThreadCGI      string
		Message        message
		ThreadPageSize int
		Pages          []int
	}{
		page,
		ca,
		t.path,
		id,
		first,
		threadURL,
		t.m,
		threadPageSize,
		pages,
	}
	renderTemplate("page_navi", s, t.wr)
}
func (t *threadCGI) printTag(ca *cache) {
	s := struct {
		Cache      *cache
		Tags       []string
		Classname  string
		Target     string
		GatewayCGI string
		adminCGI   string
	}{
		ca,
		ca.tags.getTagstrSlice(),
		"tags",
		"changes",
		gatewayURL,
		adminURL,
	}
	renderTemplate("thread_tags", s, t.wr)
}

func (t *threadCGI) printThread(path, id, page string) {
	filePath := fileEncode("thread", t.path)
	ca := newCache(filePath)
	if id != "" && t.req.FormValue("ajax") != "" {
		t.printThreadAjax(id)
		return
	}
	switch {
	case ca.hasRecord():
	case t.checkGetCache():
		if t.req.FormValue("search_new_file") != "" {
			ca.setupDirectories()
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
	nPage, err := strconv.Atoi(page)
	if err != nil {
		log.Println(err)
		return
	}
	if useCookie && ca.Len() > 0 && id == "" && page == "" {
		cookie, err := t.req.Cookie("access")
		if err != nil {
			access = cookie.Value
		} else {
			log.Println(err)
		}
		newcookie = t.setCookie(ca, access)
	}
	rss := gatewayURL + "/rss"
	t.header(t.path, rss, newcookie, false, nil)
	tags := strings.Fields(strings.TrimSpace(t.req.FormValue("tag")))
	if t.isAdmin && len(tags) > 0 {
		ca.tags.addString(tags)
		ca.tags.sync()
		userTagList.addString(tags)
		userTagList.sync()
	}
	t.printTag(ca)
	var lastrec *record
	ids := ca.keys()
	if ca.Len() > 0 && page == "" && id == "" && len(ids) == 0 {
		lastrec = ca.recs[ids[len(ids)-1]]
	}
	resAnchor := t.resAnchor(lastrec.ID[:8], threadURL, t.path, false)
	s := struct {
		Path      string
		Cache     *cache
		Lastrec   *record
		IsFriend  bool
		IsAdmin   bool
		Message   message
		threadCGI string
		adminCGI  string
		ResAnchor string
	}{
		t.path,
		ca,
		lastrec,
		t.isFriend,
		t.isAdmin,
		t.m,
		threadURL,
		adminURL,
		resAnchor,
	}
	renderTemplate("thread_top", s, t.wr)
	t.printPageNavi(nPage, ca, id)
	fmt.Fprintln(t.wr, "</p>\n<dl id=\"records\">")
	var inrange []string
	switch {
	case id != "":
		inrange = ids
	case t.path != "":
		inrange = ids[len(ids)-threadPageSize*(nPage+1) : len(ids)-threadPageSize*nPage]
	default:
		inrange = ids[len(ids)-threadPageSize*(nPage+1):]
	}
	for _, k := range inrange {
		rec := ca.get(k, nil)
		if (id == "" || rec.ID[:8] == id) && rec.loadBody() == nil {
			t.printRecord(ca, rec)
		}
	}
	fmt.Fprintln(t.wr, "</dl>")
	escapedPath := html.EscapeString(t.path)
	escapedPath = strings.Replace(escapedPath, "  ", "&nbsp;&nbsp;", -1)
	ss := struct {
		Cache   *cache
		Message message
	}{
		ca,
		t.m,
	}
	renderTemplate("thread_bottom", ss, t.wr)
	if ca.Len() > 0 {
		t.printPageNavi(nPage, ca, id)
		fmt.Fprintf(t.wr, "</p>")
	}
	t.printPostForm(ca)
	t.printTag(ca)
	t.removeFileForm(ca, escapedPath)
	t.footer(t.makeMenubar("bottom", rss))
}

func (t *threadCGI) printThreadAjax(id string) {
	filePath := fileEncode("thread", t.path)
	ca := newCache(filePath)
	if !ca.hasRecord() {
		return
	}
	fmt.Fprintln(t.wr, "<dl>")
	for _, rec := range ca.recs {
		if id == "" || rec.ID[:8] == id && rec.loadBody() == nil {
			t.printRecord(ca, rec)
		}
	}
	fmt.Fprintln(t.wr, "<dl>")
}

func (t *threadCGI) printRecord(ca *cache, rec *record) {
	thumbnailSize := ""
	var suffix string
	var attachSize int64
	if rec.GetBodyValue("attach", "") != "" {
		attachFile := rec.attachPath("", "")
		attachSize = fileSize(attachFile)
		suffix = rec.GetBodyValue("suffix", "")
		reg := regexp.MustCompile("^[0-9A-Za-z]+")
		if !reg.MatchString(suffix) {
			suffix = "txt"
		}
		typ := mime.TypeByExtension(suffix)
		if typ == "" {
			typ = "text/plain"
		}
		if isValidImage(typ, attachFile) {
			thumbnailSize = defaultThumbnailSize
		}
	}
	body := rec.GetBodyValue("body", "")
	body = t.htmlFormat(body, threadURL, t.path, false)
	removeID := rec.GetBodyValue("remove_id", "")[:8]
	resAnchor := t.resAnchor(removeID, threadURL, t.path, false)

	s := struct {
		Cache      *cache
		Rec        *record
		Sid        string
		Path       string
		AttachSize int64
		Suffix     string
		Body       string
		ThreadCGI  string
		Thumbnail  string
		IsAdmin    bool
		RemoveID   string
		ResAnchor  string
	}{
		ca,
		rec,
		rec.GetBodyValue("id", "")[:8],
		t.path,
		attachSize,
		suffix,
		body,
		threadURL,
		thumbnailSize,
		t.isAdmin,
		removeID,
		resAnchor,
	}
	renderTemplate("record", s, t.wr)
}

func (t *threadCGI) printPostForm(ca *cache) {
	mimes := []string{
		".css", ".gif", ".htm", ".html", ".jpg", ".js", ".pdf", ".png", ".svg",
		".txt", ".xml",
	}
	s := struct {
		Cache      *cache
		Suffixes   []string
		Limit      int
		IsAdmin    bool
		Message    message
		ThreadCGI  string
		GatewayCGI string
	}{
		ca,
		mimes,
		recordLimit * 3 >> 2,
		t.isAdmin,
		t.m,
		threadURL,
		gatewayURL,
	}
	renderTemplate("post_form", s, t.wr)
}

func (t *threadCGI) printAttach(datfile, stampStr, id, thumbnailSize, suffix string) {
	ca := newCache(datfile)
	typ := mime.TypeByExtension("suffix")
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
	rec := newRecord(ca.Datfile, stampStr+"_"+id)
	if !rec.Exists() {
		t.print404(ca, "")
		return
	}
	attachFile := rec.attachPath(suffix, thumbnailSize)
	if thumbnailSize != "" && !isFile(attachFile) && (forceThumbnail || thumbnailSize == thumbnailSize) {
		rec.makeThumbnail(suffix, thumbnailSize)
	}
	stamp, err := strconv.ParseInt(stampStr, 10, 64)
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
		defer close(f)

		if err != nil {
			log.Println(err)
			t.print404(ca, "")
		}
		_, err = io.Copy(t.wr, f)
		if err != nil {
			log.Println(err)
		}
	}
}
