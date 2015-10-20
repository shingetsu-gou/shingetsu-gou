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
	"html/template"
	htmlTemplate "html/template"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"regexp"
	"strings"
	textTemplate "text/template"
	"time"

	"gopkg.in/ini.v1"
)

const (
	clientCycle        = 5 * time.Minute    // Seconds; Access client.cgi
	pingCycle          = 5 * time.Minute    // Seconds; Check nodes
	syncCycle          = 1 * time.Hour      // Seconds; Check cache
	initCycle          = 20 * time.Minute   // Seconds; Check initial node
	defaultUpdateRange = 24 * time.Hour     // Seconds
	timeErrorSigma     = 60                 // Seconds
	searchTimeout      = 10 * time.Minute   // Seconds
	defaultTimeout     = 20 * time.Second   // Seconds; Timeout for TCP
	getTimeout         = 2 * time.Minute    // Seconds; Timeout for /get
	clientTimeout      = 30 * time.Minute   // Seconds; client_timeout < sync_cycle
	retry              = 5                  // Times; Common setting
	retryJoin          = 2                  // Times; Join network
	defaultNodes       = 5                  // Nodes keeping in node list
	shareNodes         = 5                  // Nodes having the file
	searchDepth        = 30                 // Search node size
	titleLimit         = 30                 //Charactors
	defaultLanguage    = "en"               // Language code (see RFC3066)
	saveCookie         = 7 * 24 * time.Hour // Seconds

	// regexp
	robot = "Google|bot|Yahoo|archiver|Wget|Crawler|Yeti|Baidu"

	dnsname        = ""  // Server name for shinGETsu protocol
	querySeparator = "/" // Must be "/"
	rootPath       = "/" // path of URI for root

	templateSuffix = ".txt"
	useCookie      = true
)

var (
	types = []string{"thread"}

	saveRecord  = make(map[string]int64)
	savesize    = make(map[string]int) // It is not seconds, but number.
	getRange    = make(map[string]int64)
	syncRange   = make(map[string]int64)
	saveRemoved = make(map[string]int64)

	//DefaultPort is listening port
	DefaultPort   int
	maxConnection int
	docroot       string
	logDir        string
	runDir        string
	fileDir       string
	cacheDir      string
	templateDir   string
	spamList      string
	initnodeList  string
	nodeAllowFile string
	nodeDenyFile  string

	reAdminStr     string
	reFriendStr    string
	reVisitorStr   string
	reAdmin        *regexp.Regexp
	reFriend       *regexp.Regexp
	reVisitor      *regexp.Regexp
	serverName     string
	tagSize        int
	rssRange       int64
	topRecentRange int64
	recentRange    int64
	recordLimit    int
	enable2ch      bool
	//EnableNAT is enable if you want to use nat.
	EnableNAT bool
	//ExternalPort is opened port by NAT.if no NAT it equals to DeafultPort.
	ExternalPort int

	motd        string
	nodeFile    string
	searchFile  string
	update      string
	recent      string
	clientLog   string
	lock        string
	searchLock  string
	adminSearch string
	adminSid    string
	pid         string
	lookup      string
	taglist     string
	sugtag      string

	serverURL  string
	gatewayURL string
	threadURL  string
	adminURL   string
	xsl        string

	threadPageSize       int
	defaultThumbnailSize string
	forceThumbnail       bool

	application map[string]string

	// asis, md5, sha1, sha224, sha256, sha384, or sha512
	//	cache_hash_method = "asis"
	//others are not implemented for gou for now.

	version string

	defaultInitNode = []string{
		"node.shingetsu.info:8000/server.cgi",
		"pushare.zenno.info:8000/server.cgi",
	}

	initNode     *confList
	cachedRule   *regexpList
	nodeAllow    *regexpList
	nodeDeny     *regexpList
	dataKeyTable *DatakeyTable

	queue             *updateQue
	suggestedTagTable *SuggestedTagTable
	userTagList       *UserTagList
	lookupTable       *LookupTable
	searchList        *SearchList
	nodeList          *NodeList
	recentList        *RecentList
	updateList        *UpdateList

	errGet  = errors.New("cannot get data")
	errSpam = errors.New("this is spam")

	htemplates = htmlTemplate.New("")
	ttemplates = textTemplate.New("")
)

//config represents ini file.
type config struct {
	i *ini.File
}

//newConfig make a config instance from the ini files and returns it.
func newConfig() *config {
	var err error
	c := &config{}
	c.i = ini.Empty()
	for _, f := range []string{"file/saku.ini", "/usr/local/etc/saku/saku.ini", "/etc/saku/saku.ini"} {
		if IsFile(f) {
			err = c.i.Append(f)
			if err != nil {
				log.Fatal("cannot load ini files")
			}
		} else {
			log.Println(f, "not found")
		}
	}
	usr, err := user.Current()
	if err == nil {
		h := usr.HomeDir + "/.saku/saku.ini"
		if IsFile(h) {
			err = c.i.Append(h)
			if err != nil {
				log.Fatal("cannot load ini files")
			}
		} else {
			log.Println(h, "not found")
		}
	}
	return c
}

//getIntValue gets int value from ini file.
func (c *config) getIntValue(section, key string, vdefault int) int {
	return c.i.Section(section).Key(key).MustInt(vdefault)
}

//getInt64Value gets int value from ini file.
func (c *config) getInt64Value(section, key string, vdefault int64) int64 {
	return c.i.Section(section).Key(key).MustInt64(vdefault)
}

//getStringValue gets string from ini file.
func (c *config) getStringValue(section, key string, vdefault string) string {
	return c.i.Section(section).Key(key).MustString(vdefault)
}

//getBoolValue gets bool value from ini file.
func (c *config) getBoolValue(section, key string, vdefault bool) bool {
	return c.i.Section(section).Key(key).MustBool(vdefault)
}

//getPathValue gets path from ini file.
func (c *config) getRelativePathValue(section, key string, vdefault string) string {
	p := c.i.Section(section).Key(key).MustString(vdefault)
	h := p
	if !path.IsAbs(p) {
		h = path.Join(docroot, p)
	}
	return h
}

//getPathValue gets path from ini file.
func (c *config) getPathValue(section, key string, vdefault string) string {
	p := c.i.Section(section).Key(key).MustString(vdefault)
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	h := p
	if !path.IsAbs(p) {
		h = path.Join(wd, p)
	}
	return h
}

//Get Gou version for useragent and servername.
func getVersion() string {
	ver := "0.0.0"

	versionFile := fileDir + "/version.txt"
	f, err := os.Open(versionFile)
	defer fclose(f)
	if err == nil {
		cont, err := ioutil.ReadAll(f)
		if err == nil {
			ver += "; git/" + string(cont)
		}
	}
	return "shinGETsu/0.7 (Gou/" + ver + ")"
}

//setupTemplate adds funcmap to template var and parse files.
func setupTemplate() {
	funcMap := map[string]interface{}{
		"add":          func(a, b int) int { return a + b },
		"sub":          func(a, b int) int { return a - b },
		"mul":          func(a, b int) int { return a * b },
		"div":          func(a, b int) int { return a / b },
		"toMB":         func(a int) float64 { return float64(a) / (1024 * 1024) },
		"toKB":         func(a int) float64 { return float64(a) / (1024) },
		"toInt":        func(a int64) int { return int(a) },
		"stopEscaping": func(a string) template.HTML { return template.HTML(a) },
		"strEncode":    strEncode,
		"escape":       escape,
		"escapeSpace":  escapeSpace,
		"localtime":    func(stamp int64) string { return time.Unix(stamp, 0).Format("2006-01-02 15:04") },
		"unescapedPrintf": func(format string, a ...interface{}) htmlTemplate.HTML {
			return htmlTemplate.HTML(fmt.Sprintf(format, a))
		},
	}

	templateFiles := templateDir + "/*" + templateSuffix
	if !IsDir(templateDir) {
		log.Fatal(templateDir, "not found")
	}
	htemplates.Funcs(htmlTemplate.FuncMap(funcMap))
	_, err := htemplates.ParseGlob(templateFiles)
	if err != nil {
		log.Fatal(err)
	}

	templateFiles = templateDir + "/rss1" + templateSuffix
	ttemplates.Funcs(textTemplate.FuncMap(funcMap))
	_, err = ttemplates.ParseFiles(templateFiles)
	if err != nil {
		log.Fatal(err)
	}
}

//InitVariables initializes some global and map vars.
func InitVariables() {
	setting := newConfig()

	DefaultPort = setting.getIntValue("Network", "port", 8010)
	maxConnection = setting.getIntValue("Network", "max_connection", 20)
	docroot = setting.getPathValue("Path", "docroot", "./www")                        //path from cwd
	logDir = setting.getPathValue("Path", "log_dir", "./log")                         //path from cwd
	runDir = setting.getRelativePathValue("Path", "run_dir", "../run")                //path from docroot
	fileDir = setting.getRelativePathValue("Path", "file_dir", "../file")             //path from docroot
	cacheDir = setting.getRelativePathValue("Path", "cache_dir", "../cache")          //path from docroot
	templateDir = setting.getRelativePathValue("Path", "template_dir", "../template") //path from docroot
	spamList = setting.getRelativePathValue("Path", "spam_list", "../file/spam.txt")
	initnodeList = setting.getRelativePathValue("Path", "initnode_list", "../file/initnode.txt")
	nodeAllowFile = setting.getRelativePathValue("Path", "node_allow", "../file/node_allow.txt")
	nodeDenyFile = setting.getRelativePathValue("Path", "node_deny", "../file/node_deny.txt")

	reAdminStr = setting.getStringValue("Gateway", "admin", "^(127|\\[::1\\])")
	reFriendStr = setting.getStringValue("Gateway", "friend", "^(127|\\[::1\\])")
	reVisitorStr = setting.getStringValue("Gateway", "visitor", ".")
	serverName = setting.getStringValue("Gateway", "server_name", "")
	tagSize = setting.getIntValue("Gateway", "tag_size", 20)
	rssRange = setting.getInt64Value("Gateway", "rss_range", 3*24*60*60)
	topRecentRange = setting.getInt64Value("Gateway", "top_recent_range", 3*24*60*60)
	recentRange = setting.getInt64Value("Gateway", "recent_range", 31*24*60*60)
	recordLimit = setting.getIntValue("Gateway", "record_limit", 2048)
	enable2ch = setting.getBoolValue("Gateway", "enable_2ch", false)
	EnableNAT = setting.getBoolValue("Gateway", "enable_nat", false)
	ExternalPort = DefaultPort

	motd = fileDir + "/motd.txt"
	nodeFile = runDir + "/node.txt"
	searchFile = runDir + "/search.txt"
	update = runDir + "/update.txt"
	recent = runDir + "/recent.txt"
	clientLog = runDir + "/client.txt"
	lock = runDir + "/lock.txt"
	searchLock = runDir + "/touch.txt"
	adminSearch = runDir + "/admintouch.txt"
	adminSid = runDir + "/sid.txt"
	pid = runDir + "/pid.txt"
	lookup = runDir + "/lookup.txt"
	taglist = runDir + "/tag.txt"
	sugtag = runDir + "/sugtag.txt"

	serverURL = rootPath + "server.cgi"
	gatewayURL = rootPath + "gateway.cgi"
	threadURL = rootPath + "thread.cgi"
	adminURL = rootPath + "admin.cgi"
	xsl = rootPath + "rss1.xsl"

	threadPageSize = setting.getIntValue("Application Thread", "page_size", 50)
	defaultThumbnailSize = setting.getStringValue("Application Thread", "thumbnail_size", "")
	forceThumbnail = setting.getBoolValue("Application Thread", "force_thumbnail", false)

	version = getVersion()

	initNode = newConfList(initnodeList, defaultInitNode)
	cachedRule = newRegexpList(spamList)
	nodeAllow = newRegexpList(nodeAllowFile)
	nodeDeny = newRegexpList(nodeDenyFile)
	dataKeyTable = newDatakeyTable(runDir + "/datakey.txt")

	suggestedTagTable = newSuggestedTagTable()
	userTagList = newUserTagList()
	lookupTable = newLookupTable()
	searchList = newSearchList()
	nodeList = newNodeList()
	recentList = newRecentList()
	updateList = newUpdateList()
	queue = newUpdateQue()

	var err error
	reAdmin, err = regexp.Compile(reAdminStr)
	if err != nil {
		log.Fatal("admin regexp string is illegal", err)
	}
	reFriend, err = regexp.Compile(reFriendStr)
	if err != nil {
		log.Fatal("freind regexp string is illegal", err)
	}
	reVisitor, err = regexp.Compile(reVisitorStr)
	if err != nil {
		log.Fatal("visitor regexp string is illegal", err)
	}
	for _, t := range types {
		ctype := "Application " + strings.ToUpper(t)
		saveRecord[t] = setting.getInt64Value(ctype, "save_record", 0)
		savesize[t] = setting.getIntValue(ctype, "save_size", 1)
		getRange[t] = setting.getInt64Value(ctype, "get_range", 31*24*60*60)
		if getRange[t] > time.Now().Unix() {
			log.Fatal("get_range is too big")
		}
		syncRange[t] = setting.getInt64Value(ctype, "sync_range", 10*24*60*60)
		if syncRange[t] > time.Now().Unix() {
			log.Fatal("sync_range is too big")
		}
		saveRemoved[t] = setting.getInt64Value(ctype, "save_removed", 50*24*60*60)
		if saveRemoved[t] > time.Now().Unix() {
			log.Fatal("save_removed is too big")
		}
	}

	setupTemplate()
}
