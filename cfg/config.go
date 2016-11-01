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

package cfg

import (
	"errors"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/util"
	"gopkg.in/ini.v1"
)

const (
	//Disconnected represents mynode is disconnected.
	Disconnected = iota
	//Port0 represents mynode is behind NAT and not opened.
	Port0
	//UPnP represents mynode is opned by uPnP.
	UPnP
	//Normal represents port was opened manually.
	Normal
)

const (
	//AdminURL is the url to admin.cgi
	AdminURL = "/admin.cgi"
	//GatewayURL is the url to gateway.cgi
	GatewayURL = "/gateway.cgi"
	//ThreadURL is the url to thread.cgi
	ThreadURL = "/thread.cgi"
	//ServerURL is the url to server.cgi
	ServerURL = "/server.cgi"
)

//data Errors.
var (
	ErrSpam = errors.New("this is spam")
	ErrGet  = errors.New("cannot get data")
)

var (
	defaultInitNode = []string{
		"node.shingetsu.info:8000/server.cgi",
	}
	//InitNode is initiali nodes.
	InitNode *util.ConfList
)

//cwd represents current working dir.
//which should be the result of getFilesDir()  at android.
var cwd = "."
var android = false

//SetAndroid sets that I'm in android
//and writable path as cwd.
//all paths are ignored when in android.
func SetAndroid(path string) {
	android = true
	cwd = path
}

//config params
var (
	Docroot     string
	LogDir      string
	RunDir      string
	FileDir     string
	TemplateDir string

	NetworkMode          int //port_opened,relay,upnp
	SaveRecord           int64
	SaveSize             int // It is not seconds, but number.
	GetRange             int64
	SyncRange            int64
	SaveRemoved          int64
	DefaultPort          int //DefaultPort is listening port
	MaxConnection        int
	SpamList             string
	InitnodeList         string
	NodeAllowFile        string
	NodeDenyFile         string
	ReAdminStr           string
	ReFriendStr          string
	ReVisitorStr         string
	ServerName           string
	TagSize              int
	RSSRange             int64
	TopRecentRange       int64
	RecentRange          int64
	RecordLimit          int
	ThreadPageSize       int
	DefaultThumbnailSize string
	Enable2ch            bool
	ForceThumbnail       bool
	EnableProf           bool
	HeavyMoon            bool
	EnableEmbed          bool
)

//SuffixTXT is suffix of text files.
var SuffixTXT = "txt"

// asis, md5, sha1, sha224, sha256, sha384, or sha512
//	cache_hash_method = "asis"
//others are not implemented for gou for now.

//Version is one of Gou. it shoud be overwritten when building on travis.
var Version = "unstable"

//getIntValue gets int value from ini file.
func getIntValue(i *ini.File, section, key string, vdefault int) int {
	return i.Section(section).Key(key).MustInt(vdefault)
}

//getInt64Value gets int value from ini file.
func getInt64Value(i *ini.File, section, key string, vdefault int64) int64 {
	return i.Section(section).Key(key).MustInt64(vdefault)
}

//getStringValue gets string from ini file.
func getStringValue(i *ini.File, section, key string, vdefault string) string {
	return i.Section(section).Key(key).MustString(vdefault)
}

//getBoolValue gets bool value from ini file.
func getBoolValue(i *ini.File, section, key string, vdefault bool) bool {
	return i.Section(section).Key(key).MustBool(vdefault)
}

//getPathValue gets path from ini file.
func getRelativePathValue(i *ini.File, section, key, vdefault, docroot string) string {
	p := i.Section(section).Key(key).MustString(vdefault)
	h := p
	if !path.IsAbs(p) {
		h = path.Join(docroot, p)
	}
	return filepath.FromSlash(h)
}

//getPathValue gets path from ini file.
func getPathValue(i *ini.File, section, key string, vdefault string) string {
	p := i.Section(section).Key(key).MustString(vdefault)
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	h := p
	if !path.IsAbs(p) {
		h = path.Join(wd, p)
	}
	return filepath.FromSlash(h)
}

//Parse makes  config vars from the ini files and returns it.
func Parse() {
	files := []string{filepath.Join(cwd, "file", "saku.ini"), "/usr/local/etc/saku/saku.ini", "/etc/saku/saku.ini"}
	usr, err := user.Current()
	if err == nil {
		files = append(files, filepath.Join(usr.HomeDir, ".saku", "saku.ini"))
	}
	i := ini.Empty()
	for _, f := range files {
		fs, err := os.Stat(f)
		if err == nil && !fs.IsDir() {
			log.Println("loading config from", f)
			if err := i.Append(f); err != nil {
				log.Fatal("cannot load ini files", f, "ignored")
			}
		}
	}
	initVariables(i)
	InitNode = util.NewConfList(InitnodeList, defaultInitNode)
}

//initVariables initializes some global and map vars.
func initVariables(i *ini.File) {
	DefaultPort = getIntValue(i, "Network", "port", 8000)
	networkModeStr := getStringValue(i, "Network", "mode", "port_opened") //port_opened,upnp,relay
	switch networkModeStr {
	case "port_opened":
		NetworkMode = Normal
	case "upnp":
		NetworkMode = UPnP
	default:
		log.Fatal("cannot understand mode", networkModeStr)
	}

	if !android {
		Docroot = getPathValue(i, "Path", "docroot", "./www")                                     //path from cwd
		RunDir = getRelativePathValue(i, "Path", "run_dir", "../run", Docroot)                    //path from docroot
		FileDir = getRelativePathValue(i, "Path", "file_dir", "../file", Docroot)                 //path from docroot
		TemplateDir = getRelativePathValue(i, "Path", "template_dir", "../gou_template", Docroot) //path from docroot
		LogDir = getPathValue(i, "Path", "log_dir", "./log")                                      //path from cwd
		SpamList = getRelativePathValue(i, "Path", "spam_list", "../file/spam.txt", Docroot)
		InitnodeList = getRelativePathValue(i, "Path", "initnode_list", "../file/initnode.txt", Docroot)
		NodeAllowFile = getRelativePathValue(i, "Path", "node_allow", "../file/node_allow.txt", Docroot)
		NodeDenyFile = getRelativePathValue(i, "Path", "node_deny", "../file/node_deny.txt", Docroot)
	} else {
		Docroot = filepath.Join(cwd, "www")
		RunDir = filepath.Join(cwd, "run")
		FileDir = filepath.Join(cwd, "file")
		TemplateDir = filepath.Join(cwd, "gou_template")
		LogDir = filepath.Join(cwd, "log")
		SpamList = filepath.Join(cwd, "file", "spam.txt")
		InitnodeList = filepath.Join(cwd, "file", "initnode.txt")
		NodeAllowFile = filepath.Join(cwd, "file", "node_allow.txt")
		NodeDenyFile = filepath.Join(cwd, "file", "node_deny.txt")
	}
	MaxConnection = getIntValue(i, "Network", "max_connection", 100)
	ReAdminStr = getStringValue(i, "Gateway", "admin", "^(127|\\[::1\\])")
	ReFriendStr = getStringValue(i, "Gateway", "friend", "^(127|\\[::1\\])")
	ReVisitorStr = getStringValue(i, "Gateway", "visitor", ".")
	ServerName = getStringValue(i, "Gateway", "server_name", "")
	TagSize = getIntValue(i, "Gateway", "tag_size", 20)
	RSSRange = getInt64Value(i, "Gateway", "rss_range", 3*24*60*60)
	TopRecentRange = getInt64Value(i, "Gateway", "top_recent_range", 3*24*60*60)
	RecentRange = getInt64Value(i, "Gateway", "recent_range", 31*24*60*60)
	RecordLimit = getIntValue(i, "Gateway", "record_limit", 2048)
	Enable2ch = getBoolValue(i, "Gateway", "enable_2ch", false)
	EnableProf = getBoolValue(i, "Gateway", "enable_prof", false)
	HeavyMoon = getBoolValue(i, "Gateway", "moonlight", false)
	EnableEmbed = getBoolValue(i, "Gateway", "enable_embed", true)
	ThreadPageSize = getIntValue(i, "Application Thread", "page_size", 50)
	DefaultThumbnailSize = getStringValue(i, "Application Thread", "thumbnail_size", "")
	ForceThumbnail = getBoolValue(i, "Application Thread", "force_thumbnail", false)
	ctype := "Application Thread"
	SaveRecord = getInt64Value(i, ctype, "save_record", 0)
	SaveSize = getIntValue(i, ctype, "save_size", 1)
	GetRange = getInt64Value(i, ctype, "get_range", 31*24*60*60)
	if GetRange > time.Now().Unix() {
		log.Fatal("get_range is too big")
	}
	SyncRange = getInt64Value(i, ctype, "sync_range", 10*24*60*60)
	if SyncRange > time.Now().Unix() {
		log.Fatal("sync_range is too big")
	}
	SaveRemoved = getInt64Value(i, ctype, "save_removed", 50*24*60*60)
	if SaveRemoved > time.Now().Unix() {
		log.Fatal("save_removed is too big")
	}

	if SyncRange == 0 {
		SaveRecord = 0
	}

	if SaveRemoved != 0 && SaveRemoved <= SyncRange {
		SyncRange = SyncRange + 1
	}

}

//Motd returns path to motd.txt
func Motd() string {
	return FileDir + "/motd.txt"
}

//PID returns path to pid.txt
func PID() string {
	return RunDir + "/pid.txt"
}
