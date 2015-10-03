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
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/shinGETsu-gou/shingetsu-gou/gou/mch"
	"gopkg.in/natefinch/lumberjack.v2"
)

func SetLogger(printLog, isSilent bool) {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	l := &lumberjack.Logger{
		Filename:   log_dir + "gou.log",
		MaxSize:    1, // megabytes
		MaxBackups: 2,
		MaxAge:     28, //days
	}
	switch {
	case isSilent:
		log.SetOutput(ioutil.Discard)
	case printLog:
		m := io.MultiWriter(os.Stdout, l)
		log.SetOutput(m)
	default:
		log.SetOutput(l)
	}
}

func SetupDaemon() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	absDocroot = path.Join(dir, docroot)
	for _, j := range []string{run_dir, cache_dir} {
		i := path.Join(docroot, j)
		if !isDir(i) {
			os.Mkdir(i, 07555)
		}
	}
}

func StartDaemon() {
	for _, lock := range []string{lock, search_lock, admin_search} {
		l := path.Join(docroot, lock)
		if !isFile(l) {
			err := os.Remove(l)
			if err != nil {
				log.Println(err)
			}
		}
	}
	pidfile := path.Join(docroot, pid)
	pid := os.Getpid()
	err := ioutil.WriteFile(pidfile, []byte(strconv.Itoa(pid)), 0666)
	if err != nil {
		log.Println(err)
	}

	sm := http.NewServeMux()
	s := &http.Server{
		Addr:           "0.0.0.0:" + strconv.Itoa(default_port),
		Handler:        sm,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go cron()
	adminSetup(sm)
	serverSetup(sm)
	gatewaySetup(sm)
	threadSetup(sm)

	if enable2ch {
		smm := http.NewServeMux()
		ss := &http.Server{
			Addr:           "0.0.0.0:" + strconv.Itoa(dat_port),
			Handler:        smm,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
		mch.Setup(smm)
		go log.Fatal(ss.ListenAndServe())
	}
	sm.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	})
	
	log.Fatal(s.ListenAndServe())
}
