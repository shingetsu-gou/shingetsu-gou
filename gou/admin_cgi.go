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
)

//adminSetups registers handlers for admin.cgi
func adminSetup(s *loggingServeMux) {
	s.registCompressHandler("/admin.cgi/status", printStatus)
	s.registCompressHandler("/admin.cgi/edittag", printEdittag)
	s.registCompressHandler("/admin.cgi/savetag", saveTagCGI)
	s.registCompressHandler("/admin.cgi/search", printSearch)
	s.registCompressHandler("/admin.cgi/", execCmd)
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
	cl := newCacheList()
	records := 0
	var size int64
	for _, ca := range cl.Caches {
		i := ca.readInfo()
		records += i.len
		size += i.size
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	s := map[string]string{
		"linked_nodes": strconv.Itoa(nodeManager.nodeLen()),
		"files":        strconv.Itoa(cl.Len()),
		"records":      strconv.Itoa(records),
		"cache_size":   fmt.Sprintf("%.1f%s", float64(size)/1024/1024, a.m["mb"]),
		"self_node":    myself.nodestr,
		"alloc_mem":    fmt.Sprintf("%.1f%s", float64(mem.Alloc)/1024/1024, a.m["mb"]),
	}
	ns := map[string][]string{
		"linked_nodes": nodeManager.getNodestrSlice(),
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
	renderTemplate("status", d, a.wr)
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
	strTitle := fileDecode(datfile)
	ca := newCache(datfile)
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
		Sugtags  tagslice
		Usertags tagslice
	}{
		a.m,
		adminURL,
		datfile,
		ca.tagString(),
		suggestedTagTable.get(ca.Datfile, nil),
		utag.get(),
	}
	a.header(fmt.Sprintf("%s: %s", a.m["edit_tag"], strTitle), "", nil, true)
	renderTemplate("edit_tag", d, a.wr)
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
	ca := newCache(datfile)
	if !ca.Exists() {
		a.print404(nil, "")
	}
	tl := strings.Fields(tags)
	ca.setTags(tl)
	ca.syncTag()
	var next string
	title := strEncode(fileDecode(datfile))
	if strings.HasPrefix(datfile, "thread_") {
		next = application["thread"] + querySeparator + title
	} else {
		next = rootPath
	}
	a.print302(next)
}

//adminCGI is for admin.cgi handler.
type adminCGI struct {
	*cgi
}

//newAdminCGI returns adminCGI obj if client is admin.
//if not render 403.
func newAdminCGI(w http.ResponseWriter, r *http.Request) (adminCGI, error) {
	a := adminCGI{newCGI(w, r)}
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
	sid := md5digest(r)
	err := ioutil.WriteFile(adminSid, []byte(sid+"\n"), 0755)
	if err != nil {
		log.Println(err)
	}
	return sid
}

//checkSid returns true if form value of "sid" == saved sid.
func (a *adminCGI) checkSid() bool {
	sid := a.req.FormValue("sid")
	bsaved, err := ioutil.ReadFile(adminSid)
	if err != nil {
		log.Println(err)
		return false
	}
	saved := strings.TrimRight(string(bsaved), "\r\n")
	if err := os.Remove(adminSid); err != nil {
		log.Println(err)
	}
	return sid == saved
}

//DeleteRecord is for renderring confirmation to a delete record.
type DeleteRecord struct {
	Message  message
	AdminCGI string
	Datfile  string
	Records  []*record
	Sid      string
}

//Getbody retuns contents of rec.
func (d DeleteRecord) Getbody(rec *record) string {
	err := rec.load()
	if err != nil {
		log.Println(err)
	}
	return rec.recstr()
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
	recs := make([]*record, len(records))
	for i, v := range records {
		recs[i] = newRecord(datfile, v)
	}
	d := DeleteRecord{
		a.m,
		adminURL,
		datfile,
		recs,
		sid,
	}
	a.header(a.m["del_record"], "", nil, true)
	renderTemplate("delete_record", d, a.wr)
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
	next := rootPath
	title := strEncode(fileDecode(datfile))
	if strings.HasPrefix(title, "thread_") {
		next = application["thread"] + querySeparator + title
	}
	ca := newCache(datfile)
	for _, r := range records {
		rec := newRecord(datfile, r)
		if rec.remove() == nil && dopost != "" {
			a.postDeleteMessage(ca, rec)
			a.print302(next)
			return
		}
	}
	a.print302(next)
}

//DelFile is for rendering confirmation page of deleting file.
type DelFile struct {
	Message  message
	adminCGI string
	Files    []*cache
	Sid      string
}

//Gettitle returns title part if *_*.
//returns ca.datfile if not.
func (d *DelFile) Gettitle(ca *cache) string {
	if strings.HasPrefix(ca.Datfile, "thread_") {
		return fileDecode(ca.Datfile)
	}
	return ca.Datfile
}

//GetContents returns recstrs of cache.
//len(recstrs) is <=2.
func (d *DelFile) GetContents(ca *cache) []string {
	contents := make([]string, 0, 2)
	recs := ca.loadRecords()
	for _, rec := range recs {
		err := rec.load()
		if err != nil {
			log.Println(err)
		}
		contents = append(contents, escape(rec.recstr()))
		if len(contents) > 2 {
			return contents
		}
	}
	return contents
}

//postDeleteMessage tells others deletion of a record.
//and adds to updateList and recentlist.
func (a *adminCGI) postDeleteMessage(ca *cache, rec *record) {
	stamp := time.Now().Unix()
	body := make(map[string]string)
	for _, key := range []string{"name", "body"} {
		if v := a.req.FormValue(key); v != "" {
			body[key] = escape(v)
		}
	}
	body["remove_stamp"] = strconv.FormatInt(rec.Stamp, 10)
	body["remove_id"] = rec.ID
	passwd := a.req.FormValue("passwd")
	id := rec.build(stamp, body, passwd)
	rec.sync()
	recentList.append(rec)
	recentList.sync()
	go nodeManager.tellUpdate(ca, stamp, id, nil)
}

//printDeleteFile renders the page for confirmation of deleting file.
func (a *adminCGI) printDeleteFile(files []string) {
	if files == nil {
		a.print404(nil, "")
	}
	sid := a.makeSid()
	cas := make([]*cache, len(files))
	for i, v := range files {
		cas[i] = newCache(v)
	}
	d := DelFile{
		a.m,
		adminURL,
		cas,
		sid,
	}
	a.header(a.m["del_file"], "", nil, true)
	renderTemplate("delete_file", d, a.wr)
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
		ca := newCache(c)
		ca.remove()
	}
	a.print302(gatewayURL + querySeparator + "changes")
}

//printSearchForm renders search_form.txt
func (a *adminCGI) printSearchForm(query string) {
	d := struct {
		Query    string
		AdminCGI string
		Message  message
	}{
		query,
		adminURL,
		a.m,
	}
	renderTemplate("search_form", d, a.wr)
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
	cl := newCacheList()
	result := cl.search(reg)
	for _, i := range cl.Caches {
		if result.has(i) {
			continue
		}
		if reg.MatchString(fileDecode(i.Datfile)) {
			result = append(result, i)
		}
	}
	sort.Sort(sort.Reverse(newSortByStamp(result)))
	a.printIndexList(result, "", true, false)
	a.footer(nil)
}
