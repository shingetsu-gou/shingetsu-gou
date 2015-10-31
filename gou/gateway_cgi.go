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
	"encoding/hex"
	"errors"
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

//GatewayURL is the url to gateway.cgi
const GatewayURL = "/gateway.cgi"
const xslURL = "/rss1.xsl"

//gatewaySetup setups handlers for gateway.cgi
func gatewaySetup(s *loggingServeMux, cfg *Config, gl *Global) {
	s.registCompressHandler(GatewayURL+"/motd", printMotd(cfg, gl))
	s.registCompressHandler(GatewayURL+"/mergedjs", printMergedJS(cfg, gl))
	s.registCompressHandler(GatewayURL+"/rss", printRSS(cfg, gl))
	s.registCompressHandler(GatewayURL+"/recent_rss", printRecentRSS(cfg, gl))
	s.registCompressHandler(GatewayURL+"/index", printGatewayIndex(cfg, gl))
	s.registCompressHandler(GatewayURL+"/changes", printIndexChanges(cfg, gl))
	s.registCompressHandler(GatewayURL+"/recent", printRecent(cfg, gl))
	s.registCompressHandler(GatewayURL+"/new", printNew(cfg, gl))
	s.registCompressHandler(GatewayURL+"/thread", printGatewayThread(cfg, gl))
	s.registCompressHandler(GatewayURL+"/", printTitle(cfg, gl))
	s.registCompressHandler(GatewayURL+"/csv/index/", printCSV(cfg, gl))
	s.registCompressHandler(GatewayURL+"/csv/changes/", printCSVChanges(cfg, gl))
	s.registCompressHandler(GatewayURL+"/csv/recent/", printCSVRecent(cfg, gl))
}

//printGateway just redirects to correspoinding url using thread.cgi.
//or renders only title.
func printGatewayThread(cfg *Config, gl *Global) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		reg := regexp.MustCompile("^/gateway.cgi/(thread)/?([^/]*)$")
		m := reg.FindStringSubmatch(r.URL.Path)
		var uri string
		switch {
		case m == nil:
			printTitle(cfg, gl)(w, r)
			return
		case m[2] != "":
			uri = ThreadURL + "/" + strEncode(m[2])
		case r.URL.RawQuery != "":
			uri = ThreadURL + "/" + r.URL.RawQuery
		default:
			printTitle(cfg, gl)(w, r)
			return
		}
		g, err := newGatewayCGI(w, r, cfg, gl)
		if err != nil {
			log.Println(err)
			return
		}
		g.print302(uri)
	}
}

//printCSV renders csv of caches saved in disk.
func printCSV(cfg *Config, gl *Global) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		g, err := newGatewayCGI(w, r, cfg, gl)
		defer g.close()
		if err != nil {
			log.Println(err)
			return
		}
		cl := newCacheList(cfg, gl)
		g.renderCSV(cl.Caches)
	}
}

//printCSVChanges renders csv of caches which changes recently and are in disk(validstamp is newer).
func printCSVChanges(cfg *Config, gl *Global) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		g, err := newGatewayCGI(w, r, cfg, gl)
		defer g.close()
		if err != nil {
			log.Println(err)
			return
		}
		cl := newCacheList(cfg, gl)
		sort.Sort(sort.Reverse(newSortByStamp(cl.Caches)))
		g.renderCSV(cl.Caches)
	}
}

//printCSVRecent renders csv of caches which are written recently(are updated remotely).
func printCSVRecent(cfg *Config, gl *Global) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		g, err := newGatewayCGI(w, r, cfg, gl)
		defer g.close()
		if err != nil {
			log.Println(err)
			return
		}
		if !g.isFriend() && !g.isAdmin() {
			g.print403()
			return
		}
		cl := gl.RecentList.makeRecentCachelist(cfg, gl)
		g.renderCSV(cl)
	}
}

//printRecentRSS renders rss of caches which are written recently(are updated remotely).
//including title,tags,last-modified.
func printRecentRSS(cfg *Config, gl *Global) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		g, err := newGatewayCGI(w, r, cfg, gl)
		defer g.close()

		if err != nil {
			log.Println(err)
			return
		}
		rsss := newRss("UTF-8", "", fmt.Sprintf("%s - %s", g.m["recent"], g.m["logo"]),
			"http://"+g.host(), "",
			"http://"+g.host()+GatewayURL+"/"+"recent_rss", g.m["description"], xslURL)
		cl := gl.RecentList.makeRecentCachelist(cfg, gl)
		for _, ca := range cl {
			title := escape(fileDecode(ca.Datfile))
			tags := gl.SuggestedTagTable.get(ca.Datfile, nil)
			tags = append(tags, ca.getTags()...)
			rsss.append(ThreadURL[1:]+"/"+strEncode(title),
				title, "", "", html.EscapeString(title), tags.getTagstrSlice(),
				ca.recentStamp(), false)
		}
		g.wr.Header().Set("Content-Type", "text/xml; charset=UTF-8")
		if rsss.Len() != 0 {
			g.wr.Header().Set("Last-Modified", g.rfc822Time(rsss.Feeds[0].Date))
		}
		rsss.makeRSS1(g.wr)
	}
}

//printRSS reneders rss including newer records.
func printRSS(cfg *Config, gl *Global) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		g, err := newGatewayCGI(w, r, cfg, gl)
		defer g.close()
		if err != nil {
			log.Println(err)
			return
		}
		rsss := newRss("UTF-8", "", g.m["logo"], "http://"+g.host(), "",
			"http://"+g.host()+GatewayURL+"/"+"rss", g.m["description"], xslURL)
		cl := newCacheList(cfg, gl)
		for _, ca := range cl.Caches {
			g.appendRSS(rsss, ca)
		}
		g.wr.Header().Set("Content-Type", "text/xml; charset=UTF-8")
		if rsss.Len() != 0 {
			g.wr.Header().Set("Last-Modified", g.rfc822Time(rsss.Feeds[0].Date))
		}
		rsss.makeRSS1(g.wr)
	}
}

//printMergedJS renders merged js with stamp.
func printMergedJS(cfg *Config, gl *Global) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		g, err := newGatewayCGI(w, r, cfg, gl)
		defer g.close()
		if err != nil {
			log.Println(err)
			return
		}

		g.wr.Header().Set("Content-Type", "application/javascript; charset=UTF-8")
		g.wr.Header().Set("Last-Modified", g.rfc822Time(g.jc.GetLatest()))
		_, err = g.wr.Write([]byte(g.jc.getContent()))
		if err != nil {
			log.Println(err)
		}
	}
}

//printMotd renders motd.
func printMotd(cfg *Config, gl *Global) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		g, err := newGatewayCGI(w, r, cfg, gl)
		defer g.close()
		if err != nil {
			log.Println(err)
			return
		}

		g.wr.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		c, err := ioutil.ReadFile(cfg.Motd())
		if err != nil {
			log.Println(err)
			return
		}
		_, err = g.wr.Write(c)
		if err != nil {
			log.Println(err)
		}
	}
}

//printNew renders the page for making new thread.
func printNew(cfg *Config, gl *Global) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		g, err := newGatewayCGI(w, r, cfg, gl)
		defer g.close()
		if err != nil {
			log.Println(err)
			return
		}

		g.header(g.m["new"], "", nil, true)
		g.printNewElementForm()
		g.footer(nil)
	}
}

//printTitle renders list of newer thread in the disk for the top page
func printTitle(cfg *Config, gl *Global) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		g, err := newGatewayCGI(w, r, cfg, gl)
		defer g.close()
		if err != nil {
			log.Println(err)
			return
		}
		if r.FormValue("cmd") != "" {
			g.jumpNewFile()
			return
		}
		cl := newCacheList(cfg, gl)
		sort.Sort(sort.Reverse(newSortByStamp(cl.Caches)))
		outputCachelist := make([]*cache, 0, cl.Len())
		for _, ca := range cl.Caches {
			if time.Now().Unix() <= ca.readInfo().stamp+g.TopRecentRange {
				outputCachelist = append(outputCachelist, ca)
			}
		}
		g.header(g.m["logo"]+" - "+g.m["description"], "", nil, false)
		s := struct {
			Cachelist     []*cache
			Target        string
			Taglist       tagslice
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
			g.UserTag.get(),
			g.mchURL(""),
			g.mchCategories(),
			g.m,
			g.isAdmin(),
			g.isFriend(),
			GatewayURL,
			AdminURL,
			"thread",
			&GatewayLink{
				htemplate: g.Htemplate,
				Message:   g.m,
			},
			ListItem{
				htemplate:         g.Htemplate,
				suggestedTagTable: g.SuggestedTagTable,
				IsAdmin:           g.isAdmin(),
				filter:            g.filter,
				tag:               g.tag,
				Message:           g.m,
			},
		}
		g.Htemplate.renderTemplate("top", s, g.wr)
		g.printNewElementForm()
		g.footer(nil)
	}
}

//printGatewayIndex renders list of new threads in the disk.
func printGatewayIndex(cfg *Config, gl *Global) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		g, err := newGatewayCGI(w, r, cfg, gl)
		defer g.close()
		if err != nil {
			log.Println(err)
			return
		}
		g.printIndex(false)
	}
}

//printIndexChanges renders list of new threads in the disk sorted by velocity.
func printIndexChanges(cfg *Config, gl *Global) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		g, err := newGatewayCGI(w, r, cfg, gl)
		defer g.close()
		if err != nil {
			log.Println(err)
			return
		}
		g.printIndex(true)
	}
}

//printRecent renders cache in recentlist.
func printRecent(cfg *Config, gl *Global) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		g, err := newGatewayCGI(w, r, cfg, gl)
		defer g.close()
		if err != nil {
			log.Println(err)
			return
		}
		title := g.m["recent"]
		if g.filter != "" {
			title = fmt.Sprintf("%s : %s", g.m["recent"], g.filter)
		}
		g.header(title, "", nil, true)
		g.printParagraph("desc_recent")
		cl := g.RecentList.makeRecentCachelist(cfg, gl)
		g.printIndexList(cl, "recent", true, false)
	}
}

//gatewayCGI is for gateway.cgi
type gatewayCGI struct {
	*cgi
}

//newGatewayCGI returns gatewayCGI obj with filter.tag value in form.
func newGatewayCGI(w http.ResponseWriter, r *http.Request, cfg *Config, gl *Global) (gatewayCGI, error) {
	a := gatewayCGI{
		cgi: newCGI(w, r, cfg, gl),
	}
	if a.cgi == nil {
		return a, errors.New("cannot make cgi")
	}
	filter := r.FormValue("filter")
	tag := r.FormValue("tag")

	if filter != "" {
		a.filter = strings.ToLower(filter)
	} else {
		a.tag = strings.ToLower(tag)
	}

	if !a.checkVisitor() {
		a.print403()
		return a, errors.New("permission denied")
	}
	return a, nil
}

//appendRSS appends cache ca to rss with contents,url to records,stamp,attached file.
func (g *gatewayCGI) appendRSS(rsss *RSS, ca *cache) {
	now := time.Now().Unix()
	if ca.readInfo().stamp+g.RSSRange < now {
		return
	}
	title := escape(fileDecode(ca.Datfile))
	path := ThreadURL + "/" + strEncode(title)
	recs := ca.loadRecords()
	for _, r := range recs {
		if r.Stamp+g.RSSRange < now {
			continue
		}
		if err := r.load(); err != nil {
			log.Println(err)
			continue
		}
		desc := rssTextFormat(r.GetBodyValue("body", ""))
		content := g.rssHTMLFormat(r.GetBodyValue("body", ""), ThreadURL, title)
		if attach := r.GetBodyValue("attach", ""); attach != "" {
			suffix := r.GetBodyValue("suffix", "")
			if reg := regexp.MustCompile("^[0-9A-Za-z]+$"); !reg.MatchString(suffix) {
				suffix = "txt"
			}
			content += fmt.Sprintf("\n    <p><a href=\"http://%s%s%s%s/%s/%d.%s\">%d.%s</a></p>",
				g.host(), ThreadURL, "/", ca.Datfile, r.ID, r.Stamp, suffix, r.Stamp, suffix)
		}
		permpath := path[1:]
		permpath = fmt.Sprintf("%s/%s", path[1:], r.ID[:8])
		rsss.append(permpath, title, rssTextFormat(r.GetBodyValue("name", "")), desc, content, ca.getTagstrSlice(), r.Stamp, false)
	}
}

//makeOneRow makes one row of CSV depending on c.
func (g *gatewayCGI) makeOneRow(c string, ca *cache, p, title string) string {
	i := ca.readInfo()
	switch c {
	case "file":
		return ca.Datfile
	case "stamp":
		return strconv.FormatInt(i.stamp, 10)
	case "date":
		return time.Unix(i.stamp, 0).String()
	case "path":
		return p
	case "uri":
		if g.host() != "" && p != "" {
			return "http://" + g.host() + p
		}
	case "type":
		return "thread"
	case "title":
		return title
	case "records":
		return strconv.Itoa(i.len)
	case "size":
		return strconv.FormatInt(i.size, 10)
	case "tag":
		return ca.tagString()
	case "sugtag":
		return g.SuggestedTagTable.string(ca.Datfile)
	}
	return ""
}

//renderCSV renders CSV including key string of caches in disk.
//key is specified in url query.
func (g *gatewayCGI) renderCSV(cl caches) {
	g.wr.Header().Set("Content-Type", "text/comma-separated-values;charset=UTF-8")
	p := strings.Split(g.path(), "/")
	if len(p) < 3 {
		g.print404(nil, "")
		return
	}
	cols := strings.Split(p[2], ",")
	cwr := csv.NewWriter(g.wr)
	for _, ca := range cl {
		title := fileDecode(ca.Datfile)
		p := ThreadURL + "/" + strEncode(title)
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
	g.printParagraph("desc_" + str)
	cl := newCacheList(g.Config, g.Global)
	if doChange {
		sort.Sort(sort.Reverse(newSortByVelocity(cl.Caches)))
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
	case t == "thread":
		tag := strEncode(g.req.FormValue("tag"))
		search := strEncode(g.req.FormValue("search_new_file"))
		g.print302(ThreadURL + "/" + strEncode(link) + "?tag=" + tag + "&search_new_file" + search)
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
	if !g.Enable2ch {
		return categories
	}
	err := eachLine(g.RunDir+"/tag.txt", func(line string, i int) error {
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
	path := "/2ch/" + strings.ToUpper(hex.EncodeToString([]byte(dat))) + "/subject.txt"
	if dat == "" {
		path = "/2ch/subject.txt"
	}
	if !g.Enable2ch {
		return ""
	}
	if g.ServerName != "" {
		return "//" + g.ServerName + path
	}
	return fmt.Sprintf("//%s%s", g.host(), path)
}
