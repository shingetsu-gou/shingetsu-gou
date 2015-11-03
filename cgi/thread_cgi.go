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

package cgi

import (
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"html/template"
	"log"
	"math"
	"math/rand"
	"mime"
	"mime/multipart"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/shingetsu-gou/shingetsu-gou/thread"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//ThreadURL is the url to thread.cgi
const ThreadURL = "/thread.cgi"

//ThreadSetup setups handlers for thread.cgi
func ThreadSetup(s *LoggingServeMux) {
	rtr := mux.NewRouter()

	registToRouter(rtr, ThreadURL+"/", printThreadIndex)

	reg := ThreadURL + "/{datfile:thread_[0-9A-F]+}/{id:[0-9a-f]{32}}/s{stamp:\\d+}.{thumbnailSize:\\d+x\\d+}.{suffix:.*}"
	registToRouter(rtr, reg, printAttach)

	reg = ThreadURL + "/{datfile:thread_[0-9A-F]+}/{id:[0-9a-f]{32}}/{stamp:\\d+}.{suffix:.*}"
	registToRouter(rtr, reg, printAttach)

	reg = ThreadURL + "/{path:[^/]+}{end:/?$}"
	registToRouter(rtr, reg, printThread)

	reg = ThreadURL + "/{path:[^/]+}/{id:[0-9a-f]{8}}{end:$}"
	registToRouter(rtr, reg, printThread)

	reg = ThreadURL + "/{path:[^/]+}/p{page:[0-9]+}{end:$}"
	registToRouter(rtr, reg, printThread)

	s.Handle(ThreadURL+"/", handlers.CompressHandler(rtr))
}

//printThreadIndex adds records in multiform and redirect to its thread page.
func printThreadIndex(w http.ResponseWriter, r *http.Request) {
	if a, err := newThreadCGI(w, r); err == nil {
		defer a.close()
		a.printThreadIndex()
	}
}

func printAttach(w http.ResponseWriter, r *http.Request) {
	if a, err := newThreadCGI(w, r); err == nil {
		defer a.close()
		m := mux.Vars(r)
		var stamp int64
		if m["stamp"] != "" {
			var err error
			stamp, err = strconv.ParseInt(m["stamp"], 10, 64)
			if err != nil {
				log.Println(err)
				return
			}
		}
		a.printAttach(m["datfile"], m["id"], stamp, m["thumbnailSize"], m["suffix"])
	}
}

//printThread renders whole thread list page.
func printThread(w http.ResponseWriter, r *http.Request) {
	if a, err := newThreadCGI(w, r); err == nil {
		defer a.close()
		m := mux.Vars(r)
		var page int
		if m["page"] != "" {
			var err error
			page, err = strconv.Atoi(m["page"])
			if err != nil {
				return
			}
		}
		a.printThread(m["path"], m["id"], page)
	}
}

//ThreadCfg is config for threadCGI struct.
//must set beforehand.
var ThreadCfg *ThreadCGIConfig

//ThreadCGIConfig is config for threadCGI struct.
type ThreadCGIConfig struct {
	ThreadPageSize       int
	DefaultThumbnailSize string
	RecordLimit          int
	ForceThumbnail       bool
	Htemplate            *util.Htemplate
	UpdateQue            *thread.UpdateQue
}

//threadCGI is for thread.cgi.
type threadCGI struct {
	*ThreadCGIConfig
	*cgi
}

//newThreadCGI returns threadCGI obj.
func newThreadCGI(w http.ResponseWriter, r *http.Request) (threadCGI, error) {
	if ThreadCfg == nil {
		log.Fatal("must set ThreadCfg")
	}
	t := threadCGI{
		ThreadCGIConfig: ThreadCfg,
		cgi:             newCGI(w, r),
	}

	if t.cgi == nil {
		t.print403()
		return t, errors.New("error while parsing form")
	}
	if !t.checkVisitor() {
		t.print403()
		return t, errors.New("visitor now allowed")
	}
	return t, nil
}

//printThreadIndex adds records in multiform and redirect to its thread page.
func (t *threadCGI) printThreadIndex() {
	err := t.req.ParseMultipartForm(int64(t.RecordLimit) << 10)
	if err != nil {
		t.print404(nil, "")
		return
	}
	if t.req.FormValue("cmd") != "post" || !strings.HasPrefix(t.req.FormValue("file"), "thread_") {
		t.print404(nil, "")
		return
	}
	id := t.doPost()
	if id == "" {
		t.print404(nil, "")
		return
	}
	datfile := t.req.FormValue("file")
	title := util.StrEncode(util.FileDecode(datfile))
	t.print302(ThreadURL + "/" + title + "#r" + id)
}

//setCookie set cookie access=now time,tmpaccess=access var.
func (t *threadCGI) setCookie(ca *thread.Cache, access string) []*http.Cookie {
	const (
		saveCookie = 7 * 24 * time.Hour // Seconds
	)

	c := http.Cookie{}
	c.Expires = time.Now().Add(saveCookie)
	c.Path = ThreadURL + "/" + util.StrEncode(util.FileDecode(ca.Datfile))
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

//printPageNavi renders page_navi.txt, part for paging.
func (t *threadCGI) printPageNavi(path string, page int, ca *thread.Cache, id string) {
	len := ca.ReadInfo().Len
	first := len / t.ThreadPageSize
	if len%t.ThreadPageSize == 0 {
		first++
	}
	pages := make([]int, first+1)
	for i := 0; i <= first; i++ {
		pages[i] = i
	}
	s := struct {
		Page           int
		CacheLen       int
		Path           string
		ID             string
		First          int
		ThreadCGI      string
		Message        message
		ThreadPageSize int
		Pages          []int
	}{
		page,
		len,
		path,
		id,
		first,
		ThreadURL,
		t.m,
		t.ThreadPageSize,
		pages,
	}
	t.Htemplate.RenderTemplate("page_navi", s, t.wr)
}

//printTag renders thread_tags.txt , part for displayng tags.
func (t *threadCGI) printTag(ca *thread.Cache) {
	s := struct {
		Datfile    string
		Tags       []string
		Classname  string
		Target     string
		GatewayCGI string
		AdminCGI   string
		IsAdmin    bool
		Message    message
	}{
		ca.Datfile,
		ca.GetTagstrSlice(),
		"tags",
		"changes",
		GatewayURL,
		AdminURL,
		t.isAdmin(),
		t.m,
	}
	t.Htemplate.RenderTemplate("thread_tags", s, t.wr)
}

//printThreadHead renders head part of thread page with cookie.
func (t *threadCGI) printThreadHead(path, id string, page int, ca *thread.Cache, rss string) error {
	switch {
	case ca.HasRecord():
	case t.checkGetCache():
		ca.SetupDirectories()
		if t.req.FormValue("search_new_file") == "" {
			ca.GetCache()
		}
	default:
		t.print404(nil, id)
		return errors.New("no records")
	}
	var access string
	var newcookie []*http.Cookie
	if ca.HasRecord() && id == "" && page == 0 {
		cookie, err := t.req.Cookie("access")
		if err == nil {
			access = cookie.Value
		} else {
			log.Println(err)
		}
		newcookie = t.setCookie(ca, access)
	}
	t.header(path, rss, newcookie, false)
	return nil
}

//printThreadTop renders toppart of thread page.
func (t *threadCGI) printThreadTop(path, id string, nPage int, ca *thread.Cache) {
	var lastrec *thread.Record
	var resAnchor string
	recs := ca.LoadRecords()
	ids := recs.Keys()
	if ca.HasRecord() && nPage == 0 && id == "" && len(ids) > 0 {
		lastrec = recs[ids[len(ids)-1]]
		resAnchor = t.resAnchor(lastrec.ID[:8], ThreadURL, t.path(), false)
	}
	s := struct {
		Path      string
		Cache     *thread.Cache
		Lastrec   *thread.Record
		IsFriend  bool
		IsAdmin   bool
		Message   message
		ThreadCGI string
		AdminCGI  string
		ResAnchor template.HTML
	}{
		path,
		ca,
		lastrec,
		t.isFriend(),
		t.isAdmin(),
		t.m,
		ThreadURL,
		AdminURL,
		template.HTML(resAnchor),
	}
	t.Htemplate.RenderTemplate("thread_top", s, t.wr)
}

//printThreadBody renders body(records list) part of thread page with paging.
func (t *threadCGI) printThreadBody(id string, nPage int, ca *thread.Cache) {
	recs := ca.LoadRecords()
	ids := recs.Keys()
	fmt.Fprintln(t.wr, "</p>\n<dl id=\"records\">")
	from := len(ids) - t.ThreadPageSize*(nPage+1)
	to := len(ids) - t.ThreadPageSize*(nPage)
	if from < 0 {
		from = 0
	}
	if to < 0 {
		to = 0
	}
	var inrange []string
	switch {
	case id != "":
		inrange = ids
	case nPage > 0:
		inrange = ids[from:to]
	default:
		inrange = ids[from:]
	}

	for _, k := range inrange {
		rec := recs.Get(k, nil)
		if (id == "" || rec.ID[:8] == id) && rec.Load() == nil {
			t.printRecord(ca, rec)
		}
	}

	fmt.Fprintln(t.wr, "</dl>")
}

//printThread renders whole thread list page.
func (t *threadCGI) printThread(path, id string, nPage int) {
	if id != "" && t.req.FormValue("ajax") != "" {
		t.printThreadAjax(id)
		return
	}
	filePath := util.FileEncode("thread", path)
	ca := thread.NewCache(filePath)
	rss := GatewayURL + "/rss"
	if t.printThreadHead(path, id, nPage, ca, rss) != nil {
		return
	}
	tags := strings.Fields(strings.TrimSpace(t.req.FormValue("tag")))
	if t.isAdmin() && len(tags) > 0 {
		ca.AddTags(tags)
		ca.SyncTag()
	}
	t.printTag(ca)
	t.printThreadTop(path, id, nPage, ca)
	t.printPageNavi(path, nPage, ca, id)
	t.printThreadBody(id, nPage, ca)

	escapedPath := html.EscapeString(path)
	escapedPath = strings.Replace(escapedPath, "  ", "&nbsp;&nbsp;", -1)
	ss := struct {
		Cache   *thread.Cache
		Message message
	}{
		ca,
		t.m,
	}
	t.Htemplate.RenderTemplate("thread_bottom", ss, t.wr)

	if ca.HasRecord() {
		t.printPageNavi(path, nPage, ca, id)
		fmt.Fprintf(t.wr, "</p>")
	}
	t.printPostForm(ca)
	t.printTag(ca)
	t.removeFileForm(ca, escapedPath)
	t.footer(t.makeMenubar("bottom", rss))
}

//printThreadAjax renders records in cache id for ajax.
func (t *threadCGI) printThreadAjax(id string) {
	filePath := util.FileEncode("thread", t.path())
	ca := thread.NewCache(filePath)
	if !ca.HasRecord() {
		return
	}
	fmt.Fprintln(t.wr, "<dl>")
	recs := ca.LoadRecords()
	for _, rec := range recs {
		if id == "" || rec.ID[:8] == id && rec.Load() == nil {
			t.printRecord(ca, rec)
		}
	}
	fmt.Fprintln(t.wr, "</dl>")
}

//printRecord renders record.txt , with records in cache ca.
func (t *threadCGI) printRecord(ca *thread.Cache, rec *thread.Record) {
	thumbnailSize := ""
	var suffix string
	var attachSize int64
	if at := rec.GetBodyValue("attach", ""); at != "" {
		log.Println("attached")
		suffix = rec.GetBodyValue("suffix", "")
		attachFile := rec.AttachPath("")
		attachSize = int64(len(at) * 57 / 78)
		reg := regexp.MustCompile("^[0-9A-Za-z]+")
		if !reg.MatchString(suffix) {
			suffix = "txt"
		}
		typ := mime.TypeByExtension("." + suffix)
		if typ == "" {
			typ = "text/plain"
		}
		if util.IsValidImage(typ, attachFile) {
			thumbnailSize = t.DefaultThumbnailSize
		}
	}
	body := rec.GetBodyValue("body", "")
	body = t.htmlFormat(body, ThreadURL, t.path(), false)
	removeID := rec.GetBodyValue("remove_id", "")
	if len(removeID) > 8 {
		removeID = removeID[:8]
	}
	resAnchor := t.resAnchor(removeID, ThreadURL, t.path(), false)

	id8 := rec.ID
	if len(id8) > 8 {
		id8 = id8[:8]
	}
	s := struct {
		Datfile    string
		Rec        *thread.Record
		RecHead    thread.RecordHead
		Sid        string
		Path       string
		AttachSize int64
		Suffix     string
		Body       template.HTML
		ThreadCGI  string
		Thumbnail  string
		IsAdmin    bool
		RemoveID   string
		ResAnchor  string
		Message    message
	}{
		ca.Datfile,
		rec,
		rec.CopyRecordHead(),
		id8,
		t.path(),
		attachSize,
		suffix,
		template.HTML(body),
		ThreadURL,
		thumbnailSize,
		t.isAdmin(),
		removeID,
		resAnchor,
		t.m,
	}
	t.Htemplate.RenderTemplate("record", s, t.wr)
}

//printPostForm renders post_form.txt,page for posting attached file.
func (t *threadCGI) printPostForm(ca *thread.Cache) {
	mimes := []string{
		".css", ".gif", ".htm", ".html", ".jpg", ".js", ".pdf", ".png", ".svg",
		".txt", ".xml",
	}
	s := struct {
		Cache      *thread.Cache
		Suffixes   []string
		Limit      int
		IsAdmin    bool
		Message    message
		ThreadCGI  string
		GatewayCGI string
	}{
		ca,
		mimes,
		t.RecordLimit * 3 >> 2,
		t.isAdmin(),
		t.m,
		ThreadURL,
		GatewayURL,
	}
	t.Htemplate.RenderTemplate("post_form", s, t.wr)
}

//renderAttach render the content of attach file with content-type=typ.
func (t *threadCGI) renderAttach(rec *thread.Record, suffix string, stamp int64, thumbnailSize string) {
	attachFile := rec.AttachPath(thumbnailSize)
	if attachFile == "" {
		return
	}
	typ := mime.TypeByExtension("." + suffix)
	if typ == "" {
		typ = "text/plain"
	}
	t.wr.Header().Set("Content-Type", typ)
	t.wr.Header().Set("Last-Modified", t.rfc822Time(stamp))
	if !util.IsValidImage(typ, attachFile) {
		t.wr.Header().Set("Content-Disposition", "attachment")
	}
	decoded, err := base64.StdEncoding.DecodeString(rec.GetBodyValue("attach", ""))
	if err != nil {
		log.Println(err)
		t.print404(nil, "")
		return
	}
	if thumbnailSize != "" && (t.ForceThumbnail || thumbnailSize == t.DefaultThumbnailSize) {
		decoded = util.MakeThumbnail(decoded, suffix, thumbnailSize)
	}
	_, err = t.wr.Write(decoded)
	if err != nil {
		log.Println(err)
		t.print404(nil, "")
	}
}

//printAttach renders the content of attach file and makes thumnail if needed and possible.
func (t *threadCGI) printAttach(datfile, id string, stamp int64, thumbnailSize, suffix string) {
	ca := thread.NewCache(datfile)
	switch {
	case ca.HasRecord():
	case t.checkGetCache():
		ca.GetCache()
	default:
		t.print404(ca, "")
		return
	}
	rec := thread.NewRecord(ca.Datfile, fmt.Sprintf("%d_%s", stamp, id))
	if !rec.Exists() {
		t.print404(ca, "")
		return
	}
	if err := rec.Load(); err != nil {
		t.print404(ca, "")
		return
	}
	if rec.GetBodyValue("suffix", "") != suffix {
		t.print404(ca, "")
		return
	}
	t.renderAttach(rec, suffix, stamp, thumbnailSize)
}

//errorTime calculates gaussian distribution by box-muller transformation.
func (t *threadCGI) errorTime() int64 {
	const timeErrorSigma = 60 // Seconds

	x1 := rand.Float64()
	x2 := rand.Float64()
	return int64(timeErrorSigma*math.Sqrt(-2*math.Log(x1))*math.Cos(2*math.Pi*x2)) + time.Now().Unix()
}

//guessSuffix guess suffix of attached at from formvalue "suffix"
func (t *threadCGI) guessSuffix(at *attached) string {
	guessSuffix := "txt"
	if at != nil {
		if e := path.Ext(at.Filename); e != "" {
			guessSuffix = strings.ToLower(e)
		}
	}

	suffix := t.req.FormValue("suffix")
	switch {
	case suffix == "" || suffix == "AUTO":
		suffix = guessSuffix
	case strings.HasPrefix(suffix, "."):
		suffix = suffix[1:]
	}
	suffix = strings.ToLower(suffix)
	reg := regexp.MustCompile("[^0-9A-Za-z]")
	return reg.ReplaceAllString(suffix, "")
}

//makeRecord builds and returns record with attached file.
//if nobody render null_article page.
func (t *threadCGI) makeRecord(at *attached, suffix string, ca *thread.Cache) *thread.Record {
	body := make(map[string]string)
	for _, name := range []string{"body", "base_stamp", "base_id", "name", "mail"} {
		if value := t.req.FormValue(name); value != "" {
			body[name] = util.Escape(value)
		}
	}

	if at != nil {
		body["attach"] = at.Data
		body["suffix"] = strings.TrimSpace(suffix)
	}
	if len(body) == 0 {
		t.header(t.m["null_article"], "", nil, true)
		t.footer(nil)
		return nil
	}
	stamp := time.Now().Unix()
	if t.req.FormValue("error") != "" {
		stamp = t.errorTime()
	}
	rec := thread.NewRecord(ca.Datfile, "")
	passwd := t.req.FormValue("passwd")
	rec.Build(stamp, body, passwd)
	return rec
}

//doPost parses multipart form ,makes record of it and adds to cache.
//if form dopost=yes broadcasts it.
func (t *threadCGI) doPost() string {
	attached, attachedErr := t.parseAttached()
	if attachedErr != nil {
		log.Println(attachedErr)
	}
	suffix := t.guessSuffix(attached)
	ca := thread.NewCache(t.req.FormValue("file"))
	rec := t.makeRecord(attached, suffix, ca)
	if rec == nil {
		return ""
	}
	proxyClient := t.req.Header.Get("X_FORWARDED_FOR")
	log.Printf("post %s/%d_%s from %s/%s\n", ca.Datfile, ca.ReadInfo().Stamp, rec.ID, t.req.RemoteAddr, proxyClient)

	if len(rec.Recstr()) > t.RecordLimit<<10 {
		t.header(t.m["big_file"], "", nil, true)
		t.footer(nil)
		return ""
	}
	if rec.IsSpam() {
		t.header(t.m["spam"], "", nil, true)
		t.footer(nil)
		return ""
	}

	if ca.Exists() {
		rec.Sync()
	} else {
		t.print404(nil, "")
		return ""
	}

	if t.req.FormValue("dopost") != "" {
		log.Println(rec.Datfile, rec.ID, "is queued")
		go t.UpdateQue.UpdateNodes(rec, nil)
	}

	return rec.ID[:8]

}

//attached represents attached file name and contents.
type attached struct {
	Filename string
	Data     string
}

//parseAttached reads attached file and returns attached obj.
//if size>recordLimit renders error page.
func (t *threadCGI) parseAttached() (*attached, error) {
	err := t.req.ParseMultipartForm(int64(t.RecordLimit) << 10)
	if err != nil {
		return nil, err
	}
	attach := t.req.MultipartForm
	if len(attach.File) == 0 {
		return nil, errors.New("attached file not found")
	}
	var fpStrAttach *multipart.FileHeader
	for _, v := range attach.File {
		fpStrAttach = v[0]
	}
	f, err := fpStrAttach.Open()
	defer util.Fclose(f)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	var strAttach = make([]byte, t.RecordLimit<<10)
	s, err := f.Read(strAttach)
	if s > t.RecordLimit<<10 {
		log.Println("attached file is too big")
		t.header(t.m["big_file"], "", nil, true)
		t.footer(nil)
		return nil, err
	}
	if err != nil {
		log.Println(err)
		return nil, err
	}
	coded := base64.StdEncoding.EncodeToString(strAttach[:s])
	return &attached{
		fpStrAttach.Filename,
		coded,
	}, nil
}
