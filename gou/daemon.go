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
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/net/netutil"

	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/cgi"
	"github.com/shingetsu-gou/shingetsu-gou/cgi/admin"
	"github.com/shingetsu-gou/shingetsu-gou/cgi/gateway"
	"github.com/shingetsu-gou/shingetsu-gou/cgi/mch"
	"github.com/shingetsu-gou/shingetsu-gou/cgi/server"
	"github.com/shingetsu-gou/shingetsu-gou/cgi/thread"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//StartDaemon setups saves pid, start cron job and a http server.
func StartDaemon() {
	p := os.Getpid()
	err := ioutil.WriteFile(cfg.PID(), []byte(strconv.Itoa(p)), 0666)
	if err != nil {
		log.Fatal(err)
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
		ReadTimeout:    3 * time.Minute,
		WriteTimeout:   3 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	go cron()

	admin.Setup(sm)
	server.Setup(sm)
	gateway.Setup(sm)
	thread.Setup(sm)

	if cfg.Enable2ch {
		fmt.Println("started 2ch interface...")
		mch.Setup(sm)
	}
	if cfg.EnableProf {
		sm.RegisterPprof()
	}
	sm.RegistCompressHandler("/", handleRoot())
	fmt.Println("started daemon and http server...")
	log.Fatal(s.Serve(limitListener))
}

//handleRoot return handler that handles url not defined other handlers.
//if root, print titles of threads. if not, serve files on disk.
func handleRoot() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			gateway.PrintTitle(w, r)
			return
		}
		pathOnDisk := filepath.Join(cfg.Docroot, r.URL.Path)

		if util.IsFile(pathOnDisk) {
			http.ServeFile(w, r, pathOnDisk)
			return
		}
		pathOnAsset := path.Join("www", r.URL.Path)
		if c, err := util.Asset(pathOnAsset); err == nil {
			i, err := util.AssetInfo(pathOnAsset)
			if err != nil {
				log.Fatal(err)
			}
			reader := bytes.NewReader(c)
			http.ServeContent(w, r, path.Base(r.URL.Path), i.ModTime(), reader)
			return
		}

		log.Println("not found", r.URL.Path)
		http.NotFound(w, r)
	}
}
