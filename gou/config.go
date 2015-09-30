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
	client_cycle        = 5 * time.Minute  // Seconds; Access client.cgi
	ping_cycle          = 5 * time.Minute  // Seconds; Check nodes
	sync_cycle          = 5 * time.Hour    // Seconds; Check cache
	init_cycle          = 20 * time.Minute // Seconds; Check initial node
	update_range        = 24 * time.Hour   // Seconds
	wait_update         = 10 * time.Second // Seconds
	time_error          = 60 * time.Second // Seconds
	search_timeout      = 10 * time.Minute // Seconds
	timeout             = 20 * time.Second // Seconds; Timeout for TCP
	get_timeout         = 2 * time.Minute  // Seconds; Timeout for /get
	client_timeout      = 30 * time.Minute // Seconds; client_timeout < sync_cycle
	tk_save_warn        = 5 * time.Minute  // Seconds
	retry               = 5                // Times; Common setting
	retry_join          = 2                // Times; Join network
	default_nodes       = 5                // Nodes keeping in node list
	share_nodes         = 5                // Nodes having the file
	search_depth        = 30               // Search node size
	tiedfile_cache_size = 30

	broadcast_py = "../tool/broadcast.py" // Broadcast script path

	rss_version = "1" // RSS version; must be "1"
	google      = "http://www.google.co.jp/search"
	language    = "en" // Language code (see RFC3066)

	// regexp
	robot = "Google|bot|Yahoo|archiver|Wget|Crawler|Yeti|Baidu"

	dnsname         = ""  // Server name for shinGETsu protocol
	query_separator = "/" // Must be "/"
	root_path       = "/" // path of URI for root

	template_suffix = ".txt"
)

var (
	setting = newConfig()

	types = []string{"thread"}

	save_record  = make(map[string]int)
	save_size    = make(map[string]int) // It is not seconds, but number.
	get_range    = make(map[string]int)
	sync_range   = make(map[string]int)
	save_removed = make(map[string]int)

	default_port   = setting.getIntValue("Network", "port", 8000)
	dat_port       = setting.getIntValue("Network", "dat_port", 8001)
	max_connection = setting.getIntValue("Network", "max_connection", 20)

	docroot         = setting.getPathValue("Path", "docroot", "./www")
	log_dir         = setting.getPathValue("Path", "log_dir", "./log")
	run_dir         = setting.getPathValue("Path", "run_dir", "../run")
	file_dir        = setting.getPathValue("Path", "file_dir", "../file")
	cache_dir       = setting.getPathValue("Path", "cache_dir", "../cache")
	template_dir    = setting.getPathValue("Path", "template_dir", "../template")
	spam_list       = setting.getPathValue("Path", "spam_list", "../file/spam.txt")
	initnode_list   = setting.getPathValue("Path", "initnode_list", "../file/initnode.txt")
	node_allow_file = setting.getPathValue("Path", "node_allow", "../file/node_allow.txt")
	node_deny_file  = setting.getPathValue("Path", "node_deny", "../file/node_deny.txt")
	apache_docroot  = setting.getPathValue("Path", "apache_docroot", "/var/local/www/shingetsu")
	archive_dir     = setting.getPathValue("Path", "archive_dir", "/var/local/www/archive")

	re_admin          = setting.getStringValue("Gateway", "admin", "^127")
	re_friend         = setting.getStringValue("Gateway", "friend", "^127")
	re_visitor        = setting.getStringValue("Gateway", "visitor", ".")
	server_name       = setting.getStringValue("Gateway", "server_name", "")
	tag_size          = setting.getIntValue("Gateway", "tag_size", 20)
	rss_range         = setting.getIntValue("Gateway", "rss_range", 3*24*60*60)
	top_recent_range  = setting.getIntValue("Gateway", "top_recent_range", 3*24*60*60)
	recent_range      = setting.getIntValue("Gateway", "recent_range", 31*24*60*60)
	record_limit      = setting.getIntValue("Gateway", "record_limit", 2048)
	proxy_destination = setting.getStringValue("Gateway", "proxy_destination", "")
	archive_uri       = setting.getStringValue("Gateway", "archive_uri", "http://archive.shingetsu.info/")
	enable2ch         = setting.getBoolValue("Gateway", "enable_2ch", false)

	motd         = file_dir + "/motd.txt"
	node_file    = run_dir + "/node.txt"
	search_file  = run_dir + "/search.txt"
	update       = run_dir + "/update.txt"
	recent       = run_dir + "/recent.txt"
	client_log   = run_dir + "/client.txt"
	lock         = run_dir + "/lock.txt"
	search_lock  = run_dir + "/touch.txt"
	admin_search = run_dir + "/admintouch.txt"
	admin_sid    = run_dir + "/sid.txt"
	pid          = run_dir + "/pid.txt"
	lookup       = run_dir + "/lookup.txt"
	taglist      = run_dir + "/tag.txt"
	sugtag       = run_dir + "/sugtag.txt"
	read_status  = run_dir + "/readstatus.txt"

	server_cgi  = root_path + "server.cgi"
	client_cgi  = root_path + "client.cgi"
	gateway_cgi = root_path + "gateway.cgi"
	thread_cgi  = root_path + "thread.cgi"
	admin_cgi   = root_path + "admin.cgi"
	xsl         = root_path + "rss1.xsl"

	thread_page_size = setting.getIntValue("Application Thread", "page_size", 50)
	thumbnail_size   = setting.getStringValue("Application Thread", "thumbnail_size", "")
	force_thumbnail  = setting.getBoolValue("Application Thread", "force_thumbnail", false)

	root_index  = setting.getStringValue("Gateway", "root_index", gateway_cgi)
	use_cookie  = true
	save_cookie = 7 * 24 * time.Hour
	// Seconds
	title_limit = 30 // Charactors

	// asis, md5, sha1, sha224, sha256, sha384, or sha512
	cache_hash_method = "asis"
	//others are not implemented for gou for now.

	version = getVersion()

	default_init_node = []string{
		"node.shingetsu.info:8000/server.cgi",
		"pushare.zenno.info:8000/server.cgi",
	}

	flags       []string // It is set by script
	cached_rule = newRegexpList(spam_list)
	absDocroot  string
	queue       = newUpdateQue()
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

	version_file := docroot + "/" + file_dir + "/version.txt"
	f, err := os.Open(version_file)
	if err == nil {
		defer f.Close()
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
		save_record[t] = setting.getIntValue(ctype, "save_record", 0)
		save_size[t] = setting.getIntValue(ctype, "save_size", 1)
		get_range[t] = setting.getIntValue(ctype, "get_range", 31*24*60*60)
		sync_range[t] = setting.getIntValue(ctype, "sync_range", 10*24*60*60)
		save_removed[t] = setting.getIntValue(ctype, "save_removed", 50*24*60*60)
	}
}
