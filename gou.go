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
	"github.com/shingetsu-gou/shingetsu-gou/node"
)

const (
	//Version is one of Gou. it shoud be overwritten when building on travis.
	Version = "Git/unstable"
)

func init() {
	node.Version = Version
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(os.Stdout)
}

//expandAssets expands all files in a Assets if not exist in disk.
func expandAssets(fileDir, templateDir, docroot string) {
	dname := map[string]string{
		"file":         fileDir,
		"gou_template": templateDir,
		"www":          docroot,
	}

	for _, fname := range AssetNames() {
		dir := filepath.SplitList(fname)[0]
		fnameDisk := strings.Replace(fname, dir, dname[dir], 1)
		if gou.IsFile(fnameDisk) {
			continue
		}
		log.Println("expanding", fnameDisk)
		path, _ := path.Split(fnameDisk)
		if !gou.IsDir(path) {
			err := os.MkdirAll(path, 0755)
			if err != nil {
				log.Fatal(err)
			}
		}
		c, err := Asset(fname)
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
	l := &lumberjack.Logger{
		Filename:   path.Join(logDir, "gou.log"),
		MaxSize:    1, // megabytes
		MaxBackups: 2,
		MaxAge:     28, //days
	}
	fmt.Println(logDir)
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

func main() {
	log.Println("starting Gou...")

	cfg := gou.NewConfig()
	var printLog, isSilent bool
	flag.BoolVar(&printLog, "verbose", false, "print logs")
	flag.BoolVar(&printLog, "v", false, "print logs")
	flag.BoolVar(&isSilent, "silent", false, "suppress logs")
	flag.Parse()
	setLogger(printLog, isSilent, cfg.LogDir)

	expandAssets(cfg.FileDir, cfg.TemplateDir, cfg.Docroot)
	gou.StartDaemon(cfg)
}
