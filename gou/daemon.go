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
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/netutil"

	"github.com/gorilla/handlers"
)

func initPackages(cfg *Config) {
	defaultInitNode := []string{
		"node.shingetsu.info:8000/server.cgi",
		"pushare.zenno.info:8000/server.cgi",
	}
	fmutex := &sync.RWMutex{}
	var myself *node
	htemplate := newHtemplate(cfg.TemplateDir)
	ttemplate := newTtemplate(cfg.TemplateDir)
	cachedRule := newRegexpList(cfg.SpamList)
	nodeAllow := newRegexpList(cfg.NodeAllowFile)
	nodeDeny := newRegexpList(cfg.NodeDenyFile)
	initNode := newConfList(cfg.InitnodeList, defaultInitNode)

	nodeManager := NewNodeManager(&NodeManagerConfig{
		serverName:  cfg.ServerName,
		lookup:      cfg.Lookup(),
		defaultPort: cfg.DefaultPort,
		enableNAT:   cfg.EnableNAT,
		fmutex:      fmutex,
		nodeAllow:   nodeAllow,
		nodeDeny:    nodeDeny,
		myself:      myself,
		initNode:    initNode,
	})
	userTag := newUserTag(&UserTagConfig{
		cacheDir: cfg.CacheDir,
		fmutex:   fmutex,
	})
	suggestedTagTable := newSuggestedTagTable(&SuggestedTagTableConfig{
		tagSize: cfg.TagSize,
		sugtag:  cfg.Sugtag(),
		fmutex:  fmutex,
	})
	recentList := newRecentList(&RecentListConfig{
		recentRange:       cfg.RecentRange,
		tagSize:           cfg.TagSize,
		recent:            cfg.Recent(),
		fmutex:            fmutex,
		nodeManager:       nodeManager,
		suggestedTagTable: suggestedTagTable,
	})
	updateQue := newUpdateQue(&UpdateQueConfig{
		recentList:  recentList,
		nodeManager: nodeManager,
	})
	datakeyTable := newDatakeyTable(&DatakeyTableConfig{
		datakey:    cfg.Datakey(),
		recentList: recentList,
		fmutex:     fmutex,
	})
	datakeyTable.load()

	newAdminCGI = func(w http.ResponseWriter, r *http.Request) (adminCGI, error) {
		return _newAdminCGI(w, r, &AdminCGIConfig{
			adminSID:          cfg.AdminSid(),
			nodeManager:       nodeManager,
			htemplate:         htemplate,
			userTag:           userTag,
			suggestedTagTable: suggestedTagTable,
			recentList:        recentList,
		})
	}

	NewCacheList = func() *cacheList {
		return _newCacheList(&CacheListConfig{
			saveSize:    cfg.SaveSize,
			saveRemoved: cfg.SaveRemoved,
			cacheDir:    cfg.CacheDir,
			saveRecord:  cfg.SaveRecord,
			fmutex:      fmutex,
		})
	}

	NewRecord = func(datfile, idstr string) *record {
		return _newRecord(datfile, idstr, &RecordConfig{
			defaultThumbnailSize: cfg.DefaultThumbnailSize,
			cacheDir:             cfg.CacheDir,
			fmutex:               fmutex,
			cachedRule:           cachedRule,
		})
	}

	NewCGI = func(w http.ResponseWriter, r *http.Request) *cgi {
		return _newCGI(w, r, &CGIConfig{
			fileDir:           cfg.FileDir,
			docroot:           cfg.Docroot,
			maxConnection:     cfg.MaxConnection,
			serverName:        cfg.ServerName,
			reAdminStr:        cfg.ReAdminStr,
			reFriendStr:       cfg.ReFriendStr,
			reVisitorStr:      cfg.ReVisitorStr,
			htemplate:         htemplate,
			userTag:           userTag,
			suggestedTagTable: suggestedTagTable,
		})
	}
	NewGatewayCGI = func(w http.ResponseWriter, r *http.Request) (gatewayCGI, error) {
		return _newGatewayCGI(w, r, &GatewayConfig{
			rssRange:       cfg.RSSRange,
			motd:           cfg.Motd(),
			topRecentRange: cfg.TopRecentRange,
			runDir:         cfg.RunDir,
			serverName:     cfg.ServerName,
			enable2ch:      cfg.Enable2ch,
			recentList:     recentList,
			ttemplate:      ttemplate,
		})
	}
	newMchCGI = func(w http.ResponseWriter, r *http.Request) (mchCGI, error) {
		return _newMchCGI(w, r, &MchConfig{
			motd:         cfg.Motd(),
			filedir:      cfg.FileDir,
			recentList:   recentList,
			datakeyTable: datakeyTable,
			updateQue:    updateQue,
		})
	}
	NewNode = func(nodestr string) *node {
		return _NewNode(nodestr, &NodeConfig{
			nodeAllow: nodeAllow,
			nodeDeny:  nodeDeny,
			myself:    myself,
		})
	}
	NewServerCGI = func(w http.ResponseWriter, r *http.Request) (serverCGI, error) {
		return newServerCGI(w, r, &ServerConfig{
			recentRange: cfg.RecentRange,
			nodeManager: nodeManager,
			initNode:    initNode,
			updateQue:   updateQue,
			recentList:  recentList,
		})
	}
	NewThreadCGI = func(w http.ResponseWriter, r *http.Request) (threadCGI, error) {
		return newThreadCGI(w, r, &ThreadCGIConfig{
			threadPageSize:       cfg.ThreadPageSize,
			defaultThumbnailSize: cfg.DefaultThumbnailSize,
			recordLimit:          cfg.RecordLimit,
			forceThumbnail:       cfg.ForceThumbnail,
			htemplate:            htemplate,
			updateQue:            updateQue,
		})
	}
	go cron(nodeManager, recentList)
}

//StartDaemon setups document root and necessary dirs.
//,rm lock files, save pid, start cron job and a http server.
func StartDaemon(cfg *Config) {
	for _, j := range []string{cfg.RunDir, cfg.CacheDir, cfg.LogDir} {
		if !IsDir(j) {
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
	sm := newLoggingServeMux()
	s := &http.Server{
		Addr:           h,
		Handler:        sm,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	initPackages(cfg)
	adminSetup(sm)
	serverSetup(sm)
	gatewaySetup(sm)
	threadSetup(sm)

	if cfg.Enable2ch {
		fmt.Println("started 2ch interface...")
		mchSetup(sm)
	}
	sm.registerPprof()
	sm.registCompressHandler("/", handleRoot(cfg.Docroot))
	fmt.Println("started daemon and http server...")
	log.Fatal(s.Serve(limitListener))
}

func (s *loggingServeMux) registerPprof() {
	s.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	s.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	s.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	s.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
}

//loggingServerMux is ServerMux with logging
type loggingServeMux struct {
	*http.ServeMux
}

//newLoggingServeMux returns loggingServeMux obj.
func newLoggingServeMux() *loggingServeMux {
	return &loggingServeMux{
		http.NewServeMux(),
	}
}

//ServeHTTP just calles http.ServeMux.ServeHTTP after logging.
func (s *loggingServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.RemoteAddr, r.Method, r.URL.Path, r.Header.Get("User-Agent"), r.Header.Get("Referer"))
	s.ServeMux.ServeHTTP(w, r)
}

//compressHandler returns handlers.CompressHandler to simplfy.
func (s *loggingServeMux) registCompressHandler(path string, fn func(w http.ResponseWriter, r *http.Request)) {
	s.Handle(path, handlers.CompressHandler(http.HandlerFunc(fn)))
}

//handleRoot return handler that handles url not defined other handlers.
//if root, print titles of threads. if not, serve files on disk.
func handleRoot(docroot string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			printTitle(w, r)
			return
		}
		pathOnDisk := path.Join(docroot, r.URL.Path)

		if IsFile(pathOnDisk) {
			http.ServeFile(w, r, pathOnDisk)
			return
		}

		log.Println("not found", r.URL.Path)
		http.NotFound(w, r)
	}
}
