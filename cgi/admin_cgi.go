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
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/node"
	"github.com/shingetsu-gou/shingetsu-gou/thread"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//AdminCfg is config for AdminCFG struct.
//it must be setted before using.
var AdminCfg *AdminCGIConfig

//AdminURL is the url to admin.cgi
const AdminURL = "/admin.cgi"

//AdminSetup registers handlers for admin.cgi
func AdminSetup(s *LoggingServeMux) {
	s.RegistCompressHandler(AdminURL+"/status", printStatus)
	s.RegistCompressHandler(AdminURL+"/edittag", printEdittag)
	s.RegistCompressHandler(AdminURL+"/savetag", saveTagCGI)
	s.RegistCompressHandler(AdminURL+"/search", printSearch)
	s.RegistCompressHandler(AdminURL+"/", execCmd)
}

//execCmd execute command specified cmd form.
//i.e. confirmagion page for deleting rec/file(rdel/fdel) and for deleting.
//(xrdel/xfdel)
func execCmd(w http.ResponseWriter, r *http.Request) {
	a, err := newAdminCGI(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	cmd := a.req.FormValue("cmd")
	rmFiles := a.req.Form["file"]
	rmRecords := a.req.Form["record"]

	log.Println("removing, cmd", cmd, "rmFiles", rmFiles, "rmRecords", rmRecords, "dopost=", a.req.FormValue("dopost"))
	switch cmd {
	case "rdel":
		a.printDeleteRecord(rmFiles, rmRecords)
	case "fdel":
		a.printDeleteFile(rmFiles)
	case "xrdel":
		a.doDeleteRecord(rmFiles, rmRecords, a.req.FormValue("dopost"))
	case "xfdel":
		a.doDeleteFile(rmFiles)
	}
	cgis <- a.cgi
}

//printSearch renders the page for searching if query=""
//or do query if query!=""
func printSearch(w http.ResponseWriter, r *http.Request) {
	a, err := newAdminCGI(w, r)
	defer a.close()
	if err != nil {
		log.Println(err)
		return
	}
	query := a.req.FormValue("query")
	if query == "" {
		a.header(a.m["search"], "", nil, true)
		a.printParagraph("desc_search")
		a.printSearchForm("")
		a.footer(nil)
	} else {
		a.printSearchResult(query)
	}
}

//printStatus renders status info, including
//#linknodes,#knownNodes,#files,#records,cacheSize,selfnode/linknodes/knownnodes
// ip:port,
func printStatus(w http.ResponseWriter, r *http.Request) {
	a, err := newAdminCGI(w, r)
	defer a.close()
	if err != nil {
		log.Println()
		return
	}
	cl := thread.NewCacheList()
	records := 0
	var size int64
	for _, ca := range cl.Caches {
		i := ca.ReadInfo()
		records += i.Len
		size += i.Size
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	s := map[string]string{
		"linked_nodes": strconv.Itoa(a.NodeManager.NodeLen()),
		"files":        strconv.Itoa(cl.Len()),
		"records":      strconv.Itoa(records),
		"cache_size":   fmt.Sprintf("%.1f%s", float64(size)/1024/1024, a.m["mb"]),
		"self_node":    a.Myself.Nodestr(),
		"alloc_mem":    fmt.Sprintf("%.1f%s", float64(mem.Alloc)/1024/1024, a.m["mb"]),
	}
	ns := map[string][]string{
		"linked_nodes": a.NodeManager.GetNodestrSlice(),
	}

	d := struct {
		Status     map[string]string
		NodeStatus map[string][]string
		Message    message
	}{
		s,
		ns,
		a.m,
	}
	a.header(a.m["status"], "", nil, true)
	a.Htemplate.RenderTemplate("status", d, a.wr)
	a.footer(nil)
}

//printEdittag renders the page for editing tags in thread specified by form "file".
func printEdittag(w http.ResponseWriter, r *http.Request) {
	a, err := newAdminCGI(w, r)
	defer a.close()
	if err != nil {
		log.Println(err)
		return
	}
	datfile := a.req.FormValue("file")
	strTitle := util.FileDecode(datfile)
	ca := thread.NewCache(datfile)
	datfile = html.EscapeString(datfile)

	if !ca.Exists() {
		a.print404(nil, "")
		return
	}
	d := struct {
		Message  message
		AdminCGI string
		Datfile  string
		Tags     string
		Sugtags  thread.Tagslice
		Usertags thread.Tagslice
	}{
		a.m,
		AdminURL,
		datfile,
		ca.TagString(),
		a.SuggestedTagTable.Get(ca.Datfile, nil),
		a.UserTag.Get(),
	}
	a.header(fmt.Sprintf("%s: %s", a.m["edit_tag"], strTitle), "", nil, true)
	a.Htemplate.RenderTemplate("edit_tag", d, a.wr)
	a.footer(nil)
}

//saveTagCGI saves edited tags of file and render this file with 302.
func saveTagCGI(w http.ResponseWriter, r *http.Request) {
	a, err := newAdminCGI(w, r)
	defer a.close()
	if err != nil {
		log.Println(err)
		return
	}
	datfile := a.req.FormValue("file")
	tags := a.req.FormValue("tag")
	if datfile == "" {
		return
	}
	ca := thread.NewCache(datfile)
	if !ca.Exists() {
		a.print404(nil, "")
	}
	tl := strings.Fields(tags)
	ca.SetTags(tl)
	ca.SyncTag()
	var next string
	title := util.StrEncode(util.FileDecode(datfile))
	if strings.HasPrefix(datfile, "thread_") {
		next = ThreadURL + "/" + title
	} else {
		next = "/"
	}
	a.print302(next)
}

//AdminCGIConfig is config for AdminCGI struct.
type AdminCGIConfig struct {
	AdminSID          string
	NodeManager       *node.Manager
	Htemplate         *util.Htemplate
	UserTag           *thread.UserTag
	SuggestedTagTable *thread.SuggestedTagTable
	RecentList        *thread.RecentList
	Myself            *node.Myself
}

//adminCGI is for admin.cgi handler.
type adminCGI struct {
	*cgi
	*AdminCGIConfig
}

//newAdminCGI returns adminCGI obj if client is admin.
//if not render 403.
func newAdminCGI(w http.ResponseWriter, r *http.Request) (adminCGI, error) {
	if AdminCfg==nil{
		log.Fatal("must set adminConfig")
	}
	a := adminCGI{
		AdminCGIConfig: AdminCfg,
		cgi:            newCGI(w, r),
	}
	if a.cgi == nil {
		return a, errors.New("cannot make cgi")
	}
	if !a.isAdmin() {
		a.print403()
	}
	return a, nil
}

//makeSid makes md5(rand) id and writes to sid file.
func (a *adminCGI) makeSid() string {
	var r string
	for i := 0; i < 4; i++ {
		r += strconv.Itoa(rand.Int())
	}
	sid := util.MD5digest(r)
	err := ioutil.WriteFile(a.AdminSID, []byte(sid+"\n"), 0755)
	if err != nil {
		log.Println(err)
	}
	return sid
}

//checkSid returns true if form value of "sid" == saved sid.
func (a *adminCGI) checkSid() bool {
	sid := a.req.FormValue("sid")
	bsaved, err := ioutil.ReadFile(a.AdminSID)
	if err != nil {
		log.Println(err)
		return false
	}
	saved := strings.TrimRight(string(bsaved), "\r\n")
	if err := os.Remove(a.AdminSID); err != nil {
		log.Println(err)
	}
	return sid == saved
}

//DeleteRecord is for renderring confirmation to a delete record.
type DeleteRecord struct {
	Message  message
	AdminCGI string
	Datfile  string
	Records  []*thread.Record
	Sid      string
}

//printDeleteRecord renders comfirmation page for deleting a record.
//renders info about rec.
func (a *adminCGI) printDeleteRecord(rmFiles []string, records []string) {
	if rmFiles == nil || records == nil {
		a.print404(nil, "")
		return
	}
	datfile := rmFiles[0]
	sid := a.makeSid()
	recs := make([]*thread.Record, len(records))
	for i, v := range records {
		recs[i] = thread.NewRecord(datfile, v)
	}
	d := DeleteRecord{
		a.m,
		AdminURL,
		datfile,
		recs,
		sid,
	}
	a.header(a.m["del_record"], "", nil, true)
	a.Htemplate.RenderTemplate("delete_record", d, a.wr)
	a.footer(nil)
}

//doDeleteRecord dels records in rmFiles files and 302 to this file page.
//with cheking sid. if dopost tells other nodes.
func (a *adminCGI) doDeleteRecord(rmFiles []string, records []string, dopost string) {
	if a.req.Method != "POST" || !a.checkSid() {
		a.print404(nil, "")
		return
	}
	if rmFiles == nil || records == nil {
		a.print404(nil, "")
		return
	}
	datfile := rmFiles[0]
	next := "/"
	title := util.StrEncode(util.FileDecode(datfile))
	if strings.HasPrefix(title, "thread_") {
		next = ThreadURL + "/" + title
	}
	ca := thread.NewCache(datfile)
	for _, r := range records {
		rec := thread.NewRecord(datfile, r)
		if rec.Remove() == nil && dopost != "" {
			a.postDeleteMessage(ca, rec)
			a.print302(next)
			return
		}
	}
	a.print302(next)
}

//postDeleteMessage tells others deletion of a record.
//and adds to updateList and recentlist.
func (a *adminCGI) postDeleteMessage(ca *thread.Cache, rec *thread.Record) {
	stamp := time.Now().Unix()
	body := make(map[string]string)
	for _, key := range []string{"name", "body"} {
		if v := a.req.FormValue(key); v != "" {
			body[key] = util.Escape(v)
		}
	}
	body["remove_stamp"] = strconv.FormatInt(rec.Stamp, 10)
	body["remove_id"] = rec.ID
	passwd := a.req.FormValue("passwd")
	id := rec.Build(stamp, body, passwd)
	rec.Sync()
	a.RecentList.Append(rec)
	a.RecentList.Sync()
	go a.NodeManager.TellUpdate(ca.Datfile, stamp, id, nil)
}

//printDeleteFile renders the page for confirmation of deleting file.
func (a *adminCGI) printDeleteFile(files []string) {
	if files == nil {
		a.print404(nil, "")
	}
	sid := a.makeSid()
	cas := make([]*thread.Cache, len(files))
	for i, v := range files {
		cas[i] = thread.NewCache(v)
	}
	d := struct {
		Message  message
		adminCGI string
		Files    []*thread.Cache
		Sid      string
	}{
		a.m,
		AdminURL,
		cas,
		sid,
	}
	a.header(a.m["del_file"], "", nil, true)
	a.Htemplate.RenderTemplate("delete_file", d, a.wr)
	a.footer(nil)
}

//doDeleteFile remove files in cache and 302 to changes page.
func (a *adminCGI) doDeleteFile(files []string) {
	if a.req.Method != "POST" || a.checkSid() {
		return
	}
	if files == nil {
		a.print404(nil, "")
	}

	for _, c := range files {
		ca := thread.NewCache(c)
		ca.Remove()
	}
	a.print302(GatewayURL + "/" + "changes")
}

//printSearchForm renders search_form.txt
func (a *adminCGI) printSearchForm(query string) {
	d := struct {
		Query    string
		AdminCGI string
		Message  message
	}{
		query,
		AdminURL,
		a.m,
	}
	a.Htemplate.RenderTemplate("search_form", d, a.wr)
}

//printSearchResult renders cachelist that its datfile matches query.
func (a *adminCGI) printSearchResult(query string) {
	strQuery := html.EscapeString(query)
	title := fmt.Sprintf("%s: %s", a.m["search"], strQuery)
	a.header(title, "", nil, true)
	a.printParagraph("desc_search")
	a.printSearchForm(strQuery)
	reg, err := regexp.Compile(html.EscapeString(query))
	if err != nil {
		a.printParagraph("regexp_error")
		a.footer(nil)
		return
	}
	cl := thread.NewCacheList()
	result := cl.Search(reg)
	for _, i := range cl.Caches {
		if result.Has(i) {
			continue
		}
		if reg.MatchString(util.FileDecode(i.Datfile)) {
			result = append(result, i)
		}
	}
	sort.Sort(sort.Reverse(thread.NewSortByStamp(result)))
	a.printIndexList(result, "", true, false)
	a.footer(nil)
}
