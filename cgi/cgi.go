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
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/russross/blackfriday"
	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/tag"
	"github.com/shingetsu-gou/shingetsu-gou/tag/suggest"
	"github.com/shingetsu-gou/shingetsu-gou/tag/user"
	"github.com/shingetsu-gou/shingetsu-gou/thread"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//CacheInfo is infos of a cache.
type CacheInfo struct {
	Cache   *thread.Cache
	Tags    tag.Slice
	Sugtags []*tag.Tag
	Title   string
}

//ListItem is for list_item.txt
type ListItem struct {
	Caches  []*CacheInfo
	StrOpts string
	Remove  bool
}

//checkCache checks cache ca has specified tag and datfile doesn't contains filterd string.
func checkCache(ca *thread.Cache, target, filter, tag string) (string, bool) {
	x := util.FileDecode(ca.Datfile)
	if x == "" {
		return "", false
	}
	if filter != "" && !strings.Contains(strings.ToLower(x), filter) {
		return "", false
	}
	if tag != "" {
		switch {
		case user.Has(ca.Datfile, strings.ToLower(tag)):
		case target == "recent" && suggest.HasTagstr(ca.Datfile, strings.ToLower(tag)):
		default:
			return "", false
		}
	}
	return x, true
}

//NewListItem returns ListItem struct from caches.
func NewListItem(caches []*thread.Cache, remove bool, target string, search bool, filter, tag string) *ListItem {
	li := &ListItem{Remove: remove}
	li.Caches = make([]*CacheInfo, 0, len(caches))
	if search {
		li.StrOpts = "?search_new_file=yes"
	}
	for _, ca := range caches {
		x, ok := checkCache(ca, target, filter, tag)
		if !ok {
			continue
		}
		ci := &CacheInfo{Cache: ca}
		li.Caches = append(li.Caches, ci)
		ci.Title = util.EscapeSpace(x)
		if target == "recent" {
			strTags := make([]string, user.Len(ca.Datfile))
			for i, v := range user.GetByThread(ca.Datfile) {
				strTags[i] = strings.ToLower(v.Tagstr)
			}
			for _, st := range suggest.Get(ca.Datfile, nil) {
				if !util.HasString(strTags, strings.ToLower(st.Tagstr)) {
					ci.Sugtags = append(ci.Sugtags, st)
				}
			}
		}
		ci.Tags = user.GetByThread(ca.Datfile)
	}
	return li
}

//Defaults is default variables for templates.
type Defaults struct {
	AdminCGI    string
	GatewayCGI  string
	ThreadCGI   string
	ServerCGI   string
	Message     Message
	IsAdmin     bool
	IsFriend    bool
	Version     string
	DescChanges string
	DescNew     string
	DescRecent  string
	DescIndex   string
	DescSearch  string
	DescStatus  string
	Path        string
	EmptyList   template.HTML
}

//CGI is a base class for all http handlers.
type CGI struct {
	M        Message
	JC       *jsCache
	Req      *http.Request
	WR       http.ResponseWriter
	IsThread bool
}

//NewCGI reads messages file, and set params , returns CGI obj.
//CGI obj is cached.
func NewCGI(w http.ResponseWriter, r *http.Request) (*CGI, error) {
	c := &CGI{
		JC:  newJsCache(cfg.Docroot),
		WR:  w,
		M:   SearchMessage(r.Header.Get("Accept-Language"), cfg.FileDir),
		Req: r,
	}
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return c, nil
}

//Defaults returns default params for templates.
func (c *CGI) Defaults() *Defaults {
	href := `<a href="` + cfg.GatewayURL + `/recent" title="` + c.M["desc_recent"] + `">` + c.M["recent"] + `</a>`

	return &Defaults{
		cfg.AdminURL,
		cfg.GatewayURL,
		cfg.ThreadURL,
		cfg.ServerURL,
		c.M,
		c.IsAdmin(),
		c.IsFriend(),
		cfg.Version,
		c.M["desc_changes"],
		c.M["desc_new"],
		c.M["desc_recent"],
		c.M["desc_index"],
		c.M["desc_search"],
		c.M["desc_status"],
		c.Path(),
		template.HTML(fmt.Sprintf(c.M["empty_list"], href)),
	}
}

//Host returns servername or host in http header.
func (c *CGI) Host() string {
	host := cfg.ServerName
	if host == "" {
		host = c.Req.Host
	}
	return host
}

//IsAdmin returns tur if matches admin regexp setted in config file.
func (c *CGI) IsAdmin() bool {
	m, err := regexp.MatchString(cfg.ReAdminStr, c.Req.RemoteAddr)
	if err != nil {
		log.Fatal(err)
	}
	return m
}

//IsFriend returns tur if matches friend regexp setted in config file.
func (c *CGI) IsFriend() bool {
	m, err := regexp.MatchString(cfg.ReFriendStr, c.Req.RemoteAddr)
	if err != nil {
		log.Fatal(err)
	}
	return m
}

//isVisitor returns tur if matches visitor regexp setted in config file.
func (c *CGI) isVisitor() bool {
	m, err := regexp.MatchString(cfg.ReVisitorStr, c.Req.RemoteAddr)
	if err != nil {
		log.Fatal(err)
	}
	return m
}

//Path returns path part of url.
//e.g. /thread.CGI/hoe/moe -> hoe/moe
func (c *CGI) Path() string {
	p := strings.Split(c.Req.URL.Path, "/")
	//  /thread.CGI/hoe
	// 0/         1/  2
	var path string
	if len(p) > 1 {
		path = strings.Join(p[2:], "/")
	}
	return path
}

//extentions reads files with suffix in root dir and return them.
//if __merged file exists return it when useMerged=true.
func (c *CGI) extension(suffix string) []string {
	var filename []string
	d, err := util.AssetDir("www")
	if err != nil {
		log.Fatal(err)
	}
	for _, fname := range d {
		if util.HasExt(fname, suffix) {
			filename = append(filename, fname)
		}
	}
	if util.IsDir(cfg.Docroot) {
		err = util.EachFiles(cfg.Docroot, func(f os.FileInfo) error {
			i := f.Name()
			if util.HasExt(i, suffix) {
				if !util.HasString(filename, i) {
					filename = append(filename, i)
				}
			}
			return nil
		})
		if err != nil {
			log.Println(err)
		}
	}
	sort.Strings(filename)
	return filename
}

//Footer render footer.
func (c *CGI) Footer(menubar *Menubar) {
	g := struct {
		Menubar *Menubar
		Version string
	}{
		menubar,
		cfg.Version,
	}
	RenderTemplate("footer", g, c.WR)
}

//RFC822Time convers stamp to "2006-01-02 15:04:05"
func (c *CGI) RFC822Time(stamp int64) string {
	return time.Unix(stamp, 0).Format("2006-01-02 15:04:05")
}

//Menubar is var set for menubar.txt
type Menubar struct {
	Defaults
	ID  string
	RSS string
}

//MakeMenubar makes and returns *Menubar obj.
func (c *CGI) MakeMenubar(id, rss string) *Menubar {
	g := &Menubar{
		*c.Defaults(),
		id,
		rss,
	}
	return g
}

//Header renders header.txt with cookie.
func (c *CGI) Header(title, rss string, cookie []*http.Cookie, denyRobot bool) {
	if rss == "" {
		rss = cfg.GatewayURL + "/rss"
	}
	var js []string
	if c.Req.FormValue("__debug_js") != "" {
		js = c.extension("js")
	} else {
		c.JC.update()
	}
	h := struct {
		RootPath   string
		Title      string
		RSS        string
		Mergedjs   *jsCache
		JS         []string
		CSS        []string
		Menubar    *Menubar
		DenyRobot  bool
		Dummyquery int64
		IsThread   bool
		Defaults
	}{
		"/",
		title,
		rss,
		c.JC,
		js,
		c.extension("css"),
		c.MakeMenubar("top", rss),
		denyRobot,
		time.Now().Unix(),
		c.IsThread,
		*c.Defaults(),
	}
	if cookie != nil {
		for _, co := range cookie {
			http.SetCookie(c.WR, co)
		}
	}
	RenderTemplate("header", h, c.WR)
}

//ResAnchor returns a href  string with url.
func (c *CGI) ResAnchor(id, appli string, title string, absuri bool) string {
	title = util.StrEncode(title)
	var prefix, innerlink string
	if absuri {
		prefix = "http://" + c.Host()
	} else {
		innerlink = " class=\"innerlink\""
	}
	return fmt.Sprintf("<a href=\"%s%s%s%s/%s\"%s>", prefix, appli, "/", title, id, innerlink)
}

//HTMLFormat converts plain text to html , including converting link string to <a href="link">.
func (c *CGI) HTMLFormat(plain, appli string, title string, absuri bool) string {
	if strings.HasPrefix(plain, "@markdown") {
		plain = strings.Replace(plain, "<br>", "\n", -1)
		plain = strings.Replace(plain, "&lt;", "<", -1)
		plain = strings.Replace(plain, "&gt;", ">", -1)
		return string(blackfriday.MarkdownCommon([]byte(plain[len("@markdown"):])))
	}
	buf := strings.Replace(plain, "<br>", "\n", -1)
	buf = strings.Replace(buf, "\t", "        ", -1)

	buf = util.Escape(buf)
	regLink := regexp.MustCompile(`https?://[^\x00-\x20"'\(\)<>\[\]\x7F-\xFF]{2,}`)
	if cfg.EnableEmbed {
		var strs []string
		for _, str := range strings.Split(buf, "<br>") {
			s := regLink.ReplaceAllString(str, `<a href="$0">$0</a>`)
			strs = append(strs, s)
			for _, link := range regLink.FindAllString(str, -1) {
				e := util.EmbedURL(link)
				if e != "" {
					strs = append(strs, e)
					strs = append(strs, "")
				}
			}
		}
		buf = strings.Join(strs, "<br>")
	}

	reg1 := regexp.MustCompile("&gt;&gt;[0-9a-f]{8}")
	buf = reg1.ReplaceAllStringFunc(buf, func(str string) string {
		regg := regexp.MustCompile("(&gt;&gt;)([0-9a-f]{8})")
		id := regg.ReplaceAllString(str, "$2")
		return regg.ReplaceAllString(str, c.ResAnchor(id, appli, title, absuri)+"$1$2</a>")
	})
	reg3 := regexp.MustCompile(`(:[a-z0-9_]+:)`)
	buf = reg3.ReplaceAllStringFunc(buf, func(str string) string {
		return util.Emoji(str)
	})
	reg2 := regexp.MustCompile(`\[\[([^<>]+?)\]\]`)
	tmp := reg2.ReplaceAllStringFunc(buf, func(str string) string {
		bl := c.bracketLink(str[2:len(str)-2], appli, absuri)
		return bl
	})
	return util.EscapeSpace(tmp)
}

//bracketLink convert ling string to [[link]] string with href tag.
// if not thread and rec link, simply return [[link]]
func (c *CGI) bracketLink(link, appli string, absuri bool) string {
	var prefix string
	if absuri {
		prefix = "http://" + c.Host()
	}
	reg := regexp.MustCompile("^/(thread)/([^/]+)/([0-9a-f]{8})$")
	if m := reg.FindStringSubmatch(link); m != nil {
		url := prefix + cfg.ThreadURL + "/" + util.StrEncode(m[2]) + "/" + m[3]
		return "<a href=\"" + url + "\" class=\"reclink\">[[" + link + "]]</a>"
	}

	reg = regexp.MustCompile("^/(thread)/([^/]+)$")
	if m := reg.FindStringSubmatch(link); m != nil {
		uri := prefix + cfg.ThreadURL + "/" + util.StrEncode(m[2])
		return "<a href=\"" + uri + "\">[[" + link + "]]</a>"
	}

	reg = regexp.MustCompile("^([^/]+)/([0-9a-f]{8})$")
	if m := reg.FindStringSubmatch(link); m != nil {
		uri := prefix + appli + "/" + util.StrEncode(m[1]) + "/" + m[2]
		return "<a href=\"" + uri + "\" class=\"reclink\">[[" + link + "]]</a>"
	}

	reg = regexp.MustCompile("^([^/]+)$")
	if m := reg.FindStringSubmatch(link); m != nil {
		uri := prefix + appli + "/" + util.StrEncode(m[1])
		return "<a href=\"" + uri + "\">[[" + link + "]]</a>"
	}
	return "[[" + link + "]]"
}

//RemoveFileForm render remove_form_form page.
func (c *CGI) RemoveFileForm(ca *thread.Cache, title string) {
	s := struct {
		Cache     *thread.Cache
		CacheSize int64
		Title     string
		Defaults
	}{
		ca,
		ca.Size(),
		title,
		*c.Defaults(),
	}
	RenderTemplate("remove_file_form", s, c.WR)
}

//printJump render jump (redirect)page.
func (c *CGI) printJump(next string) {
	s := struct {
		Next template.HTML
	}{
		template.HTML(next),
	}
	RenderTemplate("jump", s, c.WR)
}

//Print302 renders jump page(meaning found and redirect)
func (c *CGI) Print302(next string) {
	c.Header("Loading...", "", nil, false)
	c.printJump(next)
	c.Footer(nil)
}

//Print403 renders 403 forbidden page with jump page.
func (c *CGI) Print403() {
	c.Header(c.M["403"], "", nil, true)
	fmt.Fprintf(c.WR, "<p>%s</p>", c.M["403_body"])
	c.Footer(nil)
}

//Print404 render the 404 page.
//if ca!=nil also renders info page of removing cache.
func (c *CGI) Print404(ca *thread.Cache, id string) {
	c.Header(c.M["404"], "", nil, true)
	fmt.Fprintf(c.WR, "<p>%s</p>", c.M["404_body"])
	if ca != nil {
		c.RemoveFileForm(ca, "")
	}
	c.Footer(nil)
}

//CheckVisitor returns true if visitor is permitted.
func (c *CGI) CheckVisitor() bool {
	return c.IsAdmin() || c.IsFriend() || c.isVisitor()
}

//HasAuth auth returns true if visitor is admin or friend.
func (c *CGI) HasAuth() bool {
	return c.IsAdmin() || c.IsFriend()
}

//PrintIndexList renders index_list.txt which renders threads in cachelist.
func (c *CGI) PrintIndexList(cl []*thread.Cache, target string, footer bool, searchNewFile bool, filter, tagg string) {
	s := struct {
		Target  string
		Filter  string
		Tag     string
		Taglist tag.Slice
		NoList  bool
		Defaults
		ListItem
	}{
		target,
		filter,
		tagg,
		user.Get(),
		len(cl) == 0,
		*c.Defaults(),
		*NewListItem(cl, true, target, searchNewFile, filter, tagg),
	}
	RenderTemplate("index_list", s, c.WR)
	if footer {
		c.PrintNewElementForm()
		c.Footer(nil)
	}
}

//PrintNewElementForm renders new_element_form.txt for posting new thread.
func (c *CGI) PrintNewElementForm() {
	const titleLimit = 30 //Charactors

	if !c.IsAdmin() && !c.IsFriend() {
		return
	}
	s := struct {
		Datfile    string
		TitleLimit int
		Defaults
	}{
		"",
		titleLimit,
		*c.Defaults(),
	}
	RenderTemplate("new_element_form", s, c.WR)
}

//IsBot returns true if client is bot.
func (c *CGI) IsBot() bool {
	robots := []string{
		"Google", "bot", "Yahoo", "archiver", "Wget", "Crawler", "Yeti", "Baidu",
	}
	agent := c.Req.Header.Get("User-Agent")
	for _, robot := range robots {
		if strings.Contains(agent, robot) {
			return true
		}
	}
	return false
}

//CheckGetCache return true
//if visitor is firend or admin and user-agent is not robot.
func (c *CGI) CheckGetCache() bool {

	if !c.HasAuth() {
		return false
	}
	return !c.IsBot()
}
