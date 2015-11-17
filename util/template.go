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

package util

import (
	"bytes"
	"fmt"
	"html/template"
	htmlTemplate "html/template"
	"io"
	"log"
	"path"
	"path/filepath"
	textTemplate "text/template"
	"time"
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
	"strEncode":    StrEncode,
	"escape":       Escape,
	"escapeSpace":  EscapeSpace,
	"localtime":    func(stamp int64) string { return time.Unix(stamp, 0).Format("2006-01-02 15:04") },
	"unescapedPrintf": func(format string, a ...interface{}) htmlTemplate.HTML {
		return htmlTemplate.HTML(fmt.Sprintf(format, a))
	},
}

//Ttemplate is for rendering text rss template.
type Ttemplate struct {
	*textTemplate.Template
}

//NewTtemplate adds funcmap to template var and parse files.
func NewTtemplate(templateDir string) *Ttemplate {
	t := &Ttemplate{textTemplate.New("")}
	t.Funcs(textTemplate.FuncMap(funcMap))
	templateFiles := filepath.Join(templateDir, "rss1.txt")
	if IsFile(templateFiles) {
		_, err := t.ParseFiles(templateFiles)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		cont, err := Asset(path.Join("gou_template", "rss1.txt"))
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

//NewHtemplate adds funcmap to template var and parse files.
func NewHtemplate(templateDir string) *Htemplate {
	t := &Htemplate{htmlTemplate.New("")}
	t.Funcs(htmlTemplate.FuncMap(funcMap))
	templateFiles := filepath.Join(templateDir, "*.txt")

	if IsDir(templateDir) {
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
	dir, err := AssetDir("gou_template")
	if err != nil {
		log.Fatal(err)
	}
	for _, a := range dir {
		if _, exist := e[path.Base(a)]; exist {
			continue
		}
		c, err := Asset(path.Join("gou_template", a))
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
func (t *Htemplate) RenderTemplate(file string, st interface{}, wr io.Writer) {
	if err := t.Template.ExecuteTemplate(wr, file, st); err != nil {
		log.Println(err)
	}
}

//ExecuteTemplate executes template and returns it as string.
func (t *Htemplate) ExecuteTemplate(file string, st interface{}) string {
	var doc bytes.Buffer
	t.RenderTemplate(file, st, &doc)
	return doc.String()
}

//RenderTemplate executes rss template and write to wr.
func (t *Ttemplate) RenderTemplate(file string, st interface{}, wr io.Writer) {
	log.Println(file, st, wr)
	if err := t.ExecuteTemplate(wr, file, st); err != nil {
		log.Println(err)
	}
}
