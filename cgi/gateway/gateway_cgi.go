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

package gateway

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

	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/cgi"
	"github.com/shingetsu-gou/shingetsu-gou/record"
	"github.com/shingetsu-gou/shingetsu-gou/tag"
	"github.com/shingetsu-gou/shingetsu-gou/tag/suggest"
	"github.com/shingetsu-gou/shingetsu-gou/tag/user"
	"github.com/shingetsu-gou/shingetsu-gou/thread"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

const xslURL = "/rss1.xsl"

//Setup setups handlers for gateway.cgi
func Setup(s *cgi.LoggingServeMux) {
	s.RegistCompressHandler(cfg.GatewayURL+"/motd", printMotd)
	s.RegistCompressHandler(cfg.GatewayURL+"/mergedjs", printMergedJS)
	s.RegistCompressHandler(cfg.GatewayURL+"/rss", printRSS)
	s.RegistCompressHandler(cfg.GatewayURL+"/recent_rss", printRecentRSS)
	s.RegistCompressHandler(cfg.GatewayURL+"/index", printGatewayIndex)
	s.RegistCompressHandler(cfg.GatewayURL+"/changes", printIndexChanges)
	s.RegistCompressHandler(cfg.GatewayURL+"/recent", printRecent)
	s.RegistCompressHandler(cfg.GatewayURL+"/new", printNew)
	s.RegistCompressHandler(cfg.GatewayURL+"/thread", printGatewayThread)
	s.RegistCompressHandler(cfg.GatewayURL+"/", PrintTitle)
	s.RegistCompressHandler(cfg.GatewayURL+"/csv/index/", printCSV)
	s.RegistCompressHandler(cfg.GatewayURL+"/csv/changes/", printCSVChanges)
	s.RegistCompressHandler(cfg.GatewayURL+"/csv/recent/", printCSVRecent)
}

//printGateway just redirects to correspoinding url using thread.cgi.
//or renders only title.
func printGatewayThread(w http.ResponseWriter, r *http.Request) {
	reg := regexp.MustCompile("^/gateway.cgi/(thread)/?([^/]*)$")
	m := reg.FindStringSubmatch(r.URL.Path)
	var uri string
	switch {
	case m == nil:
		PrintTitle(w, r)
		return
	case m[2] != "":
		uri = cfg.ThreadURL + "/" + util.StrEncode(m[2])
	case r.URL.RawQuery != "":
		uri = cfg.ThreadURL + "/" + r.URL.RawQuery
	default:
		PrintTitle(w, r)
		return
	}
	g, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	g.Print302(uri)
}

//printCSV renders csv of caches saved in disk.
func printCSV(w http.ResponseWriter, r *http.Request) {
	g, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	g.renderCSV(thread.AllCaches())
}

//printCSVChanges renders csv of caches which changes recently and are in disk(validstamp is newer).
func printCSVChanges(w http.ResponseWriter, r *http.Request) {
	g, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	all := thread.AllCaches()
	sort.Sort(sort.Reverse(thread.NewSortByStamp(all, false)))
	g.renderCSV(all)
}

//printCSVRecent renders csv of caches which are written recently(are updated remotely).
func printCSVRecent(w http.ResponseWriter, r *http.Request) {
	g, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	if !g.IsFriend() && !g.IsAdmin() {
		g.Print403()
		return
	}
	cl := thread.MakeRecentCachelist()
	g.renderCSV(cl)
}

//printRecentRSS renders rss of caches which are written recently(are updated remotely).
//including title,tags,last-modified.
func printRecentRSS(w http.ResponseWriter, r *http.Request) {
	g, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	rsss := cgi.NewRSS("UTF-8", "", fmt.Sprintf("%s - %s", g.M["recent"], g.M["logo"]),
		"http://"+g.Host(), "",
		"http://"+g.Host()+cfg.GatewayURL+"/"+"recent_rss", g.M["description"], xslURL)
	cl := thread.MakeRecentCachelist()
	for _, ca := range cl {
		title := util.Escape(util.FileDecode(ca.Datfile))
		tags := suggest.Get(ca.Datfile, nil)
		tags = append(tags, user.GetByThread(ca.Datfile)...)
		rsss.Append(cfg.ThreadURL[1:]+"/"+util.StrEncode(title),
			title, "", "", html.EscapeString(title), tags.GetTagstrSlice(),
			ca.RecentStamp(), false)
	}
	g.WR.Header().Set("Content-Type", "text/xml; charset=UTF-8")
	if rsss.Len() != 0 {
		g.WR.Header().Set("Last-Modified", g.RFC822Time(rsss.Feeds[0].Date))
	}
	rsss.MakeRSS1(g.WR)
}

//printRSS reneders rss including newer records.
func printRSS(w http.ResponseWriter, r *http.Request) {
	g, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	rsss := cgi.NewRSS("UTF-8", "", g.M["logo"], "http://"+g.Host(), "",
		"http://"+g.Host()+cfg.GatewayURL+"/"+"rss", g.M["description"], xslURL)
	for _, ca := range thread.AllCaches() {
		g.appendRSS(rsss, ca)
	}
	g.WR.Header().Set("Content-Type", "text/xml; charset=UTF-8")
	if rsss.Len() != 0 {
		g.WR.Header().Set("Last-Modified", g.RFC822Time(rsss.Feeds[0].Date))
	}
	rsss.MakeRSS1(g.WR)
}

//printMergedJS renders merged js with stamp.
func printMergedJS(w http.ResponseWriter, r *http.Request) {
	g, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}

	g.WR.Header().Set("Content-Type", "application/javascript; charset=UTF-8")
	g.WR.Header().Set("Last-Modified", g.RFC822Time(g.JC.GetLatest()))
	_, err = g.WR.Write([]byte(g.JC.GetContent()))
	if err != nil {
		log.Println(err)
	}
}

//printMotd renders motd.
func printMotd(w http.ResponseWriter, r *http.Request) {
	g, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}

	g.WR.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	c, err := ioutil.ReadFile(cfg.Motd())
	if err != nil {
		log.Println(err)
		return
	}
	_, err = g.WR.Write(c)
	if err != nil {
		log.Println(err)
	}
}

//printNew renders the page for making new thread.
func printNew(w http.ResponseWriter, r *http.Request) {
	g, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}

	g.Header(g.M["new"], "", nil, true)
	g.PrintNewElementForm()
	g.Footer(nil)
}

//PrintTitle renders list of newer thread in the disk for the top page
func PrintTitle(w http.ResponseWriter, r *http.Request) {
	g, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	if r.FormValue("cmd") != "" {
		g.jumpNewFile()
		return
	}
	all := thread.AllCaches()
	sort.Sort(sort.Reverse(thread.NewSortByStamp(all, false)))
	outputCachelist := make([]*thread.Cache, 0, thread.Len())
	for _, ca := range all {
		if time.Now().Unix() <= ca.Stamp()+cfg.TopRecentRange {
			outputCachelist = append(outputCachelist, ca)
		}
	}

	g.Header(g.M["logo"]+" - "+g.M["description"], "", nil, false)
	s := struct {
		Target        string
		Taglist       tag.Slice
		MchURL        string
		MchCategories []*mchCategory
		Types         string
		NoList        bool
		cgi.ListItem
		cgi.Defaults
	}{
		"changes",
		user.Get(),
		g.mchURL(""),
		g.mchCategories(),
		"thread",
		len(outputCachelist) == 0,
		*cgi.NewListItem(outputCachelist, false, "changes", false, g.Filter, g.Tag),
		*g.Defaults(),
	}
	cgi.RenderTemplate("top", s, g.WR)
	g.PrintNewElementForm()
	g.Footer(nil)
}

//printGatewayIndex renders list of new threads in the disk.
func printGatewayIndex(w http.ResponseWriter, r *http.Request) {
	g, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	g.printIndex(false)
}

//printIndexChanges renders list of new threads in the disk sorted by velocity.
func printIndexChanges(w http.ResponseWriter, r *http.Request) {
	g, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	g.printIndex(true)
}

//printRecent renders cache in recentlist.
func printRecent(w http.ResponseWriter, r *http.Request) {
	g, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	title := g.M["recent"]
	if g.Filter != "" {
		title = fmt.Sprintf("%s : %s", g.M["recent"], g.Filter)
	}
	g.Header(title, "", nil, true)
	fmt.Fprintf(g.WR, "<p>%s</p>", g.M["desc_recent"])
	cl := thread.MakeRecentCachelist()
	g.PrintIndexList(cl, "recent", true, false, g.Filter, g.Tag)
}

//gatewayCGI is for gateway.cgi
type gatewayCGI struct {
	*cgi.CGI
	Filter string
	Tag    string
}

//new returns gatewayCGI obj with filter.tag value in form.
func new(w http.ResponseWriter, r *http.Request) (*gatewayCGI, error) {
	c, err := cgi.NewCGI(w, r)
	if err != nil {
		return nil, err
	}
	a := gatewayCGI{
		CGI: c,
	}

	filter := r.FormValue("filter")
	tag := r.FormValue("tag")

	if filter != "" {
		a.Filter = strings.ToLower(filter)
	} else {
		a.Tag = strings.ToLower(tag)
	}

	if !a.CheckVisitor() {
		a.Print403()
		return nil, errors.New("permission denied")
	}
	return &a, nil
}

//appendRSS appends cache ca to rss with contents,url to records,stamp,attached file.
func (g *gatewayCGI) appendRSS(rsss *cgi.RSS, ca *thread.Cache) {
	now := time.Now().Unix()
	if ca.Stamp()+cfg.RSSRange < now {
		return
	}
	title := util.Escape(util.FileDecode(ca.Datfile))
	path := cfg.ThreadURL + "/" + util.StrEncode(title)
	recs := ca.LoadRecords(record.Alive)
	for _, r := range recs {
		if r.Stamp+cfg.RSSRange < now {
			continue
		}
		if err := r.Load(); err != nil {
			log.Println(err)
			continue
		}
		desc := cgi.RSSTextFormat(r.GetBodyValue("body", ""))
		content := g.rssHTMLFormat(r.GetBodyValue("body", ""), cfg.ThreadURL, title)
		if attach := r.GetBodyValue("attach", ""); attach != "" {
			suffix := r.GetBodyValue("suffix", "")
			if reg := regexp.MustCompile("^[0-9A-Za-z]+$"); !reg.MatchString(suffix) {
				suffix = cfg.SuffixTXT
			}
			content += fmt.Sprintf("\n    <p><a href=\"http://%s%s%s%s/%s/%d.%s\">%d.%s</a></p>",
				g.Host(), cfg.ThreadURL, "/", ca.Datfile, r.ID, r.Stamp, suffix, r.Stamp, suffix)
		}
		permpath := fmt.Sprintf("%s/%s", path[1:], r.ID[:8])
		rsss.Append(permpath, title, cgi.RSSTextFormat(r.GetBodyValue("name", "")), desc, content, user.GetStrings(ca.Datfile), r.Stamp, false)
	}
}

//makeOneRow makes one row of CSV depending on c.
func (g *gatewayCGI) makeOneRow(c string, ca *thread.Cache, p, title string) string {
	switch c {
	case "file":
		return ca.Datfile
	case "stamp":
		return strconv.FormatInt(ca.Stamp(), 10)
	case "date":
		return time.Unix(ca.Stamp(), 0).String()
	case "path":
		return p
	case "uri":
		if g.Host() != "" && p != "" {
			return "http://" + g.Host() + p
		}
	case "type":
		return "thread"
	case "title":
		return title
	case "records":
		return strconv.Itoa(ca.Len(record.Alive))
	case "size":
		return strconv.FormatInt(ca.Size(), 10)
	case "tag":
		return user.String(ca.Datfile)
	case "sugtag":
		return suggest.String(ca.Datfile)
	}
	return ""
}

//renderCSV renders CSV including key string of caches in disk.
//key is specified in url query.
func (g *gatewayCGI) renderCSV(cl thread.Caches) {
	g.WR.Header().Set("Content-Type", "text/comma-separated-values;charset=UTF-8")
	p := strings.Split(g.Path(), "/")
	if len(p) < 3 {
		g.Print404(nil, "")
		return
	}
	cols := strings.Split(p[2], ",")
	cwr := csv.NewWriter(g.WR)
	for _, ca := range cl {
		title := util.FileDecode(ca.Datfile)
		p := cfg.ThreadURL + "/" + util.StrEncode(title)
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
	title := g.M[str]
	if doChange {
		str = "change"
		title = g.M["changes"]
	}

	if g.Filter != "" {
		title = fmt.Sprintf("%s : %s", g.M["string"], g.Filter)
	}
	g.Header(title, "", nil, true)
	fmt.Fprintf(g.WR, "<p>%s</p>", g.M["desc_"+str])
	cl := thread.AllCaches()
	if doChange {
		sort.Sort(sort.Reverse(thread.NewSortByStamp(cl, false)))
	} else {
		sort.Sort(sort.Reverse(thread.NewSortByVelocity(cl)))
	}
	g.PrintIndexList(cl, str, true, false, g.Filter, g.Tag)
}

//jumpNewFile renders 302 redirect to page for making new thread specified in url query
//"link"(thred name) "type"(thread) "tag" "search_new_file"("yes" or "no")
func (g *gatewayCGI) jumpNewFile() {
	link := g.Req.FormValue("link")
	t := g.Req.FormValue("type")
	switch {
	case link == "":
		g.Header(g.M["null_title"], "", nil, true)
		g.Footer(nil)
	case strings.ContainsAny(link, "/[]<>"):
		g.Header(g.M["bad_title"], "", nil, true)
		g.Footer(nil)
	case t == "":
		g.Header(g.M["null_type"], "", nil, true)
		g.Footer(nil)
	case t == "thread":
		tag := util.StrEncode(g.Req.FormValue("tag"))
		search := util.StrEncode(g.Req.FormValue("search_new_file"))
		g.Print302(cfg.ThreadURL + "/" + util.StrEncode(link) + "?tag=" + tag + "&search_new_file" + search)
	default:
		g.Print404(nil, "")
	}
}

//rssHTMLFormat converts and returns plain string to html formats.
func (g *gatewayCGI) rssHTMLFormat(plain, appli, path string) string {
	title := util.StrDecode(path)
	buf := g.HTMLFormat(plain, appli, title, true)
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
	if !cfg.Enable2ch {
		return categories
	}
	for _, t := range user.Get() {
		tag := t.Tagstr
		catURL := g.mchURL(tag)
		categories = append(categories, &mchCategory{
			catURL,
			tag,
		})
	}

	return categories
}

//mchURL returns url for 2ch interface.
func (g *gatewayCGI) mchURL(dat string) string {
	path := "/2ch/" + strings.ToUpper(hex.EncodeToString([]byte(dat))) + "/subject.txt"
	if dat == "" {
		path = "/2ch/subject.txt"
	}
	if !cfg.Enable2ch {
		return ""
	}
	if cfg.ServerName != "" {
		return "//" + cfg.ServerName + path
	}
	return fmt.Sprintf("//%s%s", g.Host(), path)
}
