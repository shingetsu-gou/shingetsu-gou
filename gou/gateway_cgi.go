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
	"encoding/csv"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

//gatewaySetup setups handlers for gateway.cgi
func gatewaySetup(s *loggingServeMux) {
	s.registCompressHandler("/gateway.cgi/motd", printMotd)
	s.registCompressHandler("/gateway.cgi/mergedjs", printMergedJS)
	s.registCompressHandler("/gateway.cgi/rss", printRSS)
	s.registCompressHandler("/gateway.cgi/recent_rss", printRecentRSS)
	s.registCompressHandler("/gateway.cgi/index", printGatewayIndex)
	s.registCompressHandler("/gateway.cgi/changes", printIndexChanges)
	s.registCompressHandler("/gateway.cgi/recent", printRecent)
	s.registCompressHandler("/gateway.cgi/new", printNew)
	s.registCompressHandler("/gateway.cgi/thread", printGatewayThread)
	s.registCompressHandler("/gateway.cgi/", printTitle)
	s.registCompressHandler("/gateway.cgi/csv/index/", printCSV)
	s.registCompressHandler("/gateway.cgi/csv/changes/", printCSVChanges)
	s.registCompressHandler("/gateway.cgi/csv/recent/", printCSVRecent)
}

//printGateway just redirects to correspoinding url using thread.cgi.
//or renders only title.
func printGatewayThread(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()

	reg := regexp.MustCompile("^/gateway.cgi/(thread)/?([^/]*)$")
	m := reg.FindStringSubmatch(r.URL.Path)
	var uri string
	switch {
	case m == nil:
		printTitle(w, r)
		return
	case m[2] != "":
		uri = application["thread"] + querySeparator + strEncode(m[2])
	case r.URL.RawQuery != "":
		uri = application["thread"] + querySeparator + r.URL.RawQuery
	default:
		printTitle(w, r)
		return
	}
	g := newGatewayCGI(w, r)
	if g == nil {
		return
	}
	g.print302(uri)
}

//printCSV renders csv of caches saved in disk.
func printCSV(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	g := newGatewayCGI(w, r)
	if g == nil {
		return
	}
	cl := newCacheList()
	g.renderCSV(cl)
}

//printCSVChanges renders csv of caches which changes recently and are in disk(validstamp is newer).
func printCSVChanges(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	g := newGatewayCGI(w, r)
	if g == nil {
		return
	}
	cl := newCacheList()
	sort.Sort(sort.Reverse(sortByValidStamp{cl.Caches}))
	g.renderCSV(cl)
}

//printCSVRecent renders csv of caches which are written recently(are updated remotely).
func printCSVRecent(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	g := newGatewayCGI(w, r)
	if g == nil {
		return
	}
	if !g.isFriend && !g.isAdmin {
		g.print403()
		return
	}
	cl := recentList.makeRecentCachelist()
	g.renderCSV(cl)
}

//printRecentRSS renders rss of caches which are written recently(are updated remotely).
//including title,tags,last-modified.
func printRecentRSS(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	g := newGatewayCGI(w, r)
	if g == nil {
		return
	}
	rsss := newRss("UTF-8", "", fmt.Sprintf("%s - %s", g.m["recent"], g.m["logo"]),
		"http://"+g.host, "",
		"http://"+g.host+gatewayURL+querySeparator+"recent_rss", g.m["description"], xsl)
	cl := recentList.makeRecentCachelist()
	for _, ca := range cl.Caches {
		title := escape(fileDecode(ca.Datfile))
		tags := make([]string, ca.tags.Len()+ca.sugtags.Len())
		for i, t := range ca.tags.Tags {
			tags[i] = t.Tagstr
		}
		for i, t := range ca.sugtags.Tags {
			tags[i+ca.tags.Len()] = t.Tagstr
		}
		if _, exist := application[ca.Typee]; !exist {
			continue
		}
		rsss.append(application[ca.Typee][1:]+querySeparator+strEncode(title),
			title, "", "", html.EscapeString(title), tags, ca.RecentStamp, false)
	}
	g.wr.Header().Set("Content-Type", "text/xml; charset=UTF-8")
	if rsss.len() != 0 {
		g.wr.Header().Set("Last-Modified", g.rfc822Time(rsss.Feeds[0].Date))
	}
	rsss.makeRSS1(g.wr)
}

//appendRSS appends cache ca to rss with contents,url to records,stamp,attached file.
func (g *gatewayCGI) appendRSS(rsss *RSS, ca *cache) {
	now := time.Now().Unix()
	if ca.ValidStamp+rssRange < now {
		return
	}
	title := escape(fileDecode(ca.Datfile))
	path := application[ca.Typee] + querySeparator + strEncode(title)
	for _, r := range ca.recs {
		if r.Stamp+rssRange >= now {
			continue
		}
		if err := r.loadBody(); err != nil {
			log.Println(err)
			continue
		}

		desc := rssTextFormat(r.GetBodyValue("body", ""))
		content := g.rssHTMLFormat(r.GetBodyValue("body", ""), application[ca.Typee], title)
		if attach := r.GetBodyValue("attach", ""); attach != "" {
			suffix := r.GetBodyValue("suffix", "")
			if reg := regexp.MustCompile("^[0-9A-Za-z]+$"); !reg.MatchString(suffix) {
				suffix = "txt"
			}
			content += fmt.Sprintf("\n    <p><a href=\"http://%s%s%s%s/%s/%d.%s\">%d.%s</a></p>",
				g.host, application[ca.Typee], querySeparator, ca.Datfile, r.ID, r.Stamp, suffix, r.Stamp, suffix)
		}
		permpath := path[1:]
		if ca.Typee == "thread" {
			permpath = fmt.Sprintf("%s/%s", path[1:], r.ID[:8])
		}
		rsss.append(permpath, title, rssTextFormat(r.GetBodyValue("name", "")), desc, content, ca.tags.getTagstrSlice(), r.Stamp, false)
	}
}

//printRSS reneders rss including newer records.
func printRSS(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	g := newGatewayCGI(w, r)
	if g == nil {
		return
	}
	rsss := newRss("UTF-8", "", g.m["logo"], "http://"+g.host, "",
		"http://"+g.host+gatewayURL+querySeparator+"rss", g.m["description"], xsl)
	cl := newCacheList()
	for _, ca := range cl.Caches {
		g.appendRSS(rsss, ca)
	}
	g.wr.Header().Set("Content-Type", "text/xml; charset=UTF-8")
	if rsss.len() != 0 {
		g.wr.Header().Set("Last-Modified", g.rfc822Time(rsss.Feeds[0].Date))
	}
	rsss.makeRSS1(g.wr)
}

//printMergedJS renders merged js with stamp.
func printMergedJS(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	g := newGatewayCGI(w, r)
	if g == nil {
		return
	}

	g.wr.Header().Set("Content-Type", "application/javascript; charset=UTF-8")
	g.wr.Header().Set("Last-Modified", g.rfc822Time(g.jc.GetLatest()))
	_, err := g.wr.Write([]byte(g.jc.getContent()))
	if err != nil {
		log.Println(err)
	}
}

//printMotd renders motd.
func printMotd(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	g := newGatewayCGI(w, r)
	if g == nil {
		return
	}

	g.wr.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	c, err := ioutil.ReadFile(motd)
	if err != nil {
		log.Println(err)
		return
	}
	_, err = g.wr.Write(c)
	if err != nil {
		log.Println(err)
	}
}

//printNew renders the page for making new thread.
func printNew(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	g := newGatewayCGI(w, r)
	if g == nil {
		return
	}

	g.header(g.m["new"], "", nil, true)
	g.printNewElementForm()
	g.footer(nil)
}

//printTitle renders list of newer thread in the disk for the top page
func printTitle(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	g := newGatewayCGI(w, r)
	if g == nil {
		return
	}
	if r.FormValue("cmd") != "" {
		g.jumpNewFile()
		return
	}
	cl := newCacheList()
	sort.Sort(sort.Reverse(sortByValidStamp{cl.Caches}))
	outputCachelist := make([]*cache, 0, cl.Len())
	for _, ca := range cl.Caches {
		if time.Now().Unix() <= ca.ValidStamp+topRecentRange {
			outputCachelist = append(outputCachelist, ca)
		}
	}
	g.header(g.m["logo"]+" - "+g.m["description"], "", nil, false)
	s := struct {
		Cachelist     []*cache
		Target        string
		Taglist       *UserTagList
		MchURL        string
		MchCategories []*mchCategory
		Message       message
		IsAdmin       bool
		IsFriend      bool
		GatewayCGI    string
		AdminCGI      string
		Types         string
		*GatewayLink
		ListItem
	}{
		outputCachelist,
		"changes",
		userTagList,
		g.mchURL(""),
		g.mchCategories(),
		g.m,
		g.isAdmin,
		g.isFriend,
		gatewayURL,
		adminURL,
		"thread",
		&GatewayLink{
			Message: g.m,
		},
		ListItem{
			IsAdmin: g.isAdmin,
			filter:  g.filter,
			tag:     g.tag,
			Message: g.m,
		},
	}
	renderTemplate("top", s, g.wr)
	g.printNewElementForm()
	g.footer(nil)
}

//printGatewayIndex renders list of new threads in the disk.
func printGatewayIndex(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	g := newGatewayCGI(w, r)
	if g == nil {
		return
	}
	g.printIndex(false)
}

//printIndexChanges renders list of new threads in the disk sorted by velocity.
func printIndexChanges(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	g := newGatewayCGI(w, r)
	if g == nil {
		return
	}
	g.printIndex(true)
}

//printRecent renders cache in recentlist.
func printRecent(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	g := newGatewayCGI(w, r)
	if g == nil {
		return
	}
	title := g.m["recent"]
	if g.filter != "" {
		title = fmt.Sprintf("%s : %s", g.m["recent"], g.filter)
	}
	g.header(title, "", nil, true)
	g.printParagraph("desc_recent")
	cl := recentList.makeRecentCachelist()
	g.printIndexList(cl.Caches, "recent", true, false)
}

//gatewayCGI is for gateway.cgi
type gatewayCGI struct {
	*cgi
}

//newGatewayCGI returns gatewayCGI obj with filter.tag value in form.
func newGatewayCGI(w http.ResponseWriter, r *http.Request) *gatewayCGI {
	c := newCGI(w, r)
	if c == nil {
		return nil
	}
	filter := r.FormValue("filter")
	tag := r.FormValue("tag")

	if filter != "" {
		c.filter = strings.ToLower(filter)
	} else {
		c.tag = strings.ToLower(tag)
	}

	if !c.checkVisitor() {
		c.print403()
		return nil
	}
	return &gatewayCGI{
		c,
	}
}

//makeOneRow makes one row of CSV depending on c.
func (g *gatewayCGI) makeOneRow(c string, ca *cache, p, title string) string {
	switch c {
	case "file":
		return ca.Datfile
	case "stamp":
		return strconv.FormatInt(ca.ValidStamp, 10)
	case "date":
		return time.Unix(ca.ValidStamp, 0).String()
	case "path":
		return p
	case "uri":
		if g.host != "" && p != "" {
			return "http://" + g.host + p
		}
	case "type":
		return ca.Typee
	case "title":
		return title
	case "records":
		return strconv.Itoa(ca.Len())
	case "size":
		return strconv.Itoa(ca.Size)
	case "tag":
		return ca.tags.string()
	case "sugtag":
		return ca.sugtags.string()
	}
	return ""
}

//renderCSV renders CSV including key string of caches in disk.
//key is specified in url query.
func (g *gatewayCGI) renderCSV(cl *cacheList) {
	g.wr.Header().Set("Content-Type", "text/comma-separated-values;charset=UTF-8")
	p := strings.Split(g.path, "/")
	if len(p) < 3 {
		g.print404(nil, "")
		return
	}
	cols := strings.Split(p[2], ",")
	cwr := csv.NewWriter(g.wr)
	for _, ca := range cl.Caches {
		title := fileDecode(ca.Datfile)
		var t, p string
		if hasString(types, ca.Typee) {
			t = ca.Typee
			p = application[t] + querySeparator + strEncode(title)
		}
		row := make([]string, len(cols))
		for i, c := range cols {
			row[i] = g.makeOneRow(c, ca, p, title)
		}
		err := cwr.Write(row)
		if err != nil {
			log.Println(err)
		}
	}
	cwr.Flush()
}

//printIndex renders threads in disk.
//id doChange threads are sorted by velocity.
func (g *gatewayCGI) printIndex(doChange bool) {
	str := "index"
	if doChange {
		str = "changes"
	}
	title := g.m["index"]
	if g.filter != "" {
		title = fmt.Sprintf("%s : %s", g.m["str"], g.filter)
	}
	g.header(title, "", nil, true)
	g.printParagraph("desc_"+str)
	cl := newCacheList()
	if doChange {
		sort.Sort(sort.Reverse(sortByVelocity{cl.Caches}))
	}
	g.printIndexList(cl.Caches, str, true, false)
}

//jumpNewFile renders 302 redirect to page for making new thread specified in url query
//"link"(thred name) "type"(thread) "tag" "search_new_file"("yes" or "no")
func (g *gatewayCGI) jumpNewFile() {
	link := g.req.FormValue("link")
	t := g.req.FormValue("type")
	switch {
	case link == "":
		g.header(g.m["null_title"], "", nil, true)
		g.footer(nil)
	case strings.ContainsAny(link, "/[]<>"):
		g.header(g.m["bad_title"], "", nil, true)
		g.footer(nil)
	case t == "":
		g.header(g.m["null_type"], "", nil, true)
		g.footer(nil)
	case hasString(types, t):
		tag := strEncode(g.req.FormValue("tag"))
		search := strEncode(g.req.FormValue("search_new_file"))
		g.print302(application[t] + querySeparator + strEncode(link) + "?tag=" + tag + "&search_new_file" + search)
	default:
		g.print404(nil, "")
	}
}

//rssHTMLFormat converts and returns plain string to html formats.
func (g *gatewayCGI) rssHTMLFormat(plain, appli, path string) string {
	title := strDecode(path)
	buf := g.htmlFormat(plain, appli, title, true)
	if buf != "" {
		buf = fmt.Sprintf("<p>%s</p>", buf)
	}
	return buf
}

//mchCategory represents category(tag) for each urls.
type mchCategory struct {
	URL  string
	Text string
}

//mchCategories returns slice of mchCategory whose tags are in tag.txt.
func (g *gatewayCGI) mchCategories() []*mchCategory {
	var categories []*mchCategory
	if !enable2ch {
		return categories
	}
	err := eachLine(runDir+"/tag.txt", func(line string, i int) error {
		tag := strings.TrimRight(line, "\r\n")
		catURL := g.mchURL(tag)
		categories = append(categories, &mchCategory{
			catURL,
			tag,
		})
		return nil
	})
	if err != nil {
		log.Println(err)
	}

	return categories
}

//mchURL returns url for 2ch interface.
func (g *gatewayCGI) mchURL(dat string) string {
	path := "/2ch/" + dat + "/subject.txt"
	if dat == "" {
		path = "/2ch/subject.txt"
	}
	if !enable2ch {
		return ""
	}
	if serverName != "" {
		return "//" + serverName + path
	}
	return fmt.Sprintf("//%s%s", g.host, path)
}
