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
	"html"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
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

	"golang.org/x/text/language"
)

//message hold string map.
type message map[string]string

//newMessage reads from the file excpet #comment and stores them with url unescaping value.
func newMessage(filedir, fname string) message {
	var err error
	m := make(map[string]string)
	var dat []byte
	file := path.Join("file", fname)
	if dat, err = util.Asset(file); err != nil {
		log.Println(err)
	}
	file = filepath.Join(filedir, fname)
	if util.IsFile(fname) {
		dat1, err := ioutil.ReadFile(file)
		if err != nil {
			log.Println(err)
		} else {
			log.Println("loaded", file)
			dat = dat1
		}
	}
	if dat == nil {
		return nil
	}

	re := regexp.MustCompile(`^\s*#`)
	for _, line := range strings.Split(string(dat), "\n") {
		line = strings.Trim(line, "\r\n")
		if line == "" || re.MatchString(line) {
			continue
		}
		buf := strings.Split(line, "<>")
		if len(buf) == 2 {
			buf[1] = html.UnescapeString(buf[1])
			m[buf[0]] = buf[1]
			continue
		}
		return nil
	}

	return m
}

//searchMessage parse Accept-Language header ,selects most-weighted(biggest q)
//language ,reads the associated message file, and creates and returns message obj.
func searchMessage(acceptLanguage, filedir string) message {
	const defaultLanguage = "en" // Language code (see RFC3066)

	var lang []string
	if acceptLanguage != "" {
		tags, _, err := language.ParseAcceptLanguage(acceptLanguage)
		if err != nil {
			log.Println(err)
		} else {
			for _, tag := range tags {
				lang = append(lang, tag.String())
			}
		}
	}
	lang = append(lang, defaultLanguage)
	for _, l := range lang {
		slang := strings.Split(l, "-")[0]
		for _, j := range []string{l, slang} {
			if m := newMessage(filedir, "message-"+j+".txt"); m != nil {
				return m
			}
		}
	}
	return nil
}

//GatewayLink is a struct for gateway_link.txt
type GatewayLink struct {
	Message     message
	CGIname     string
	Command     string
	Description string
}

//Render renders "gateway_link.txt" and returns its resutl string which is not escaped in template.
//GatewayLink.Message must be setted up previously.
func (c GatewayLink) Render(cginame, command string) template.HTML {
	c.CGIname = cginame
	c.Command = command
	c.Description = c.Message["desc_"+command]
	return template.HTML(tmpH.ExecuteTemplate("gateway_link", c))
}

//ListItem is for list_item.txt
type ListItem struct {
	Cache      *thread.Cache
	CacheSize  int64
	Title      string
	Tags       tag.Slice
	Sugtags    []*tag.Tag
	Target     string
	Remove     bool
	IsAdmin    bool
	StrOpts    string
	GatewayCGI string
	ThreadURL  string
	Message    message
	CacheInfo  *thread.CacheInfo
	filter     string
	tag        string
}

//checkCache checks cache ca has specified tag and datfile doesn't contains filterd string.
func (l *ListItem) checkCache(ca *thread.Cache, target string) (string, bool) {
	x := util.FileDecode(ca.Datfile)
	if x == "" {
		return "", false
	}
	if l.filter != "" && !strings.Contains(strings.ToLower(x), l.filter) {
		return "", false
	}
	if l.tag != "" {
		switch {
		case ca.HasTagstr(strings.ToLower(l.tag)):
		case target == "recent" && suggest.HasTagstr(ca.Datfile, strings.ToLower(l.tag)):
		default:
			return "", false
		}
	}
	return x, true
}

//Render renders "list_items.txt" and returns its result string which is not escaped in template.
//ListItem.IsAdmin,filter,tag,Message must be setted up previously.
func (l ListItem) Render(ca *thread.Cache, remove bool, target string, search bool) template.HTML {
	x, ok := l.checkCache(ca, target)
	if !ok {
		return template.HTML("")
	}
	x = util.EscapeSpace(x)
	var strOpts string
	if search {
		strOpts = "?search_new_file=yes"
	}
	var sugtags []*tag.Tag
	if target == "recent" {
		strTags := make([]string, ca.LenTags())
		for i, v := range ca.GetTags() {
			strTags[i] = strings.ToLower(v.Tagstr)
		}
		for _, st := range suggest.Get(ca.Datfile, nil) {
			if !util.HasString(strTags, strings.ToLower(st.Tagstr)) {
				sugtags = append(sugtags, st)
			}
		}
	}
	l.CacheInfo = ca.ReadInfo()
	l.Cache = ca
	l.Title = x
	l.Tags = ca.GetTags()
	l.Sugtags = sugtags
	l.Target = target
	l.Remove = remove
	l.StrOpts = strOpts
	l.GatewayCGI = cfg.GatewayURL
	l.ThreadURL = cfg.ThreadURL
	return template.HTML(tmpH.ExecuteTemplate("list_item", l))
}

//cgi is a base class for all http handlers.
type cgi struct {
	m        message
	jc       *jsCache
	req      *http.Request
	wr       http.ResponseWriter
	filter   string
	tag      string
	IsThread bool
}

//newCGI reads messages file, and set params , returns cgi obj.
//cgi obj is cached.
func newCGI(w http.ResponseWriter, r *http.Request) *cgi {

	c := &cgi{
		jc:  newJsCache(cfg.Docroot),
		wr:  w,
		m:   searchMessage(r.Header.Get("Accept-Language"), cfg.FileDir),
		req: r,
	}
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return nil
	}
	return c
}

//host returns servername or host in http header.
func (c *cgi) host() string {
	host := cfg.ServerName
	if host == "" {
		host = c.req.Host
	}
	return host
}

//isAdmin returns tur if matches admin regexp setted in config file.
func (c *cgi) isAdmin() bool {
	m, err := regexp.MatchString(cfg.ReAdminStr, c.req.RemoteAddr)
	if err != nil {
		log.Fatal(err)
	}
	return m
}

//isFriend returns tur if matches friend regexp setted in config file.
func (c *cgi) isFriend() bool {
	m, err := regexp.MatchString(cfg.ReFriendStr, c.req.RemoteAddr)
	if err != nil {
		log.Fatal(err)
	}
	return m
}

//isVisitor returns tur if matches visitor regexp setted in config file.
func (c *cgi) isVisitor() bool {
	m, err := regexp.MatchString(cfg.ReVisitorStr, c.req.RemoteAddr)
	if err != nil {
		log.Fatal(err)
	}
	return m
}

//path returns path part of url.
//e.g. /thread.cgi/hoe/moe -> hoe/moe
func (c *cgi) path() string {
	p := strings.Split(c.req.URL.Path, "/")
	//  /thread.cgi/hoe
	// 0/         1/  2
	var path string
	if len(p) > 1 {
		path = strings.Join(p[2:], "/")
	}
	return path
}

//extentions reads files with suffix in root dir and return them.
//if __merged file exists return it when useMerged=true.
func (c *cgi) extension(suffix string) []string {
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

//footer render footer.
func (c *cgi) footer(menubar *Menubar) {
	g := struct {
		Menubar *Menubar
		Version string
	}{
		menubar,
		cfg.Version,
	}
	tmpH.RenderTemplate("footer", g, c.wr)
}

//rfc822Time convers stamp to "2006-01-02 15:04:05"
func (c *cgi) rfc822Time(stamp int64) string {
	return time.Unix(stamp, 0).Format("2006-01-02 15:04:05")
}

//printParagraph render paragraph.txt,just print constents.
//contentsKey must be a key of Message map.
func (c *cgi) printParagraph(contentsKey string) {
	g := struct {
		Contents template.HTML
	}{
		Contents: template.HTML(c.m[contentsKey]),
	}
	tmpH.RenderTemplate("paragraph", g, c.wr)
}

//Menubar is var set for menubar.txt
type Menubar struct {
	GatewayLink
	GatewayCGI string
	Message    message
	ID         string
	RSS        string
	IsAdmin    bool
	IsFriend   bool
}

//mekaMenubar makes and returns *Menubar obj.
func (c *cgi) makeMenubar(id, rss string) *Menubar {
	g := &Menubar{
		GatewayLink{
			Message: c.m,
		},
		cfg.GatewayURL,
		c.m,
		id,
		rss,
		c.isAdmin(),
		c.isFriend(),
	}
	return g
}

//header renders header.txt with cookie.
func (c *cgi) header(title, rss string, cookie []*http.Cookie, denyRobot bool) {
	if rss == "" {
		rss = cfg.GatewayURL + "/rss"
	}
	var js []string
	if c.req.FormValue("__debug_js") != "" {
		js = c.extension("js")
	} else {
		c.jc.update()
	}
	h := struct {
		Message    message
		RootPath   string
		Title      string
		RSS        string
		Mergedjs   *jsCache
		JS         []string
		CSS        []string
		Menubar    *Menubar
		DenyRobot  bool
		Dummyquery int64
		ThreadCGI  string
		IsThread   bool
	}{
		c.m,
		"/",
		title,
		rss,
		c.jc,
		js,
		c.extension("css"),
		c.makeMenubar("top", rss),
		denyRobot,
		time.Now().Unix(),
		cfg.ThreadURL,
		c.IsThread,
	}
	if cookie != nil {
		for _, co := range cookie {
			http.SetCookie(c.wr, co)
		}
	}
	tmpH.RenderTemplate("header", h, c.wr)
}

//resAnchor returns a href  string with url.
func (c *cgi) resAnchor(id, appli string, title string, absuri bool) string {
	title = util.StrEncode(title)
	var prefix, innerlink string
	if absuri {
		prefix = "http://" + c.host()
	} else {
		innerlink = " class=\"innerlink\""
	}
	return fmt.Sprintf("<a href=\"%s%s%s%s/%s\"%s>", prefix, appli, "/", title, id, innerlink)
}

//htmlFormat converts plain text to html , including converting link string to <a href="link">.
func (c *cgi) htmlFormat(plain, appli string, title string, absuri bool) string {
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
		return regg.ReplaceAllString(str, c.resAnchor(id, appli, title, absuri)+"$1$2</a>")
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
func (c *cgi) bracketLink(link, appli string, absuri bool) string {
	var prefix string
	if absuri {
		prefix = "http://" + c.host()
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

//removeFileForm render remove_form_form page.
func (c *cgi) removeFileForm(ca *thread.Cache, title string) {
	s := struct {
		Cache     *thread.Cache
		CacheSize int64
		Title     string
		IsAdmin   bool
		AdminCGI  string
		Message   message
	}{
		ca,
		ca.ReadInfo().Size,
		title,
		c.isAdmin(),
		cfg.AdminURL,
		c.m,
	}
	tmpH.RenderTemplate("remove_file_form", s, c.wr)
}

//printJump render jump (redirect)page.
func (c *cgi) printJump(next string) {
	s := struct {
		Next template.HTML
	}{
		template.HTML(next),
	}
	tmpH.RenderTemplate("jump", s, c.wr)
}

//print302 renders jump page(meaning found and redirect)
func (c *cgi) print302(next string) {
	c.header("Loading...", "", nil, false)
	c.printJump(next)
	c.footer(nil)
}

//print403 renders 403 forbidden page with jump page.
func (c *cgi) print403() {
	c.header(c.m["403"], "", nil, true)
	c.printParagraph("403_body")
	c.footer(nil)
}

//print404 render the 404 page.
//if ca!=nil also renders info page of removing cache.
func (c *cgi) print404(ca *thread.Cache, id string) {
	c.header(c.m["404"], "", nil, true)
	c.printParagraph("404_body")
	if ca != nil {
		c.removeFileForm(ca, "")
	}
	c.footer(nil)
}

//checkVisitor returns true if visitor is permitted.
func (c *cgi) checkVisitor() bool {
	return c.isAdmin() || c.isFriend() || c.isVisitor()
}

//hasAuth auth returns true if visitor is admin or friend.
func (c *cgi) hasAuth() bool {
	return c.isAdmin() || c.isFriend()
}

//printIndexList renders index_list.txt which renders threads in cachelist.
func (c *cgi) printIndexList(cl []*thread.Cache, target string, footer bool, searchNewFile bool) {
	s := struct {
		Target        string
		Filter        string
		Tag           string
		Taglist       tag.Slice
		Cachelist     []*thread.Cache
		GatewayCGI    string
		AdminCGI      string
		Message       message
		SearchNewFile bool
		IsAdmin       bool
		IsFriend      bool
		GatewayLink
		ListItem
	}{
		target,
		c.filter,
		c.tag,
		user.Get(),
		cl,
		cfg.GatewayURL,
		cfg.AdminURL,
		c.m,
		searchNewFile,
		c.isAdmin(),
		c.isFriend(),
		GatewayLink{
			Message: c.m,
		},
		ListItem{
			IsAdmin: c.isAdmin(),
			filter:  c.filter,
			tag:     c.tag,
			Message: c.m,
		},
	}
	tmpH.RenderTemplate("index_list", s, c.wr)
	if footer {
		c.printNewElementForm()
		c.footer(nil)
	}
}

//printNewElementForm renders new_element_form.txt for posting new thread.
func (c *cgi) printNewElementForm() {
	const titleLimit = 30 //Charactors

	if !c.isAdmin() && !c.isFriend() {
		return
	}
	s := struct {
		Datfile    string
		CGIname    string
		Message    message
		TitleLimit int
		IsAdmin    bool
	}{
		"",
		cfg.GatewayURL,
		c.m,
		titleLimit,
		c.isAdmin(),
	}
	tmpH.RenderTemplate("new_element_form", s, c.wr)
}

//isBot returns true if client is bot.
func (c *cgi) isBot() bool {
	robots := []string{
		"Google", "bot", "Yahoo", "archiver", "Wget", "Crawler", "Yeti", "Baidu",
	}
	agent := c.req.Header.Get("User-Agent")
	for _, robot := range robots {
		if strings.Contains(agent, robot) {
			return true
		}
	}
	return false
}

//checkGetCache return true
//if visitor is firend or admin and user-agent is not robot.
func (c *cgi) checkGetCache() bool {

	if !c.hasAuth() {
		return false
	}
	return !c.isBot()
}
