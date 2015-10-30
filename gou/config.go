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
	"log"
	"os"
	"os/user"
	"path"
	"sync"
	"time"

	"gopkg.in/ini.v1"
)

const (
	//Version is one of Gou. it shoud be overwritten when building on travis.
	Version = "Git/unstable"
)

//Get Gou version for useragent and servername.
func getVersion() string {
	return "shinGETsu/0.7 (Gou/" + Version + ")"
}

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
	return h
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
	return h
}

//Config contains params ini file.
type Config struct {
	SaveRecord           int64
	SaveSize             int // It is not seconds, but number.
	GetRange             int64
	SyncRange            int64
	SaveRemoved          int64
	DefaultPort          int //DefaultPort is listening port
	MaxConnection        int
	Docroot              string
	LogDir               string
	RunDir               string
	FileDir              string
	CacheDir             string
	TemplateDir          string
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
	EnableNAT            bool //EnableNAT is enabled if you want to use nat.
	ForceThumbnail       bool

	Fmutex sync.RWMutex //Fmutex is rwmutex for syncing the disk

	// asis, md5, sha1, sha224, sha256, sha384, or sha512
	//	cache_hash_method = "asis"
	//others are not implemented for gou for now.
}

//NewConfig make a struct including configs from the ini files and returns it.
func NewConfig() *Config {
	files := []string{"file/saku.ini", "/usr/local/etc/saku/saku.ini", "/etc/saku/saku.ini"}
	usr, err := user.Current()
	if err == nil {
		files = append(files, usr.HomeDir+"/.saku/saku.ini")
	}
	i := ini.Empty()
	for _, f := range files {
		if IsFile(f) {
			if err := i.Append(f); err != nil {
				log.Fatal("cannot load ini files", f, "ignored")
			}
		} else {
			log.Println(f, "not found, ignored.")
		}
	}
	c := &Config{}
	c.initVariables(i)
	return c
}

//initVariables initializes some global and map vars.
func (c *Config) initVariables(i *ini.File) {
	c.DefaultPort = getIntValue(i, "Network", "port", 8000)
	c.MaxConnection = getIntValue(i, "Network", "max_connection", 20)
	c.Docroot = getPathValue(i, "Path", "docroot", "./www")                                       //path from cwd
	c.RunDir = getRelativePathValue(i, "Path", "run_dir", "../run", c.Docroot)                    //path from docroot
	c.FileDir = getRelativePathValue(i, "Path", "file_dir", "../file", c.Docroot)                 //path from docroot
	c.CacheDir = getRelativePathValue(i, "Path", "cache_dir", "../cache", c.Docroot)              //path from docroot
	c.TemplateDir = getRelativePathValue(i, "Path", "template_dir", "../gou_template", c.Docroot) //path from docroot
	c.SpamList = getRelativePathValue(i, "Path", "spam_list", "../file/spam.txt", c.Docroot)
	c.InitnodeList = getRelativePathValue(i, "Path", "initnode_list", "../file/initnode.txt", c.Docroot)
	c.NodeAllowFile = getRelativePathValue(i, "Path", "node_allow", "../file/node_allow.txt", c.Docroot)
	c.NodeDenyFile = getRelativePathValue(i, "Path", "node_deny", "../file/node_deny.txt", c.Docroot)
	c.ReAdminStr = getStringValue(i, "Gateway", "admin", "^(127|\\[::1\\])")
	c.ReFriendStr = getStringValue(i, "Gateway", "friend", "^(127|\\[::1\\])")
	c.ReVisitorStr = getStringValue(i, "Gateway", "visitor", ".")
	c.ServerName = getStringValue(i, "Gateway", "server_name", "")
	c.TagSize = getIntValue(i, "Gateway", "tag_size", 20)
	c.RSSRange = getInt64Value(i, "Gateway", "rss_range", 3*24*60*60)
	c.TopRecentRange = getInt64Value(i, "Gateway", "top_recent_range", 3*24*60*60)
	c.RecentRange = getInt64Value(i, "Gateway", "recent_range", 31*24*60*60)
	c.RecordLimit = getIntValue(i, "Gateway", "record_limit", 2048)
	c.Enable2ch = getBoolValue(i, "Gateway", "enable_2ch", false)
	c.EnableNAT = getBoolValue(i, "Gateway", "enable_nat", false)
	c.LogDir = getPathValue(i, "Path", "log_dir", "./log") //path from cwd
	c.ThreadPageSize = getIntValue(i, "Application Thread", "page_size", 50)
	c.DefaultThumbnailSize = getStringValue(i, "Application Thread", "thumbnail_size", "")
	c.ForceThumbnail = getBoolValue(i, "Application Thread", "force_thumbnail", false)
	ctype := "Application thread"
	c.SaveRecord = getInt64Value(i, ctype, "save_record", 0)
	c.SaveSize = getIntValue(i, ctype, "save_size", 1)
	c.GetRange = getInt64Value(i, ctype, "get_range", 31*24*60*60)
	if c.GetRange > time.Now().Unix() {
		log.Fatal("get_range is too big")
	}
	c.SyncRange = getInt64Value(i, ctype, "sync_range", 10*24*60*60)
	if c.SyncRange > time.Now().Unix() {
		log.Fatal("sync_range is too big")
	}
	c.SaveRemoved = getInt64Value(i, ctype, "save_removed", 50*24*60*60)
	if c.SaveRemoved > time.Now().Unix() {
		log.Fatal("save_removed is too big")
	}

	if c.SyncRange == 0 {
		c.SaveRecord = 0
	}

	if c.SaveRemoved != 0 && c.SaveRemoved <= c.SyncRange {
		c.SyncRange = c.SyncRange + 1
	}

}

//Motd returns path to motd.txt
func (c *Config) Motd() string {
	return c.FileDir + "/motd.txt"
}

//Recent returns path to recent.txt
func (c *Config) Recent() string {
	return c.RunDir + "/recent.txt"
}

//AdminSid returns path to sid.txt
func (c *Config) AdminSid() string {
	return c.RunDir + "/sid.txt"
}

//PID returns path to pid.txt
func (c *Config) PID() string {
	return c.RunDir + "/pid.txt"
}

//Lookup returns path to lookup.txt
func (c *Config) Lookup() string {
	return c.RunDir + "/lookup.txt"
}

//Sugtag returns path to sugtag.txt
func (c *Config) Sugtag() string {
	return c.RunDir + "/sugtag.txt"
}

//Datakey returns path to datakey.txt
func (c *Config) Datakey() string {
	return c.RunDir + "/datakey.txt"
}
