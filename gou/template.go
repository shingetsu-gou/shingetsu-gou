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
	"html/template"
	htmlTemplate "html/template"
	"io"
	"log"
	textTemplate "text/template"
	"time"
)

var (
	ttemplates = textTemplate.New("")
	htemplates = htmlTemplate.New("")
)

//TemplateSetup adds funcmap to template var and parse files.
func TemplateSetup(templateDir string) {
	funcMap := map[string]interface{}{
		"add":          func(a, b int) int { return a + b },
		"sub":          func(a, b int) int { return a - b },
		"mul":          func(a, b int) int { return a * b },
		"div":          func(a, b int) int { return a / b },
		"toMB":         func(a int) float64 { return float64(a) / (1024 * 1024) },
		"toKB":         func(a int) float64 { return float64(a) / (1024) },
		"toInt":        func(a int64) int { return int(a) },
		"stopEscaping": func(a string) template.HTML { return template.HTML(a) },
		"strEncode":    strEncode,
		"escape":       escape,
		"escapeSpace":  escapeSpace,
		"localtime":    func(stamp int64) string { return time.Unix(stamp, 0).Format("2006-01-02 15:04") },
		"unescapedPrintf": func(format string, a ...interface{}) htmlTemplate.HTML {
			return htmlTemplate.HTML(fmt.Sprintf(format, a))
		},
	}

	templateFiles := templateDir + "/*.txt"
	if !IsDir(templateDir) {
		log.Fatal(templateDir, "not found")
	}
	htemplates.Funcs(htmlTemplate.FuncMap(funcMap))
	_, err := htemplates.ParseGlob(templateFiles)
	if err != nil {
		log.Fatal(err)
	}

	templateFiles = templateDir + "/rss1.txt"
	ttemplates.Funcs(textTemplate.FuncMap(funcMap))
	_, err = ttemplates.ParseFiles(templateFiles)
	if err != nil {
		log.Fatal(err)
	}
}

//renderTemplate executes template and write to wr.
func renderTemplate(file string, st interface{}, wr io.Writer) {
	if err := htemplates.ExecuteTemplate(wr, file, st); err != nil {
		log.Println(err)
	}
}

//executeTemplate executes template and returns it as string.
func executeTemplate(file string, st interface{}) string {
	var doc bytes.Buffer
	renderTemplate(file, st, &doc)
	return doc.String()
}

//renderRSSTemplate executes rss template and write to wr.
func renderRSSTemplate(file string, st interface{}, wr io.Writer) {
	if err := ttemplates.ExecuteTemplate(wr, "rss1", st); err != nil {
		log.Println(err)
	}
}
