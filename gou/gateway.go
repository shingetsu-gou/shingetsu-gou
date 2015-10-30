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
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/language"
)

//const searchTimeout = 10 * time.Minute // used for timeout of lockfile,for now not used

//message hold string map.
type message map[string]string

//newMessage reads from the file excpet #comment and stores them with url unescaping value.
func newMessage(file string) message {
	m := make(map[string]string)
	re := regexp.MustCompile(`^\s*#`)
	err := eachLine(file, func(line string, i int) error {
		var err error
		if re.MatchString(line) {
			return nil
		}
		buf := strings.Split(line, "<>")
		if len(buf) == 2 {
			buf[1] = html.UnescapeString(buf[1])
			m[buf[0]] = buf[1]
		}
		return err
	})
	if err != nil {
		log.Println(file, err)
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
			file := path.Join(filedir, "message-"+j+".txt")
			if IsFile(file) {
				return newMessage(file)
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
	return template.HTML(executeTemplate("gateway_link", c))
}

//ListItem is for list_item.txt
type ListItem struct {
	Cache      *cache
	Title      string
	Tags       tagslice
	Sugtags    []*tag
	Target     string
	Remove     bool
	IsAdmin    bool
	StrOpts    string
	GatewayCGI string
	ThreadURL  string
	Message    message
	filter     string
	tag        string
}

//checkCache checks cache ca has specified tag and datfile doesn't contains filterd string.
func (l *ListItem) checkCache(ca *cache, target string) (string, bool) {
	x := fileDecode(ca.Datfile)
	if x == "" {
		return "", false
	}
	if l.filter != "" && !strings.Contains(l.filter, strings.ToLower(x)) {
		return "", false
	}
	if l.tag != "" {
		switch {
		case ca.hasTagstr(strings.ToLower(l.tag)):
		case target == "recent" && suggestedTagTable.hasTagstr(ca.Datfile, strings.ToLower(l.tag)):
		default:
			return "", false
		}
	}
	return x, true
}

//Render renders "list_items.txt" and returns its resutl string which is not escaped in template.
//ListItem.IsAdmin,filter,tag,Message must be setted up previously.
func (l ListItem) Render(ca *cache, remove bool, target string, search bool) template.HTML {
	x, ok := l.checkCache(ca, target)
	if !ok {
		return template.HTML("")
	}
	x = escapeSpace(x)
	var strOpts string
	if search {
		strOpts = "?search_new_file=yes"
	}
	var sugtags []*tag
	if target == "recent" {
		strTags := make([]string, ca.lenTags())
		for i, v := range ca.getTags() {
			strTags[i] = strings.ToLower(v.Tagstr)
		}
		for _, st := range suggestedTagTable.get(ca.Datfile, nil) {
			if !hasString(strTags, strings.ToLower(st.Tagstr)) {
				sugtags = append(sugtags, st)
			}
		}
	}
	l.Cache = ca
	l.Title = x
	l.Tags = ca.getTags()
	l.Sugtags = sugtags
	l.Target = target
	l.Remove = remove
	l.StrOpts = strOpts
	l.GatewayCGI = GatewayURL
	l.ThreadURL = ThreadURL
	return template.HTML(executeTemplate("list_item", l))
}

type cgiConfig struct {
	serverName    string
	reAdminStr    string
	reFriendStr   string
	reVisitorStr  string
	docroot       string
	fileDir       string
	maxConnection int
}

func newCGIConfig(cfg *Config) *cgiConfig {
	return &cgiConfig{
		serverName:    cfg.ServerName,
		reAdminStr:    cfg.ReAdminStr,
		reFriendStr:   cfg.ReFriendStr,
		reVisitorStr:  cfg.ReVisitorStr,
		docroot:       cfg.Docroot,
		fileDir:       cfg.FileDir,
		maxConnection: cfg.MaxConnection,
	}
}

//cgi is a base class for all http handlers.
type cgi struct {
	*cgiConfig
	m      message
	jc     *jsCache
	req    *http.Request
	wr     http.ResponseWriter
	filter string
	tag    string
}

func (c *cgi) host() string {
	host := c.serverName
	if host == "" {
		host = c.req.Host
	}
	return host
}

func (c *cgi) isAdmin() bool {
	m, err := regexp.MatchString(c.reAdminStr, c.req.RemoteAddr)
	if err != nil {
		log.Fatal(err)
	}
	return m
}

func (c *cgi) isFriend() bool {
	m, err := regexp.MatchString(c.reFriendStr, c.req.RemoteAddr)
	if err != nil {
		log.Fatal(err)
	}
	return m
}

func (c *cgi) isVisitor() bool {
	m, err := regexp.MatchString(c.reVisitorStr, c.req.RemoteAddr)
	if err != nil {
		log.Fatal(err)
	}
	return m
}

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

var cgis chan *cgi

//newCGI reads messages file, and set params , returns cgi obj.
func newCGI(w http.ResponseWriter, r *http.Request, cfg *cgiConfig) *cgi {
	if cgis == nil {
		cgis = make(chan *cgi, cfg.maxConnection)
	}
	var c *cgi
	select {
	case c = <-cgis:
	default:
		c = &cgi{
			cgiConfig: cfg,
		}
	}
	c.jc = newJsCache(c.docroot)
	c.wr = w
	c.m = searchMessage(r.Header.Get("Accept-Language"), c.fileDir)
	c.req = r
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return nil
	}
	return c
}

//close returns cgi instance to channel(free list).
func (c *cgi) close() {
	if c == nil {
		return
	}
	select {
	case cgis <- c:
	default:
	}
}

//extentions reads files with suffix in root dir and return them.
//if __merged file exists return it when useMerged=true.
func (c *cgi) extension(suffix string, useMerged bool) []string {
	var filename []string
	var merged string
	err := eachFiles(c.docroot, func(f os.FileInfo) error {
		i := f.Name()
		if strings.HasSuffix(i, "."+suffix) && (!strings.HasPrefix(i, ".") || strings.HasPrefix(i, "_")) {
			filename = append(filename, i)
		} else {
			if useMerged && i == "__merged."+suffix {
				merged = i
			}
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}

	if merged != "" {
		return []string{merged}
	}
	sort.Strings(filename)
	return filename
}

//footer render footer.
func (c *cgi) footer(menubar *Menubar) {
	g := struct {
		Menubar *Menubar
	}{
		menubar,
	}
	renderTemplate("footer", g, c.wr)
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
	renderTemplate("paragraph", g, c.wr)
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
		GatewayURL,
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
		rss = GatewayURL + "/rss"
	}
	var js []string
	if c.req.FormValue("__debug_js") != "" {
		js = c.extension("js", false)
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
		AppliType  string
	}{
		c.m,
		"/",
		title,
		rss,
		c.jc,
		js,
		c.extension("css", false),
		c.makeMenubar("top", rss),
		denyRobot,
		time.Now().Unix(),
		ThreadURL,
		"thread",
	}
	if cookie != nil {
		for _, co := range cookie {
			http.SetCookie(c.wr, co)
		}
	}
	renderTemplate("header", h, c.wr)
}

//resAnchor retuns a href  string with url.
func (c *cgi) resAnchor(id, appli string, title string, absuri bool) string {
	title = strEncode(title)
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
	buf := strings.Replace(plain, "<br>", "\n", -1)
	buf = strings.Replace(buf, "\t", "        ", -1)
	buf = escape(buf)
	reg := regexp.MustCompile(`https?://[^\x00-\x20"'\(\)<>\[\]\x7F-\xFF]{2,}`)
	buf = reg.ReplaceAllString(buf, `<a href="$0">$0</a>`)
	reg = regexp.MustCompile("&gt;&gt;[0-9a-f]{8}")
	buf = reg.ReplaceAllStringFunc(buf, func(str string) string {
		regg := regexp.MustCompile("(&gt;&gt;)([0-9a-f]{8})")
		id := regg.ReplaceAllString(str, "$2")
		return regg.ReplaceAllString(str, c.resAnchor(id, appli, title, absuri)+"$1$2</a>")
	})
	reg = regexp.MustCompile(`\[\[([^<>]+?)\]\]`)
	tmp := reg.ReplaceAllStringFunc(buf, func(str string) string {
		bl := c.bracketLink(str[2:len(str)-2], appli, absuri)
		return bl
	})
	return escapeSpace(tmp)
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
		url := prefix + ThreadURL + "/" + strEncode(m[2]) + "/" + m[3]
		return "<a href=\"" + url + "\" class=\"reclink\">[[" + link + "]]</a>"
	}

	reg = regexp.MustCompile("^/(thread)/([^/]+)$")
	if m := reg.FindStringSubmatch(link); m != nil {
		uri := prefix + ThreadURL + "/" + strEncode(m[2])
		return "<a href=\"" + uri + "\">[[" + link + "]]</a>"
	}

	reg = regexp.MustCompile("^([^/]+)/([0-9a-f]{8})$")
	if m := reg.FindStringSubmatch(link); m != nil {
		uri := prefix + appli + "/" + strEncode(m[1]) + "/" + m[2]
		return "<a href=\"" + uri + "\" class=\"reclink\">[[" + link + "]]</a>"
	}

	reg = regexp.MustCompile("^([^/]+)$")
	if m := reg.FindStringSubmatch(link); m != nil {
		uri := prefix + appli + "/" + strEncode(m[1])
		return "<a href=\"" + uri + "\">[[" + link + "]]</a>"
	}
	return "[[" + link + "]]"
}

//removeFileForm render remove_form_form page.
func (c *cgi) removeFileForm(ca *cache, title string) {
	s := struct {
		Cache    *cache
		Title    string
		IsAdmin  bool
		AdminCGI string
		Message  message
	}{
		ca,
		title,
		c.isAdmin(),
		AdminURL,
		c.m,
	}
	renderTemplate("remove_file_form", s, c.wr)
}

//printJump render jump (redirect)page.
func (c *cgi) printJump(next string) {
	s := struct {
		Next template.HTML
	}{
		template.HTML(next),
	}
	renderTemplate("jump", s, c.wr)
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
func (c *cgi) print404(ca *cache, id string) {
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
func (c *cgi) printIndexList(cl []*cache, target string, footer bool, searchNewFile bool) {
	s := struct {
		Target        string
		Filter        string
		Tag           string
		Taglist       tagslice
		Cachelist     []*cache
		GatewayCGI    string
		AdminCGI      string
		Message       message
		SearchNewFile bool
		IsAdmin       bool
		IsFriend      bool
		Types         []string
		GatewayLink
		ListItem
	}{
		target,
		c.filter,
		c.tag,
		utag.get(),
		cl,
		GatewayURL,
		AdminURL,
		c.m,
		searchNewFile,
		c.isAdmin(),
		c.isFriend(),
		[]string{"thread"},
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
	renderTemplate("index_list", s, c.wr)
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
		GatewayURL,
		c.m,
		titleLimit,
		c.isAdmin(),
	}
	renderTemplate("new_element_form", s, c.wr)
}

//checkGetCache return true
//if visitor is firend or admin and user-agent is not robot.
func (c *cgi) checkGetCache() bool {
	const (
		robot = "Google|bot|Yahoo|archiver|Wget|Crawler|Yeti|Baidu"
	)

	if !c.hasAuth() {
		return false
	}
	agent := c.req.Header.Get("User-Agent")
	reg, err := regexp.Compile(robot)
	if err != nil {
		log.Println(err)
		return true
	}
	if reg.MatchString(agent) {
		log.Println("ng")
		return false
	}
	return true
}
