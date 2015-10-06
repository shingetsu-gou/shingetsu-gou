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
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

const (
	clientCycle = 5 * time.Minute  // Seconds; Access client.cgi
	pingCycle   = 5 * time.Minute  // Seconds; Check nodes
	syncCycle   = 5 * time.Hour    // Seconds; Check cache
	initCycle   = 20 * time.Minute // Seconds; Check initial node
	updateRange = 24 * time.Hour   // Seconds
	//	time_error          = 60 * time.Second // Seconds
	searchTimeout  = 10 * time.Minute // Seconds
	defaultTimeout = 20 * time.Second // Seconds; Timeout for TCP
	getTimeout     = 2 * time.Minute  // Seconds; Timeout for /get
	clientTimeout  = 30 * time.Minute // Seconds; client_timeout < sync_cycle
	retry          = 5                // Times; Common setting
	retryJoin      = 2                // Times; Join network
	defaultNodes   = 5                // Nodes keeping in node list
	shareNodes     = 5                // Nodes having the file
	searchDepth    = 30               // Search node size

	defaultLanguage = "en" // Language code (see RFC3066)

	// regexp
	robot = "Google|bot|Yahoo|archiver|Wget|Crawler|Yeti|Baidu"

	dnsname        = ""  // Server name for shinGETsu protocol
	querySeparator = "/" // Must be "/"
	rootPath       = "/" // path of URI for root

	templateSuffix = ".txt"
)

var (
	setting = newConfig()

	types = []string{"thread"}

	saveRecord  = make(map[string]int)
	savesize    = make(map[string]int) // It is not seconds, but number.
	getRange    = make(map[string]int)
	syncRange   = make(map[string]int)
	saveRemoved = make(map[string]int)

	defaultPort = setting.getIntValue("Network", "port", 8000)
	datPort     = setting.getIntValue("Network", "dat_port", 8001)
	//	max_connection = setting.getIntValue("Network", "max_connection", 20)

	docroot       = setting.getPathValue("Path", "docroot", "./www")
	logDir        = setting.getPathValue("Path", "log_dir", "./log")
	runDir        = setting.getPathValue("Path", "run_dir", "../run")
	fileDir       = setting.getPathValue("Path", "file_dir", "../file")
	cacheDir      = setting.getPathValue("Path", "cache_dir", "../cache")
	templateDir   = setting.getPathValue("Path", "template_dir", "../template")
	spamList      = setting.getPathValue("Path", "spam_list", "../file/spam.txt")
	initnodeList  = setting.getPathValue("Path", "initnode_list", "../file/initnode.txt")
	nodeAllowFile = setting.getPathValue("Path", "node_allow", "../file/node_allow.txt")
	nodeDenyFile  = setting.getPathValue("Path", "node_deny", "../file/node_deny.txt")

	reAdmin        = setting.getStringValue("Gateway", "admin", "^127")
	reFriend       = setting.getStringValue("Gateway", "friend", "^127")
	reVisitor      = setting.getStringValue("Gateway", "visitor", ".")
	serverName     = setting.getStringValue("Gateway", "server_name", "")
	tagSize        = setting.getIntValue("Gateway", "tag_size", 20)
	rssRange       = setting.getIntValue("Gateway", "rss_range", 3*24*60*60)
	topRecentRange = setting.getIntValue("Gateway", "top_recent_range", 3*24*60*60)
	recentRange    = setting.getIntValue("Gateway", "recent_range", 31*24*60*60)
	recordLimit    = setting.getIntValue("Gateway", "record_limit", 2048)
	enable2ch      = setting.getBoolValue("Gateway", "enable_2ch", false)

	motd        = fileDir + "/motd.txt"
	nodeFile    = runDir + "/node.txt"
	searchFile  = runDir + "/search.txt"
	update      = runDir + "/update.txt"
	recent      = runDir + "/recent.txt"
	clientLog   = runDir + "/client.txt"
	lock        = runDir + "/lock.txt"
	searchLock  = runDir + "/touch.txt"
	adminSearch = runDir + "/admintouch.txt"
	adminSid    = runDir + "/sid.txt"
	pid         = runDir + "/pid.txt"
	lookup      = runDir + "/lookup.txt"
	taglist     = runDir + "/tag.txt"
	sugtag      = runDir + "/sugtag.txt"

	serverCgi  = rootPath + "server.cgi"
	gatewayCgi = rootPath + "gateway.cgi"
	threadCgi  = rootPath + "thread.cgi"
	adminCgi   = rootPath + "admin.cgi"
	xsl        = rootPath + "rss1.xsl"

	threadPageSize       = setting.getIntValue("Application Thread", "page_size", 50)
	defaultThumbnailSize = setting.getStringValue("Application Thread", "thumbnail_size", "")
	forceThumbnail       = setting.getBoolValue("Application Thread", "force_thumbnail", false)

	application = map[string]string{"thread": threadCgi}
	useCookie   = true
	saveCookie  = 7 * 24 * time.Hour
	// Seconds

	// asis, md5, sha1, sha224, sha256, sha384, or sha512
	//	cache_hash_method = "asis"
	//others are not implemented for gou for now.

	version = getVersion()

	defaultInitNode = []string{
		"node.shingetsu.info:8000/server.cgi",
		"pushare.zenno.info:8000/server.cgi",
	}

	cachedRule = newRegexpList(spamList)
	absDocroot string
	queue      = newUpdateQue()
)

type config struct {
	i *ini.File
}

func newConfig() *config {
	var err error
	c := &config{}
	c.i, err = ini.Load("file/saku.ini", "/usr/local/etc/saku/saku.ini", "/etc/saku/saku.ini")
	if err != nil {
		log.Fatal("cannot load ini files")
	}
	usr, err := user.Current()
	if err == nil {
		h := usr.HomeDir + "/.saku/saku.ini"
		err = c.i.Append(h)
		if err != nil {
			log.Fatal("cannot load ini files")
		}
	}
	return c
}

func (c *config) getIntValue(section, key string, vdefault int) int {
	return c.i.Section(section).Key(key).MustInt(vdefault)
}

func (c *config) getStringValue(section, key string, vdefault string) string {
	return c.i.Section(section).Key(key).MustString(vdefault)
}

func (c *config) getBoolValue(section, key string, vdefault bool) bool {
	return c.i.Section(section).Key(key).MustBool(vdefault)
}

func (c *config) getPathValue(section, key string, vdefault string) string {
	p := c.i.Section(section).Key(key).MustString(vdefault)
	usr, err := user.Current()
	h := p
	if err == nil {
		h = usr.HomeDir + p
	}
	return h
}

//Get Gou version for useragent and servername.
func getVersion() string {
	ver := "0.0.1"

	versionFile := docroot + "/" + fileDir + "/version.txt"
	f, err := os.Open(versionFile)
	if err == nil {
		defer close(f)
		cont, err := ioutil.ReadAll(f)
		if err == nil {
			ver += "; git/" + string(cont)
		}
	}
	return "shinGETsu/0.7 (Gou/" + ver + ")"
}

func InitConfig() {

	for _, t := range types {
		ctype := "Application " + strings.ToUpper(t)
		saveRecord[t] = setting.getIntValue(ctype, "save_record", 0)
		savesize[t] = setting.getIntValue(ctype, "save_size", 1)
		getRange[t] = setting.getIntValue(ctype, "get_range", 31*24*60*60)
		syncRange[t] = setting.getIntValue(ctype, "sync_range", 10*24*60*60)
		saveRemoved[t] = setting.getIntValue(ctype, "save_removed", 50*24*60*60)
	}
}
