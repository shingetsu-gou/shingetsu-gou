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
	"time"

	"golang.org/x/net/netutil"

	"github.com/gorilla/handlers"
)

//SetupDaemon setups document root and necessary dirs.
func SetupDaemon(cfg *Config) {
	for _, j := range []string{cfg.RunDir, cfg.CacheDir, cfg.LogDir} {
		if !IsDir(j) {
			err := os.Mkdir(j, 0755)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	NodeSetup(cfg.EnableNAT, cfg.DefaultPort, cfg.InitnodeList)
	CGISetup(cfg.MaxConnection)
	TemplateSetup(cfg.TemplateDir)
	DatakeySetup(cfg.RunDir)
	TagSetup(cfg.Sugtag(), cfg.CacheDir)
	QueSetup()
	RecentListSetup(cfg.Recent())
	RecordSetup(cfg.SpamList)
}

//StartDaemon rm lock files, save pid, start cron job and a http server.
func StartDaemon(cfg *Config) {
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

	go cron()
	sm.registerPprof()
	sm.registCompressHandler("/", handleRoot(cfg.Docroot))
	adminSetup(sm, cfg.AdminSid())
	serverSetup(sm)
	gatewaySetup(sm)
	threadSetup(sm)

	if cfg.Enable2ch {
		fmt.Println("started 2ch interface...")
		mchSetup(sm)
	}
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
