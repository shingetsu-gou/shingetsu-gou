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

	"github.com/gorilla/handlers"
)

func gatewaySetup(s *http.ServeMux) {
	s.Handle("/gateway.cgi/motd", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newGatewayCGI(w, r); a != nil {
			a.printMotd()
		}
	})))
	s.Handle("/gateway.cgi/mergedjs", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newGatewayCGI(w, r); a != nil {
			a.printMergedJS()
		}
	})))
	s.Handle("/gateway.cgi/rss", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newGatewayCGI(w, r); a != nil {
			a.printRSS()
		}
	})))
	s.Handle("/gateway.cgi/recent_rss", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newGatewayCGI(w, r); a != nil {
			a.printRecentRSS()
		}
	})))
	s.Handle("/gateway.cgi/index", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newGatewayCGI(w, r); a != nil {
			a.printIndex(false)
		}
	})))
	s.Handle("/gateway.cgi/changes", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newGatewayCGI(w, r); a != nil {
			a.printIndex(true)
		}
	})))
	s.Handle("/gateway.cgi/recent", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newGatewayCGI(w, r); a != nil {
			a.printRecent()
		}
	})))
	s.Handle("/gateway.cgi/new", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newGatewayCGI(w, r); a != nil {
			a.printNew()
		}
	})))
	s.Handle("/gateway.cgi/thread", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newGatewayCGI(w, r); a != nil {
			a.printThread()
		}
	})))
	s.Handle("/gateway.cgi/", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newGatewayCGI(w, r); a != nil {
			a.printTitle()
		}
	})))
	s.Handle("/gateway.cgi/csv/index/", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newGatewayCGI(w, r); a != nil {
			a.printCSV(false)
		}
	})))
	s.Handle("/gateway.cgi/csv/changes/", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newGatewayCGI(w, r); a != nil {
			a.printCSV(true)
		}
	})))
	s.Handle("/gateway.cgi/csv/recent/", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newGatewayCGI(w, r); a != nil {
			a.printCSVRecent()
		}
	})))
	s.Handle("/", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newGatewayCGI(w, r); a != nil {
			a.printTitle()
		}
	})))
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
		c.strFilter = html.EscapeString(filter)
	} else {
		c.tag = strings.ToLower(tag)
		c.strTag = html.EscapeString(tag)
	}
	c.host = serverName
	if c.host == "" {
		c.host = r.Host
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
	p := strings.Split(g.req.URL.Path, "/")
	if len(p) < 3 {
		g.print404(nil, "")
		return
	}
	cols := strings.Split(p[2], ",")
	cwr := csv.NewWriter(g.wr)
	for _, ca := range cl.caches {
		title := fileDecode(ca.datfile)
		var t, p string
		if hasString(stringSlice(types), ca.typee) {
			t = ca.typee
			p = application[t] + querySeparator + strEncode(title)
		}
		row := make([]string, len(cols))
		for i, c := range cols {
			switch c {
			case "file":
				row[i] = ca.datfile
			case "stamp":
				row[i] = strconv.FormatInt(ca.validStamp, 10)
			case "date":
				row[i] = time.Unix(ca.validStamp, 0).String()
			case "path":
				row[i] = p
			case "uri":
				if g.host != "" && p != "" {
					row[i] = "http://" + g.host + p
				} else {
					row[i] = ""
				}
			case "type":
				row[i] = ca.typee
			case "title":
				row[i] = title
			case "records":
				row[i] = strconv.Itoa(ca.Len())
			case "size":
				row[i] = strconv.Itoa(ca.size)
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

func (g *gatewayCGI) printThread() {
	reg := regexp.MustCompile("^/(thread)/?([^/]*)$")
	m := reg.FindStringSubmatch(g.req.URL.Path)
	var uri string
	switch {
	case m == nil:
		g.printTitle()
		return
	case m[2] != "":
		uri = application["thread"] + querySeparator + strEncode(m[2])
	case g.req.URL.RawQuery != "":
		uri = application["thread"] + querySeparator + g.req.URL.RawQuery
	default:
		g.printTitle()
		return
	}
	g.print302(uri)
}

func (g *gatewayCGI) printCSV(doSort bool) {
	cl := newCacheList()
	if doSort {
		sort.Sort(sort.Reverse(sortByValidStamp{cl.caches}))
	}
	g.renderCSV(cl)
}

func (g *gatewayCGI) printCSVRecent() {
	if !g.isFriend && !g.isAdmin {
		g.print403("")
		return
	}
	cl := g.makeRecentCachelist()
	g.renderCSV(cl)
}
func (g *gatewayCGI) printRecentRSS() {
	rsss := newRss("UTF-8", "", fmt.Sprintf("%s - %s", g.m["recent"], g.m["logo"]),
		"http://"+g.host, "",
		"http://"+g.host+gatewayCgi+querySeparator+"recent_rss", g.m["description"], xsl)
	cl := g.makeRecentCachelist()
	for _, ca := range cl.caches {
		title := escape(fileDecode(ca.datfile))
		tags := make([]string, ca.tags.Len()+ca.sugtags.Len())
		for i, t := range ca.tags.tags {
			tags[i] = t.tagstr
		}
		for i, t := range ca.sugtags.tags {
			tags[i+ca.tags.Len()] = t.tagstr
		}
		if _, exist := application[ca.typee]; !exist {
			continue
		}
		rsss.append(application[ca.typee][1:]+querySeparator+strEncode(title),
			title, "", "", html.EscapeString(title), tags, ca.recentStamp, false)
	}
	g.wr.Header().Set("Content-Type", "text/xml; charset=UTF-8")
	k := rsss.keys()
	if len(k) != 0 {
		g.wr.Header().Set("Last-Modified", g.rfc822Time(rsss.items[k[0]].date))
	}
	rsss.makeRSS1(g.wr)

}

func (g *gatewayCGI) printRSS() {
	now := time.Now().Unix()
	rsss := newRss("UTF-8", "", g.m["logo"], "http://"+g.host, "",
		"http://"+g.host+gatewayCgi+querySeparator+"rss", g.m["description"], xsl)
	cl := newCacheList()
	for _, ca := range cl.caches {
		if ca.validStamp+int64(rssRange) >= now {
			title := escape(fileDecode(ca.datfile))
			path := application[ca.typee] + querySeparator + strEncode(title)
			for _, r := range ca.recs {
				if r.stamp+int64(rssRange) < now {
					continue
				}
				err := r.loadBody()
				if err != nil {
					log.Println(err)
				}

				desc := g.rssTextFormat(r.Get("body", ""))
				content := g.rssHTMLFormat(r.Get("body", ""), application[ca.typee], title)
				if attach := r.Get("attach", ""); attach != "" {
					suffix := r.Get("suffix", "")
					if reg := regexp.MustCompile("^[0-9A-Za-z]+$"); reg.MatchString(suffix) {
						suffix = "txt"
					}
					content += fmt.Sprintf("\n    <p><a href=\"http://%s%s%s%s/%s/%d.%s\">%d.%s</a></p>",
						g.host, application[ca.typee], querySeparator, ca.datfile, r.id, r.stamp, suffix, r.stamp, suffix)
				}
				permpath := path[1:]
				if ca.typee == "thread" {
					permpath = fmt.Sprintf("%s/%s", path[1:], r.id[:8])
				}
				rsss.append(permpath, title, g.rssTextFormat(r.Get("name", "")), desc, content, ca.tags.tags.toStringSlice(), r.stamp, false)
				r.free()
			}
		}
	}
	g.wr.Header().Set("Content-Type", "text/xml; charset=UTF-8")
	k := rsss.keys()
	if len(k) != 0 {
		g.wr.Header().Set("Last-Modified", g.rfc822Time(rsss.items[k[0]].date))
	}
	rsss.makeRSS1(g.wr)

}

func (g *gatewayCGI) printMergedJS() {
	g.wr.Header().Set("Content-Type", "application/javascript; charset=UTF-8")
	g.wr.Header().Set("Last-Modified", g.rfc822Time(g.jc.getLatest()))
	_, err := g.wr.Write([]byte(g.jc.getContent()))
	if err != nil {
		log.Println(err)
	}

}
func (g *gatewayCGI) printMotd() {
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
func (g *gatewayCGI) printNew() {
	g.header(g.m["new"], "", nil, true, nil)
	g.printNewElementForm()
	g.footer(nil)
}
func (g *gatewayCGI) printTitle() {
	cl := newCacheList()
	sort.Sort(sort.Reverse(sortByValidStamp{cl.caches}))
	outputCachelist := make([]*cache, 0, cl.Len())
	for _, ca := range cl.caches {
		if time.Now().Unix() <= ca.validStamp+int64(topRecentRange) {
			outputCachelist = append(outputCachelist, ca)
		}
	}
	g.header(g.m["logo"]+" - "+g.m["description"], "", nil, false, nil)
	s := struct {
		Cachelist     []*cache
		Target        string
		Taglist       *userTagList
		MchURL        string
		MchCategories []*mchCategory
	}{
		outputCachelist,
		"changes",
		newUserTagList(),
		g.mchURL(),
		g.mchCategories(),
	}
	renderTemplate("top", s, g.wr)
	g.printNewElementForm()
	g.footer(nil)
}

func (g *gatewayCGI) printIndex(doChange bool) {
	str := "index"
	if doChange {
		str = "changes"
	}
	title := g.m["index"]
	if g.strFilter != "" {
		title = fmt.Sprintf("%s : %s", g.m["str"], g.filter)
	}
	g.header(title, "", nil, true, nil)
	g.printParagraph(g.m["desc_"+str])
	cl := newCacheList()
	sort.Sort(sort.Reverse(sortByVelocity{cl.caches}))
	g.printIndexList(cl, str, false, false)
}

func (g *gatewayCGI) makeRecentCachelist() *cacheList {
	rl := newRecentList()
	sort.Sort(sort.Reverse(rl))
	var cl []*cache
	var check []string
	for _, rec := range rl.tiedlist {
		if !hasString(stringSlice(check), rec.datfile) {
			ca := newCache(rec.datfile, nil, nil)
			ca.recentStamp = rec.stamp
			cl = append(cl, ca)
			check = append(check, rec.datfile)
		}
	}
	return &cacheList{caches: cl}
}

func (g *gatewayCGI) printRecent() {
	title := g.m["recent"]
	if g.strFilter != "" {
		title = fmt.Sprintf("%s : %s", g.m["recent"], g.filter)
	}
	g.header(title, "", nil, true, nil)
	g.printParagraph(g.m["desc_recent"])
	cl := g.makeRecentCachelist()
	g.printIndexList(cl, "recent", false, false)
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
	case hasString(stringSlice(types), t):
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
