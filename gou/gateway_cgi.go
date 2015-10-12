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

func gatewaySetup(s *http.ServeMux) {
	registCompressHandler(s, "/gateway.cgi/motd", printMotd)
	registCompressHandler(s, "/gateway.cgi/mergedjs", printMergedJS)
	registCompressHandler(s, "/gateway.cgi/rss", printRSS)
	registCompressHandler(s, "/gateway.cgi/recent_rss", printRecentRSS)
	registCompressHandler(s, "/gateway.cgi/index", printGatewayIndex)
	registCompressHandler(s, "/gateway.cgi/changes", printIndexChanges)
	registCompressHandler(s, "/gateway.cgi/recent", printRecent)
	registCompressHandler(s, "/gateway.cgi/new", printNew)
	registCompressHandler(s, "/gateway.cgi/thread", printGatewayThread)
	registCompressHandler(s, "/gateway.cgi/", printTitle)
	registCompressHandler(s, "/gateway.cgi/csv/index/", printCSV)
	registCompressHandler(s, "/gateway.cgi/csv/changes/", printCSVChanges)
	registCompressHandler(s, "/gateway.cgi/csv/recent/", printCSVRecent)
	registCompressHandler(s, "/", printTitle)
}

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
		g.print403("")
		return
	}
	cl := g.makeRecentCachelist()
	g.renderCSV(cl)
}

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
	cl := g.makeRecentCachelist()
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
	k := rsss.keys()
	if len(k) != 0 {
		g.wr.Header().Set("Last-Modified", g.rfc822Time(rsss.Feeds[k[0]].Date))
	}
	rsss.makeRSS1(g.wr)

}

func printRSS(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	g := newGatewayCGI(w, r)
	if g == nil {
		return
	}
	now := time.Now().Unix()
	rsss := newRss("UTF-8", "", g.m["logo"], "http://"+g.host, "",
		"http://"+g.host+gatewayURL+querySeparator+"rss", g.m["description"], xsl)
	cl := newCacheList()
	for _, ca := range cl.Caches {
		if ca.ValidStamp+rssRange < now {
			continue
		}
		title := escape(fileDecode(ca.Datfile))
		path := application[ca.Typee] + querySeparator + strEncode(title)
		for _, r := range ca.recs {
			if r.Stamp+rssRange < now {
				continue
			}
			if err := r.loadBody(); err != nil {
				log.Println(err)
			}

			desc := g.rssTextFormat(r.GetBodyValue("body", ""))
			content := g.rssHTMLFormat(r.GetBodyValue("body", ""), application[ca.Typee], title)
			if attach := r.GetBodyValue("attach", ""); attach != "" {
				suffix := r.GetBodyValue("suffix", "")
				if reg := regexp.MustCompile("^[0-9A-Za-z]+$"); reg.MatchString(suffix) {
					suffix = "txt"
				}
				content += fmt.Sprintf("\n    <p><a href=\"http://%s%s%s%s/%s/%d.%s\">%d.%s</a></p>",
					g.host, application[ca.Typee], querySeparator, ca.Datfile, r.ID, r.Stamp, suffix, r.Stamp, suffix)
			}
			permpath := path[1:]
			if ca.Typee == "thread" {
				permpath = fmt.Sprintf("%s/%s", path[1:], r.ID[:8])
			}
			rsss.append(permpath, title, g.rssTextFormat(r.GetBodyValue("name", "")), desc, content, ca.tags.getTagstrSlice(), r.Stamp, false)
		}
	}
	g.wr.Header().Set("Content-Type", "text/xml; charset=UTF-8")
	if k := rsss.keys(); len(k) != 0 {
		g.wr.Header().Set("Last-Modified", g.rfc822Time(rsss.Feeds[k[0]].Date))
	}
	rsss.makeRSS1(g.wr)
}

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
func printNew(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	g := newGatewayCGI(w, r)
	if g == nil {
		return
	}

	g.header(g.m["new"], "", nil, true, nil)
	g.printNewElementForm()
	g.footer(nil)
}
func printTitle(w http.ResponseWriter, r *http.Request) {
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
	outputCachelist := make([]*cache, 0, cl.Len())
	for _, ca := range cl.Caches {
		if time.Now().Unix() <= ca.ValidStamp+topRecentRange {
			outputCachelist = append(outputCachelist, ca)
		}
	}
	g.header(g.m["logo"]+" - "+g.m["description"], "", nil, false, nil)
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
		GatewayLink
		ListItem
	}{
		outputCachelist,
		"changes",
		userTagList,
		g.mchURL(),
		g.mchCategories(),
		g.m,
		g.isAdmin,
		g.isFriend,
		gatewayURL,
		adminURL,
		"thread",
		GatewayLink{
			Message: g.m,
		},
		ListItem{
			IsAdmin: g.isAdmin,
			filter:  g.filter,
			tag:     g.tag,
		},
	}
	renderTemplate("top", s, g.wr)
	g.printNewElementForm()
	g.footer(nil)
}

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
	g.header(title, "", nil, true, nil)
	g.printParagraph(g.m["desc_recent"])
	cl := g.makeRecentCachelist()
	g.printIndexList(cl.Caches, "recent", false, false)
}

type gatewayCGI struct {
	*cgi
}

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
		c.print403("")
		return nil
	}
	return &gatewayCGI{
		c,
	}
}

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
			switch c {
			case "file":
				row[i] = ca.Datfile
			case "stamp":
				row[i] = strconv.FormatInt(ca.ValidStamp, 10)
			case "date":
				row[i] = time.Unix(ca.ValidStamp, 0).String()
			case "path":
				row[i] = p
			case "uri":
				if g.host != "" && p != "" {
					row[i] = "http://" + g.host + p
				} else {
					row[i] = ""
				}
			case "type":
				row[i] = ca.Typee
			case "title":
				row[i] = title
			case "records":
				row[i] = strconv.Itoa(ca.Len())
			case "size":
				row[i] = strconv.Itoa(ca.Size)
			case "tag":
				row[i] = ca.tags.string()
			case "sugtag":
				row[i] = ca.sugtags.string()
			default:
			}
		}
		err := cwr.Write(row)
		if err != nil {
			log.Println(err)
		}

	}
	cwr.Flush()
}

func (g *gatewayCGI) printIndex(doChange bool) {
	str := "index"
	if doChange {
		str = "changes"
	}
	title := g.m["index"]
	if g.filter != "" {
		title = fmt.Sprintf("%s : %s", g.m["str"], g.filter)
	}
	g.header(title, "", nil, true, nil)
	g.printParagraph(g.m["desc_"+str])
	cl := newCacheList()
	if doChange {
		sort.Sort(sort.Reverse(sortByVelocity{cl.Caches}))
	}
	g.printIndexList(cl.Caches, str, false, false)
}

func (g *gatewayCGI) makeRecentCachelist() *cacheList {
	sort.Sort(sort.Reverse(recentList))
	var cl []*cache
	var check []string
	for _, rec := range recentList.infos {
		if !hasString(check, rec.datfile) {
			ca := newCache(rec.datfile)
			ca.RecentStamp = rec.stamp
			cl = append(cl, ca)
			check = append(check, rec.datfile)
		}
	}
	return &cacheList{Caches: cl}
}

func (g *gatewayCGI) jumpNewFile() {
	link := g.req.FormValue("link")
	t := g.req.FormValue("type")
	switch {
	case link == "":
		g.header(g.m["null_title"], "", nil, true, nil)
		g.footer(nil)
	case strings.ContainsAny(link, "/[]<>"):
		g.header(g.m["bad_title"], "", nil, true, nil)
		g.footer(nil)
	case t == "":
		g.header(g.m["null_type"], "", nil, true, nil)
		g.footer(nil)
	case hasString(types, t):
		tag := strEncode(g.req.FormValue("tag"))
		search := strEncode(g.req.FormValue("search_new_file"))
		g.print302(application[t] + querySeparator + strEncode(link) + "?tag=" + tag + "&search_new_filter" + search)
	default:
		g.print404(nil, "")
	}
}
func (g *gatewayCGI) rssTextFormat(plain string) string {
	buf := strings.Replace(plain, "<br>", " ", -1)
	buf = strings.Replace(buf, "&", "&amp;", -1)
	reg := regexp.MustCompile("&amp;(#\\d+|lt|gt|amp);")
	buf = reg.ReplaceAllString(buf, "&\\1")
	buf = strings.Replace(buf, "<", "&lt;", -1)
	buf = strings.Replace(buf, ">", "&gt;", -1)
	buf = strings.Replace(buf, "\r", "", -1)
	buf = strings.Replace(buf, "\n", "", -1)
	return buf
}

func (g *gatewayCGI) rssHTMLFormat(plain, appli, path string) string {
	title := strDecode(path)
	buf := g.htmlFormat(plain, appli, title, true)
	if buf != "" {
		buf = fmt.Sprintf("<p>%s</p>", buf)
	}
	return buf
}

type mchCategory struct {
	URL  string
	Text string
}

func (g *gatewayCGI) mchCategories() []*mchCategory {
	var categories []*mchCategory
	if !enable2ch {
		return categories
	}
	mchURL := g.mchURL()
	err := eachLine(runDir+"/tag.txt", func(line string, i int) error {
		tag := strings.TrimRight(line, "\r\n")
		catURL := strings.Replace(mchURL, "2ch", fileEncode("2ch", tag), -1)
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

func (g *gatewayCGI) mchURL() string {
	path := "/2ch/subject.txt"
	if !enable2ch {
		return ""
	}
	if serverName != "" {
		return "//" + serverName + path
	}
	reg := regexp.MustCompile(":\\d+")
	host := reg.ReplaceAllString(g.req.Host, "")
	if host == "" {
		return ""
	}
	return fmt.Sprintf("//%s:%d%s", host, datPort, path)
}
