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

package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/shingetsu-gou/shingetsu-gou/gou"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//Version is one of Gou. it shoud be overwritten when building on travis.
var VERSION = "unstable"

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(os.Stdout)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "P2P anonymous BBS shinGETsu Gou %s\n", VERSION)
		fmt.Fprintf(os.Stderr, "%s <options>\n", os.Args[0])
		flag.PrintDefaults()
	}
}

//expandAssets expands files in /file in an Assets if not exist in disk.
func expandAssets(fileDir, templateDir, docroot string) {
	dir, err := util.AssetDir("file")
	if err != nil {
		log.Fatal(err)
	}
	for _, fname := range dir {
		if fname == "message-ja.txt" || fname == "message-en.txt" {
			continue
		}
		fnameDisk := path.Join(fileDir, fname)
		fnameDisk = filepath.FromSlash(fnameDisk)
		if util.IsFile(fnameDisk) {
			continue
		}
		log.Println("expanding", fnameDisk)
		d := filepath.Dir(fnameDisk)
		if !util.IsDir(d) {
			err := os.MkdirAll(d, 0755)
			if err != nil {
				log.Fatal(err, d)
			}
		}
		c, err := util.Asset(path.Join("file", fname))
		if err != nil {
			log.Fatal(err)
		}
		err = ioutil.WriteFile(fnameDisk, c, 0644)
		if err != nil {
			log.Fatal(err)
		}
	}
}

//setLogger setups logger. whici outputs nothing, or file , or file and stdout
func setLogger(printLog, isSilent bool, logDir string) {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	l := &lumberjack.Logger{
		Filename:   path.Join(logDir, "gou.log"),
		MaxSize:    5, // megabytes
		MaxBackups: 10,
		MaxAge:     28, //days
	}
	switch {
	case isSilent:
		fmt.Println("logging is discarded")
		log.SetOutput(ioutil.Discard)
	case printLog:
		fmt.Println("outputs logs to stdout and ", logDir)
		m := io.MultiWriter(os.Stdout, l)
		log.SetOutput(m)
	default:
		fmt.Println("output logs to ", logDir)
		log.SetOutput(l)
	}
}

//setupDirectories makes necessary dirs.
func setupDirectories(cfg *gou.Config) {
	for _, j := range []string{cfg.RunDir, cfg.CacheDir, cfg.LogDir} {
		if !util.IsDir(j) {
			err := os.MkdirAll(j, 0755)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func main() {
	fmt.Println("starting Gou...")
	VERSION = strings.Trim(VERSION, "'")
	cfg := gou.NewConfig()
	var printLog, isSilent, sakurifice bool
	flag.BoolVar(&printLog, "verbose", false, "print logs")
	flag.BoolVar(&printLog, "v", false, "print logs")
	flag.BoolVar(&isSilent, "silent", false, "suppress logs")
	flag.BoolVar(&sakurifice, "sakurifice", false, "makes caches compatible with saku")
	flag.Parse()
	setupDirectories(cfg)
	setLogger(printLog, isSilent, cfg.LogDir)
	expandAssets(cfg.FileDir, cfg.TemplateDir, cfg.Docroot)
	if sakurifice {
		gou.Sakurifice(cfg)
	} else {
		gou.StartDaemon(cfg, VERSION)
	}
}
