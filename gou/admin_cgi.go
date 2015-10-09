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
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func adminSetup(s *http.ServeMux) {
	registCompressHandler(s, "/admin.cgi/status", printStatus)
	registCompressHandler(s, "/admin.cgi/edittag", printEdittag)
	registCompressHandler(s, "/admin.cgi/savetag", saveTagCGI)
	registCompressHandler(s, "/admin.cgi/search", printSearch)
	registCompressHandler(s, "/admin.cgi", execCmd)
}

func execCmd(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	a := newAdminCGI(w, r)
	if a == nil {
		return
	}
	cmd := a.req.FormValue("cmd")
	rmFiles := a.req.Form["file"]
	rmRecords := a.req.Form["record"]

	switch cmd {
	case "rdel":
		if rmFiles != nil || rmRecords != nil {
			a.print404(nil, "")
		}
		a.printDeleteRecord(rmFiles[0], rmRecords)
	case "fdel":
		if rmFiles != nil {
			a.print404(nil, "")
		}
		a.printDeleteFile(rmFiles)
	case "xrdel":
		if a.req.Method != "POST" || a.checkSid(a.req.FormValue("sid")) {
			return
		}
		if rmFiles != nil || rmRecords != nil {
			a.print404(nil, "")
		}
		a.doDeleteRecord(rmFiles[0], rmRecords, a.req.FormValue("dopost"))
	case "xfdel":
		if a.req.Method != "POST" || a.checkSid(a.req.FormValue("sid")) {
			return
		}
		if rmFiles != nil {
			a.print404(nil, "")
		}
		a.doDeleteFile(rmFiles)
	}
}

func printSearch(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	a := newAdminCGI(w, r)
	if a == nil {
		return
	}
	query := a.req.FormValue("query")
	if query == "" {
		query = a.path[len("search/"):]
	}
	if query == "" {
		query = strDecode(a.req.URL.RawQuery)
	}
	if query == "" {
		a.header(a.m["search"], "", nil, true, nil)
		a.printParagraph(a.m["desc_search"])
		a.printSearchForm("")
		a.footer(nil)
	} else {
		a.printSearchResult(query)
	}
}

func printStatus(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	a := newAdminCGI(w, r)
	if a == nil {
		return
	}
	cl := newCacheList()
	records := 0
	size := 0
	for _, ca := range cl.caches {
		records += ca.Len()
		size += ca.size
	}
	my := nodeList.myself()
	s := struct {
		LinedNodes int
		KnownNodes int
		Files      int
		Records    int
		CacheSize  string
		SelfNode   *node
	}{
		nodeList.Len(),
		searchList.Len(),
		cl.Len(),
		records,
		fmt.Sprintf("%.1f%s", float64(size)/1024/1024, a.m["mb"]),
		my,
	}
	ns := struct {
		LinkedNodes NodeList
		KnownNodes  SearchList
	}{
		*nodeList,
		*searchList,
	}

	d := struct {
		*DefaultVariable
		Status     interface{}
		NodeStatus interface{}
	}{
		a.makeDefaultVariable(),
		s,
		ns,
	}
	a.header(a.m["status"], "", nil, true, nil)
	renderTemplate("status", d, a.wr)
	a.footer(nil)
}

func printEdittag(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	a := newAdminCGI(w, r)
	if a == nil {
		return
	}
	datfile := a.req.FormValue("file")
	strTitle := fileEncode(datfile, "")
	ca := newCache(datfile)
	datfile = html.EscapeString(datfile)

	if !ca.exists() {
		a.print404(nil, "")
		return
	}
	d := struct {
		Datfile  string
		Tags     string
		Sugtags  suggestedTagList
		Usertags UserTagList
	}{
		datfile,
		ca.tags.string(),
		*ca.sugtags,
		*userTagList,
	}
	a.header(fmt.Sprintf("%s: %s", a.m["edit_tag"], strTitle), "", nil, true, nil)
	renderTemplate("edit_tag", d, a.wr)
	a.footer(nil)
}

func saveTagCGI(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	a := newAdminCGI(w, r)
	if a == nil {
		return
	}
	datfile := a.req.FormValue("file")
	tags := a.req.FormValue("tag")
	if datfile == "" {
		return
	}
	ca := newCache(datfile)
	if !ca.exists() {
		a.print404(nil, "")
	}
	tl := strings.Fields(tags)
	ca.tags.update(tl)
	ca.tags.sync()
	userTagList.addString(tl)
	userTagList.sync()
	var next string
	for _, t := range types {
		title := strEncode(fileDecode(datfile))
		if strings.HasPrefix(datfile, t+"_") {
			next = application[t] + querySeparator + title
			break
		}
		next = rootPath
	}
	a.print302(next)
}

type adminCGI struct {
	*cgi
}

func newAdminCGI(w http.ResponseWriter, r *http.Request) *adminCGI {
	c := newCGI(w, r)
	if c == nil || !c.isAdmin {
		c.print403("")
		return nil
	}
	return &adminCGI{
		c,
	}
}

func (a *adminCGI) makeSid() string {
	var r string
	for i := 0; i < 4; i++ {
		r += strconv.Itoa(rand.Int())
	}
	sid := md5digest(r)
	err := ioutil.WriteFile(adminSid, []byte(sid+"\n"), 0755)
	if err != nil {
		log.Println(adminSid, err)
	}
	return sid
}

func (a *adminCGI) checkSid(sid string) bool {
	bsaved, err := ioutil.ReadFile(adminSid)
	if err != nil {
		log.Println(adminSid, err)
		return false
	}
	saved := strings.TrimRight(string(bsaved), "\r\n")
	if err := os.Remove(adminSid); err != nil {
		log.Println(err)
	}
	return sid == saved
}

type DeleteRecord struct {
	*DefaultVariable
	Datfile string
	Records []*record
	Sid     string
}

func (d *DeleteRecord) Getbody(rec *record) string {
	err := rec.loadBody()
	if err != nil {
		log.Println(err)
	}
	recstr := html.EscapeString(rec.recstr())
	return recstr
}

func (a *adminCGI) printDeleteRecord(datfile string, records []string) {
	sid := a.makeSid()
	recs := make([]*record, len(records))
	for i, v := range records {
		recs[i] = newRecord(datfile, v)
	}
	d := DeleteRecord{
		a.makeDefaultVariable(),
		datfile,
		recs,
		sid,
	}
	a.header(a.m["del_record"], "", nil, true, nil)
	renderTemplate("delete_record", d, a.wr)
	a.footer(nil)
}

func (a *adminCGI) doDeleteRecord(datfile string, records []string, dopost string) {
	var next string
	for _, t := range types {
		title := strEncode(fileDecode(datfile))
		if strings.HasPrefix(title, t+"_") {
			next = application[t] + querySeparator + title
			break
		}
		next = rootPath
	}
	ca := newCache(datfile)
	for _, r := range records {
		rec := newRecord(datfile, r)
		ca.size -= int(rec.size())
		if rec.remove() == nil {
			ca.count--
			if dopost != "" {
				ca.syncStatus()
				a.postDeleteMessage(ca, rec)
				break
			}
		}
	}
	if dopost == "" {
		ca.syncStatus()
	}
	a.print302(next)
}

type DelFile struct {
	*DefaultVariable
	Files []string
	Sid   string
}

func (d *DelFile) Gettitle(ca *cache) string {
	for _, t := range types {
		if strings.HasPrefix(ca.datfile, t+"_") {
			return fileDecode(ca.datfile)
		}
	}
	return ca.datfile
}

func (d *DelFile) GetContents(ca *cache) []string {
	contents := make([]string, 0, 2)
	for _, rec := range ca.recs {
		err := rec.loadBody()
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
func (a *adminCGI) postDeleteMessage(ca *cache, rec *record) {
	stamp := time.Now().Unix()
	body := make(map[string]string)
	for _, key := range []string{"name", "body"} {
		if v := a.req.FormValue(key); v != "" {
			body[key] = escape(v)
		}
		body["remote_stamp"] = strconv.FormatInt(rec.stamp, 10)
		body["remote_id"] = rec.id
	}
	passwd := a.req.FormValue("passwd")
	id := rec.build(stamp, body, passwd)
	ca.addData(rec, true)
	ca.syncStatus()
	updateList.append(rec)
	updateList.sync()
	recentList.append(rec)
	recentList.sync()
	nodeList.tellUpdate(ca, stamp, id, nil)

}
func (a *adminCGI) printDeleteFile(files []string) {
	sid := a.makeSid()
	cas := make([]*cache, len(files))
	for i, v := range files {
		cas[i] = newCache(v)
	}
	d := DelFile{
		a.makeDefaultVariable(),
		files,
		sid,
	}
	a.header(a.m["del_file"], "", nil, true, nil)
	renderTemplate("delete_file", d, a.wr)
	a.footer(nil)
}

func (a *adminCGI) doDeleteFile(files []string) {
	for _, c := range files {
		ca := newCache(c)
		ca.remove()
	}
	a.print302(gatewayCgi + querySeparator + "changes")
}

func (a *adminCGI) printSearchForm(query string) {
	d := struct {
		*DefaultVariable
		Query string
	}{
		a.makeDefaultVariable(),
		query,
	}
	renderTemplate("search_form", d, a.wr)
}

func (a *adminCGI) printSearchResult(query string) {
	strQuery := html.EscapeString(query)
	title := fmt.Sprintf("%s: %s", a.m["search"], strQuery)
	a.header(title, "", nil, true, nil)
	a.printParagraph(a.m["desc_search"])
	a.printSearchForm(strQuery)
	reg, err := regexp.Compile(html.EscapeString(query))
	if err != nil {
		a.printParagraph(a.m["regexp_error"])
		a.footer(nil)
		return
	}
	cl := newCacheList()
	result := cl.search(reg)
	for _, i := range cl.caches {
		if result.has(i) {
			continue
		}
		if reg.MatchString(fileDecode(i.datfile)) {
			result = append(result, i)
		}
	}
	a.footer(nil)
}
