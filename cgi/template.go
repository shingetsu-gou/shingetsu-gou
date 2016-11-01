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

package cgi

import (
	"html/template"
	htmlTemplate "html/template"
	"io"
	"log"
	"path"
	"path/filepath"
	textTemplate "text/template"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

var (
	//tmpH is template for html.
	tmpH *Htemplate
	//tmpT is template for text(rss).
	tmpT *Ttemplate
)

var funcMap = map[string]interface{}{
	"add":          func(a, b int) int { return a + b },
	"sub":          func(a, b int) int { return a - b },
	"mul":          func(a, b int) int { return a * b },
	"div":          func(a, b int) int { return a / b },
	"toMB":         func(a int) float64 { return float64(a) / (1024 * 1024) },
	"toKB":         func(a int) float64 { return float64(a) / (1024) },
	"toInt":        func(a int64) int { return int(a) },
	"stopEscaping": func(a string) template.HTML { return template.HTML(a) },
	"strEncode":    util.StrEncode,
	"escape":       util.Escape,
	"escapeSpace":  util.EscapeSpace,
	"localtime":    func(stamp int64) string { return time.Unix(stamp, 0).Format("2006-01-02 15:04") },
}

//Ttemplate is for rendering text rss template.
type Ttemplate struct {
	*textTemplate.Template
}

//newTtemplate adds funcmap to template var and parse files.
func newTtemplate(templateDir string) *Ttemplate {
	t := &Ttemplate{textTemplate.New("")}
	t.Funcs(textTemplate.FuncMap(funcMap))
	templateFiles := filepath.Join(templateDir, "rss1.txt")
	if util.IsFile(templateFiles) {
		_, err := t.ParseFiles(templateFiles)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		cont, err := util.Asset(path.Join("gou_template", "rss1.txt"))
		if err != nil {
			log.Fatal(err)
		}
		_, err = t.Parse(string(cont))
		if err != nil {
			log.Fatal(err)
		}
	}
	return t
}

//Htemplate is for rendering html stuff.
type Htemplate struct {
	*htmlTemplate.Template
}

//newHtemplate adds funcmap to template var and parse files.
func newHtemplate(templateDir string) *Htemplate {
	t := &Htemplate{htmlTemplate.New("")}
	t.Funcs(htmlTemplate.FuncMap(funcMap))
	templateFiles := filepath.Join(templateDir, "*.txt")

	if util.IsDir(templateDir) {
		_, err := t.ParseGlob(templateFiles)
		if err != nil {
			log.Fatal(err)
		}
	}
	mat, err := filepath.Glob(templateFiles)
	if err != nil {
		log.Fatal(err)
	}
	e := make(map[string]struct{})
	for _, m := range mat {
		e[filepath.Base(m)] = struct{}{}
	}
	dir, err := util.AssetDir("gou_template")
	if err != nil {
		log.Fatal(err)
	}
	for _, a := range dir {
		if _, exist := e[path.Base(a)]; exist {
			continue
		}
		c, err := util.Asset(path.Join("gou_template", a))
		if err != nil {
			log.Fatal(err)
		}

		if _, err := t.Parse(string(c)); err != nil {
			log.Fatal(err)
		}
	}

	return t
}

//RenderTemplate executes template and write to wr.
func RenderTemplate(file string, st interface{}, wr io.Writer) {
	if tmpH == nil {
		tmpH = newHtemplate(cfg.TemplateDir)
	}
	if err := tmpH.ExecuteTemplate(wr, file, st); err != nil {
		log.Println(err)
	}
}

//RenderRSS executes rss template and write to wr.
func RenderRSS(st interface{}, wr io.Writer) {
	if tmpT == nil {
		tmpT = newTtemplate(cfg.TemplateDir)
	}
	if err := tmpT.ExecuteTemplate(wr, "rss1", st); err != nil {
		log.Println(err)
	}
}
