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
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"golang.org/x/net/netutil"

	"github.com/shingetsu-gou/go-nat"
	"github.com/shingetsu-gou/shingetsu-gou/cgi"
	"github.com/shingetsu-gou/shingetsu-gou/mch"
	"github.com/shingetsu-gou/shingetsu-gou/node"
	"github.com/shingetsu-gou/shingetsu-gou/thread"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

func initPackages(cfg *Config) (*node.Manager, *thread.RecentList) {
	externalPort := cfg.DefaultPort
	if cfg.EnableNAT {
		externalPort = setUPnP(cfg.DefaultPort)
	}

	myself := node.NewMyself(externalPort, cgi.ServerURL, cfg.ServerName)

	defaultInitNode := []string{
		"node.shingetsu.info:8000/server.cgi",
		"pushare.zenno.info:8000/server.cgi",
	}
	fmutex := util.NewRWMutex()
	htemplate := util.NewHtemplate(cfg.TemplateDir)
	ttemplate := util.NewTtemplate(cfg.TemplateDir)
	cachedRule := util.NewRegexpList(cfg.SpamList)
	nodeAllow := util.NewRegexpList(cfg.NodeAllowFile)
	nodeDeny := util.NewRegexpList(cfg.NodeDenyFile)
	initNode := util.NewConfList(cfg.InitnodeList, defaultInitNode)

	//nodecfg must be first!
	node.NodeCfg = &node.NodeConfig{
		Myself:    myself,
		NodeAllow: nodeAllow,
		NodeDeny:  nodeDeny,
	}

	nodeManager := node.NewManager(&node.ManagerConfig{
		Lookup:    cfg.Lookup(),
		Fmutex:    fmutex,
		NodeAllow: nodeAllow,
		NodeDeny:  nodeDeny,
		Myself:    myself,
		InitNode:  initNode,
	})
	userTag := thread.NewUserTag(&thread.UserTagConfig{
		CacheDir: cfg.CacheDir,
		Fmutex:   fmutex,
	})
	suggestedTagTable := thread.NewSuggestedTagTable(&thread.SuggestedTagTableConfig{
		TagSize: cfg.TagSize,
		Sugtag:  cfg.Sugtag(),
		Fmutex:  fmutex,
	})
	recentList := thread.NewRecentList(&thread.RecentListConfig{
		RecentRange:       cfg.RecentRange,
		TagSize:           cfg.TagSize,
		Recent:            cfg.Recent(),
		Fmutex:            fmutex,
		NodeManager:       nodeManager,
		SuggestedTagTable: suggestedTagTable,
	})
	updateQue := thread.NewUpdateQue(&thread.UpdateQueConfig{
		RecentList:  recentList,
		NodeManager: nodeManager,
	})
	datakeyTable := mch.NewDatakeyTable(&mch.DatakeyTableConfig{
		Datakey:    cfg.Datakey(),
		RecentList: recentList,
		Fmutex:     fmutex,
	})

	thread.CacheCfg = &thread.CacheConfig{
		CacheDir:          cfg.CacheDir,
		RecordLimit:       cfg.RecordLimit,
		SyncRange:         cfg.SyncRange,
		GetRange:          cfg.GetRange,
		NodeManager:       nodeManager,
		UserTag:           userTag,
		SuggestedTagTable: suggestedTagTable,
		RecentList:        recentList,
		Fmutex:            fmutex,
	}

	cgi.AdminCfg = &cgi.AdminCGIConfig{
		AdminSID:          cfg.AdminSid(),
		NodeManager:       nodeManager,
		Htemplate:         htemplate,
		UserTag:           userTag,
		SuggestedTagTable: suggestedTagTable,
		RecentList:        recentList,
		Myself:            myself,
	}

	thread.CacheListCfg = &thread.CacheListConfig{
		SaveSize:    cfg.SaveSize,
		SaveRemoved: cfg.SaveRemoved,
		CacheDir:    cfg.CacheDir,
		SaveRecord:  cfg.SaveRecord,
		Fmutex:      fmutex,
	}

	thread.RecordCfg = &thread.RecordConfig{
		DefaultThumbnailSize: cfg.DefaultThumbnailSize,
		CacheDir:             cfg.CacheDir,
		Fmutex:               fmutex,
		CachedRule:           cachedRule,
		RecordLimit:          cfg.RecordLimit,
	}

	cgi.CGICfg = &cgi.CGIConfig{
		FileDir:           cfg.FileDir,
		Docroot:           cfg.Docroot,
		MaxConnection:     cfg.MaxConnection,
		ServerName:        cfg.ServerName,
		ReAdminStr:        cfg.ReAdminStr,
		ReFriendStr:       cfg.ReFriendStr,
		ReVisitorStr:      cfg.ReVisitorStr,
		Htemplate:         htemplate,
		UserTag:           userTag,
		SuggestedTagTable: suggestedTagTable,
	}
	cgi.GatewayCfg = &cgi.GatewayConfig{
		RSSRange:       cfg.RSSRange,
		Motd:           cfg.Motd(),
		TopRecentRange: cfg.TopRecentRange,
		RunDir:         cfg.RunDir,
		ServerName:     cfg.ServerName,
		Enable2ch:      cfg.Enable2ch,
		RecentList:     recentList,
		Ttemplate:      ttemplate,
	}
	cgi.MchCfg = &cgi.MchConfig{
		Motd:         cfg.Motd(),
		RecentList:   recentList,
		DatakeyTable: datakeyTable,
		UpdateQue:    updateQue,
	}

	cgi.ServerCfg = &cgi.ServerConfig{
		RecentRange: cfg.RecentRange,
		NodeManager: nodeManager,
		InitNode:    initNode,
		UpdateQue:   updateQue,
		RecentList:  recentList,
	}
	cgi.ThreadCfg = &cgi.ThreadCGIConfig{
		ThreadPageSize:       cfg.ThreadPageSize,
		DefaultThumbnailSize: cfg.DefaultThumbnailSize,
		RecordLimit:          cfg.RecordLimit,
		ForceThumbnail:       cfg.ForceThumbnail,
		Htemplate:            htemplate,
		UpdateQue:            updateQue,
	}
	datakeyTable.Load()
	return nodeManager, recentList

}

//setUPnP gets external port by upnp and return external port.
//returns defaultPort if failed.
func setUPnP(defaultPort int) int {
	nt, err := nat.NewNetStatus()
	if err != nil {
		log.Println(err)
	} else {
		m, err := nt.LoopPortMapping("tcp", defaultPort, "shingetsu-gou", 10*time.Minute)
		if err != nil {
			log.Println(err)
		} else {
			return m.ExternalPort
		}
	}
	return defaultPort
}

//StartDaemon setups document root and necessary dirs.
//And save pid, start cron job and a http server.
func StartDaemon(cfg *Config) {
	for _, j := range []string{cfg.RunDir, cfg.CacheDir, cfg.LogDir} {
		if !util.IsDir(j) {
			err := os.Mkdir(j, 0755)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	p := os.Getpid()
	err := ioutil.WriteFile(cfg.PID(), []byte(strconv.Itoa(p)), 0666)
	if err != nil {
		log.Println(err)
	}

	h := fmt.Sprintf("0.0.0.0:%d", cfg.DefaultPort)
	listener, err := net.Listen("tcp", h)
	if err != nil {
		log.Fatalln(err)
	}
	limitListener := netutil.LimitListener(listener, cfg.MaxConnection)
	sm := cgi.NewLoggingServeMux()
	s := &http.Server{
		Addr:           h,
		Handler:        sm,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	nm, rl := initPackages(cfg)
	go cron(nm, rl, cfg.HeavyMoon)

	cgi.AdminSetup(sm)
	cgi.ServerSetup(sm)
	cgi.GatewaySetup(sm)
	cgi.ThreadSetup(sm)

	if cfg.Enable2ch {
		fmt.Println("started 2ch interface...")
		cgi.MchSetup(sm)
	}
	if cfg.EnableProf {
		sm.RegisterPprof()
	}
	sm.RegistCompressHandler("/", handleRoot(cfg.Docroot))
	fmt.Println("started daemon and http server...")
	log.Fatal(s.Serve(limitListener))
}

//handleRoot return handler that handles url not defined other handlers.
//if root, print titles of threads. if not, serve files on disk.
func handleRoot(docroot string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			cgi.PrintTitle(w, r)
			return
		}
		pathOnDisk := path.Join(docroot, r.URL.Path)

		if util.IsFile(pathOnDisk) {
			http.ServeFile(w, r, pathOnDisk)
			return
		}

		log.Println("not found", r.URL.Path)
		http.NotFound(w, r)
	}
}

//Sakurifice makes cache be compatible with saku.
//i.e. makes body dir ,attach dir and dat.stat in under cache dir.
func Sakurifice(cfg *Config) {
	initPackages(cfg)
	f := path.Join(cfg.RunDir, "tag.txt")
	if !util.IsFile(f) {
		writeFile(f, []byte{})
	}

	cl := thread.NewCacheList()
	log.Println("# of cache", cl.Len())
	for _, ca := range cl.Caches {
		log.Println("processing", ca.Datfile)
		f := path.Join(ca.Datpath(), "dat.stat")
		writeFile(f, []byte(ca.Datfile))
		bodypath := path.Join(ca.Datpath(), "body")
		mkdir(bodypath)
		attachPath := path.Join(ca.Datpath(), "attach")
		mkdir(attachPath)
		recs := ca.LoadRecords()
		for _, rec := range recs {
			if err := rec.Load(); err != nil {
				log.Fatal(err)
			}
			at := rec.GetBodyValue("attach", "")
			sign := rec.GetBodyValue("sign", "")
			pubkey := rec.GetBodyValue("pubkey", "")
			if at != "" || sign != "" || pubkey != "" {
				f := path.Join(bodypath, rec.Idstr())
				writeFile(f, []byte(rec.BodyString()))
			}
			if at != "" {
				decoded, err := base64.StdEncoding.DecodeString(at)
				if err != nil {
					log.Fatal(err)
				}
				f = rec.AttachPath("")
				writeFile(f, decoded)
				if cfg.DefaultThumbnailSize == "" {
					continue
				}
				decoded = util.MakeThumbnail(decoded, rec.GetBodyValue("suffix", ""), cfg.DefaultThumbnailSize)
				if decoded != nil {
					f = rec.AttachPath(cfg.DefaultThumbnailSize)
					writeFile(f, decoded)
				}
			}
		}
	}
}

func mkdir(path string) {
	if !util.IsDir(path) {
		err := os.Mkdir(path, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func writeFile(fname string, data []byte) {
	err := ioutil.WriteFile(fname, data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
