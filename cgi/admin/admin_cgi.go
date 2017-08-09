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

package admin

import (
	"fmt"
	"html"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"errors"

	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/cgi"
	"github.com/shingetsu-gou/shingetsu-gou/myself"
	"github.com/shingetsu-gou/shingetsu-gou/node"
	"github.com/shingetsu-gou/shingetsu-gou/node/manager"
	"github.com/shingetsu-gou/shingetsu-gou/recentlist"
	"github.com/shingetsu-gou/shingetsu-gou/record"
	"github.com/shingetsu-gou/shingetsu-gou/tag"
	"github.com/shingetsu-gou/shingetsu-gou/tag/suggest"
	"github.com/shingetsu-gou/shingetsu-gou/tag/user"
	"github.com/shingetsu-gou/shingetsu-gou/thread"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

var adminSID = ""

//Setup registers handlers for admin.cgi
func Setup(s *cgi.LoggingServeMux) {
	s.RegistCompressHandler(cfg.AdminURL+"/status", printStatus)
	s.RegistCompressHandler(cfg.AdminURL+"/edittag", printEdittag)
	s.RegistCompressHandler(cfg.AdminURL+"/savetag", saveTagCGI)
	s.RegistCompressHandler(cfg.AdminURL+"/search", printSearch)
	s.RegistCompressHandler(cfg.AdminURL+"/", execCmd)
}

//execCmd execute command specified cmd form.
//i.e. confirmagion page for deleting rec/file(rdel/fdel) and for deleting.
//(xrdel/xfdel)
func execCmd(w http.ResponseWriter, r *http.Request) {
	a, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	cmd := a.Req.FormValue("cmd")
	rmFiles := a.Req.Form["file"]
	rmRecords := a.Req.Form["record"]

	switch cmd {
	case "rdel":
		a.printDeleteRecord(rmFiles, rmRecords)
	case "fdel":
		a.printDeleteFile(rmFiles)
	case "xrdel":
		a.doDeleteRecord(rmFiles, rmRecords, a.Req.FormValue("dopost"))
	case "xfdel":
		a.doDeleteFile(rmFiles)
	}
}

//printSearch renders the page for searching if query=""
//or do query if query!=""
func printSearch(w http.ResponseWriter, r *http.Request) {
	a, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	query := a.Req.FormValue("query")
	if query == "" {
		a.Header(a.M["search"], "", nil, true)
		fmt.Fprintf(a.WR, "<p>%s</p>", a.M["desc_search"])
		a.printSearchForm("")
		a.Footer(nil)
	} else {
		a.printSearchResult(query)
	}
}

//printStatus renders status info, including
//#linknodes,#knownNodes,#files,#records,cacheSize,selfnode/linknodes/knownnodes
// ip:port,
func printStatus(w http.ResponseWriter, r *http.Request) {
	a, err := new(w, r)
	if err != nil {
		log.Println()
		return
	}
	records := 0
	var size int64
	for _, ca := range thread.AllCaches() {
		records += ca.Len(record.All)
		size += ca.Size()
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	var port0 string
	switch myself.GetStatus() {
	case cfg.Normal:
		port0 = a.M["opened"]
	case cfg.UPnP:
		port0 = "UPnP"
	case cfg.Port0:
		port0 = a.M["port0"]
	case cfg.Disconnected:
		port0 = a.M["disconnected"]
	}

	s := map[string]string{
		"known_nodes":       strconv.Itoa(manager.NodeLen()),
		"linked_nodes":      strconv.Itoa(manager.ListLen()),
		"files":             strconv.Itoa(thread.Len()),
		"records":           strconv.Itoa(records),
		"cache_size":        fmt.Sprintf("%.1f%s", float64(size)/1024/1024, a.M["mb"]),
		"self_node":         node.Me(false).Nodestr,
		"alloc_mem":         fmt.Sprintf("%.1f%s", float64(mem.Alloc)/1024/1024, a.M["mb"]),
		"connection_status": port0,
	}
	ns := map[string][]string{
		"known_nodes":  manager.GetNodestrSlice(),
		"linked_nodes": manager.GetNodestrSliceInList(),
	}

	d := struct {
		Status     map[string]string
		NodeStatus map[string][]string
		Message    cgi.Message
	}{
		s,
		ns,
		a.M,
	}
	a.Header(a.M["status"], "", nil, true)
	cgi.RenderTemplate("status", d, a.WR)
	a.Footer(nil)
}

//printEdittag renders the page for editing tags in thread specified by form "file".
func printEdittag(w http.ResponseWriter, r *http.Request) {
	a, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	datfile := a.Req.FormValue("file")
	strTitle := util.FileDecode(datfile)
	ca := thread.NewCache(datfile)
	datfile = html.EscapeString(datfile)

	if !ca.Exists() {
		a.Print404(nil, "")
		return
	}
	d := struct {
		Message  cgi.Message
		AdminCGI string
		Datfile  string
		Tags     string
		Sugtags  tag.Slice
		Usertags tag.Slice
	}{
		a.M,
		cfg.AdminURL,
		datfile,
		user.String(ca.Datfile),
		suggest.Get(ca.Datfile, nil),
		user.GetByThread(ca.Datfile),
	}
	a.Header(fmt.Sprintf("%s: %s", a.M["edit_tag"], strTitle), "", nil, true)
	cgi.RenderTemplate("edit_tag", d, a.WR)
	a.Footer(nil)
}

//saveTagCGI saves edited tags of file and render this file with 302.
func saveTagCGI(w http.ResponseWriter, r *http.Request) {
	a, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	datfile := a.Req.FormValue("file")
	tags := a.Req.FormValue("tag")
	if datfile == "" {
		return
	}
	ca := thread.NewCache(datfile)
	if !ca.Exists() {
		a.Print404(nil, "")
	}
	tl := strings.Fields(tags)
	user.Set(datfile, tl)
	var next string
	title := util.StrEncode(util.FileDecode(datfile))
	if strings.HasPrefix(datfile, "thread_") {
		next = cfg.ThreadURL + "/" + title
	} else {
		next = "/"
	}
	a.Print302(next)
}

//adminCGI is for admin.cgi handler.
type adminCGI struct {
	*cgi.CGI
}

//new returns adminCGI obj if client is admin.
//if not render 403.
func new(w http.ResponseWriter, r *http.Request) (*adminCGI, error) {
	c, err := cgi.NewCGI(w, r)
	if err != nil {
		return nil, err
	}
	a := adminCGI{
		CGI: c,
	}
	if !a.IsAdmin() {
		a.Print403()
		return nil, errors.New("permission denied")
	}
	return &a, nil
}

//makeSid makes md5(rand) id and writes to sid file.
func (a *adminCGI) makeSid() string {
	var r string
	for i := 0; i < 4; i++ {
		r += strconv.Itoa(rand.Int())
	}
	adminSID = util.MD5digest(r)
	return adminSID
}

//checkSid returns true if form value of "sid" == saved sid.
func (a *adminCGI) checkSid() bool {
	sid := a.Req.FormValue("sid")
	r := (adminSID != "" && adminSID == sid)
	adminSID = ""
	return r
}

//DeleteRecord is for renderring confirmation to a delete record.
type DeleteRecord struct {
	Message  cgi.Message
	AdminCGI string
	Datfile  string
	Records  []*record.Record
	Sid      string
}

//printDeleteRecord renders comfirmation page for deleting a record.
//renders info about rec.
func (a *adminCGI) printDeleteRecord(rmFiles []string, records []string) {
	if rmFiles == nil || records == nil {
		a.Print404(nil, "")
		return
	}
	datfile := rmFiles[0]
	sid := a.makeSid()
	recs := make([]*record.Record, len(records))
	var err error
	for i, v := range records {
		recs[i], err = record.NewIDstr(datfile, v)
		if err != nil {
			log.Println(err)
		}
	}
	d := DeleteRecord{
		a.M,
		cfg.AdminURL,
		datfile,
		recs,
		sid,
	}
	a.Header(a.M["del_record"], "", nil, true)
	cgi.RenderTemplate("delete_record", d, a.WR)
	a.Footer(nil)
}

//doDeleteRecord dels records in rmFiles files and 302 to this file page.
//with cheking sid. if dopost tells other nodes.
func (a *adminCGI) doDeleteRecord(rmFiles []string, records []string, dopost string) {
	if a.Req.Method != "POST" || !a.checkSid() {
		a.Print404(nil, "")
		return
	}
	if rmFiles == nil || records == nil {
		a.Print404(nil, "")
		return
	}
	datfile := rmFiles[0]
	next := "/"
	title := util.StrEncode(util.FileDecode(datfile))
	if strings.HasPrefix(title, "thread_") {
		next = cfg.ThreadURL + "/" + title
	}
	ca := thread.NewCache(datfile)
	for _, r := range records {
		rec, err := record.NewIDstr(datfile, r)
		if err != nil || rec.Remove() == nil && dopost != "" {
			a.postDeleteMessage(ca, rec)
			a.Print302(next)
			return
		}
	}
	a.Print302(next)
}

//postDeleteMessage tells others deletion of a record.
//and adds to updateList and recentlist.
func (a *adminCGI) postDeleteMessage(ca *thread.Cache, rec *record.Record) {
	stamp := time.Now().Unix()
	body := make(map[string]string)
	for _, key := range []string{"name", "body"} {
		if v := a.Req.FormValue(key); v != "" {
			body[key] = util.Escape(v)
		}
	}
	body["remove_stamp"] = strconv.FormatInt(rec.Stamp, 10)
	body["remove_id"] = rec.ID
	passwd := a.Req.FormValue("passwd")
	id := rec.Build(stamp, body, passwd)
	rec.Sync()
	recentlist.Append(rec.Head)
	go manager.TellUpdate(ca.Datfile, stamp, id, nil)
}

//printDeleteFile renders the page for confirmation of deleting file.
func (a *adminCGI) printDeleteFile(files []string) {
	if files == nil {
		a.Print404(nil, "")
	}
	sid := a.makeSid()
	cas := make([]*thread.Cache, len(files))
	for i, v := range files {
		cas[i] = thread.NewCache(v)
	}
	d := struct {
		Message  cgi.Message
		AdminCGI string
		Files    []*thread.Cache
		Sid      string
	}{
		a.M,
		cfg.AdminURL,
		cas,
		sid,
	}
	a.Header(a.M["del_file"], "", nil, true)
	cgi.RenderTemplate("delete_file", d, a.WR)
	a.Footer(nil)
}

//doDeleteFile remove files in cache and 302 to changes page.
func (a *adminCGI) doDeleteFile(files []string) {
	if a.Req.Method != "POST" || !a.checkSid() {
		a.Print404(nil, "")
	}
	if files == nil {
		a.Print404(nil, "")
	}

	for _, c := range files {
		ca := thread.NewCache(c)
		ca.Remove()
	}
	a.Print302(cfg.GatewayURL + "/" + "changes")
}

//printSearchForm renders search_form.txt
func (a *adminCGI) printSearchForm(query string) {
	d := struct {
		Query    string
		AdminCGI string
		Message  cgi.Message
	}{
		query,
		cfg.AdminURL,
		a.M,
	}
	cgi.RenderTemplate("search_form", d, a.WR)
}

//printSearchResult renders cachelist that its datfile matches query.
func (a *adminCGI) printSearchResult(query string) {
	strQuery := html.EscapeString(query)
	title := fmt.Sprintf("%s: %s", a.M["search"], strQuery)
	a.Header(title, "", nil, true)
	fmt.Fprintf(a.WR, "<p>%s</p>", a.M["desc_search"])
	a.printSearchForm(strQuery)
	reg := html.EscapeString(query)
	regg, err := regexp.Compile(reg)
	if err != nil {
		log.Println(err)
		return
	}
	result := thread.Search(reg)
	for _, i := range thread.AllCaches() {
		if result.Has(i) {
			continue
		}
		if regg.MatchString(util.FileDecode(i.Datfile)) {
			result = append(result, i)
		}
	}
	sort.Sort(sort.Reverse(thread.NewSortByStamp(result, false)))
	a.PrintIndexList(result, "", true, false, "", "")
}
