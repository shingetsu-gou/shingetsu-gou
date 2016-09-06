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
	"bufio"
	"bytes"
	"encoding/json"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	"github.com/nfnt/resize"
)

//provider represents oembed provider.
type provider struct {
	ProviderURL    string `json:"provider_url"`
	regProviderURL *regexp.Regexp
	Endpoints      []*struct {
		Schemes    []string
		regSchemes []*regexp.Regexp
		URL        string
	}
}

type emoji struct {
	Unicode      string
	Shortname    string
	AliasesASCII []string `json:"aliases_ascii"`
}

//prov is oembed providers from oembed_providers.go
var prov []*provider
var emojis map[string]*emoji

//init loads oembed providers in json.
func init() {
	if err := json.Unmarshal([]byte(oembedProviders), &prov); err != nil {
		log.Fatal(err)
	}
	for _, p := range prov {
		p.regProviderURL = regexp.MustCompile(p.ProviderURL + ".*")
		for _, e := range p.Endpoints {
			e.regSchemes = make([]*regexp.Regexp, len(e.Schemes))
			for i, s := range e.Schemes {
				e.regSchemes[i] = regexp.MustCompile(s + ".*")
			}
		}
	}

	js, err := Asset("www/emoji.json")
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(js, &emojis); err != nil {
		log.Fatal(err)
	}
}

//EachIOLine iterates each line to  a ReadCloser ,calls func and close f.
func EachIOLine(f io.ReadCloser, handler func(line string, num int) error) error {
	defer Fclose(f)
	r := bufio.NewReader(f)
	var err error
	for i := 0; err == nil; i++ {
		var line string
		line, err = r.ReadString('\n')
		if err != nil && line == "" {
			break
		}
		line = strings.Trim(line, "\n\r")
		errr := handler(line, i)
		if errr != nil {
			return errr
		}
	}
	if err == io.EOF {
		return nil
	}
	return err
}

//EachLine iterates each lines and calls a func.
func EachLine(path string, handler func(line string, num int) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	return EachIOLine(f, handler)
}

//HasString returns true if ary has val.
func HasString(s []string, val string) bool {
	return FindString(s, val) != -1
}

//FindString search the val in ary and returns index. it returns -1 if not found.
func FindString(s []string, val string) int {
	for i, v := range s {
		if v == val {
			return i
		}
	}
	return -1
}

//EachFiles iterates each dirs in dir and calls handler,not recirsively.
func EachFiles(dir string, handler func(dir os.FileInfo) error) error {
	dirs, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, i := range dirs {
		if err := handler(i); err != nil {
			return err
		}
	}
	return nil
}

//IsFile returns true is path is an existing file.
func IsFile(path string) bool {
	fs, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !fs.IsDir()
}

//IsDir returns true is path is an existing dir.
func IsDir(path string) bool {
	fs, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fs.IsDir()
}

//Fclose closes io.Close, if err exists ,println err.
func Fclose(f io.Closer) {
	if err := f.Close(); err != nil {
		log.Println(err)
	}
}

//MakeThumbnail makes thumbnail to suffix image format with thumbnailSize.
func MakeThumbnail(encoded []byte, suffix, thumbnailSize string) []byte {
	size := strings.Split(thumbnailSize, "x")
	if len(size) != 2 {
		return nil
	}
	x, err1 := strconv.Atoi(size[0])
	y, err2 := strconv.Atoi(size[1])
	if err1 != nil || err2 != nil {
		log.Println(thumbnailSize, "is illegal format")
		return nil
	}

	file := bytes.NewReader(encoded)
	img, _, err := image.Decode(file)
	if err != nil {
		log.Println(err)
		return nil
	}
	m := resize.Resize(uint(x), uint(y), img, resize.Lanczos3)
	var out bytes.Buffer
	switch suffix {
	case "jpg", "jpeg":
		err = jpeg.Encode(&out, m, nil)
	case "png":
		err = png.Encode(&out, m)
	case "gif":
		err = gif.Encode(&out, m, nil)
	default:
		log.Println("illegal format", suffix)
	}
	if err != nil {
		log.Println(err)
	}
	return out.Bytes()
}

// ToSJIS converts an string (a valid UTF-8 string) to a ShiftJIS string
func ToSJIS(b string) string {
	return convertSJIS(b, true)
}

// convertSJIS converts an string (a valid UTF-8 string) to/from a ShiftJIS string
func convertSJIS(b string, toSJIS bool) string {
	var t transform.Transformer
	t = japanese.ShiftJIS.NewDecoder()
	if toSJIS {
		tt := japanese.ShiftJIS.NewEncoder()
		t = encoding.ReplaceUnsupported(tt)
	}
	r, err := ioutil.ReadAll(transform.NewReader(bytes.NewReader([]byte(b)), t))
	if err != nil {
		log.Println(err)
	}
	return string(r)
}

//FromSJIS converts an array of bytes (a valid ShiftJIS string) to a UTF-8 string
func FromSJIS(b string) string {
	return convertSJIS(b, false)
}

//miscURL returns url for embeding for nicovideo, images.
func miscURL(url string) string {
	reg1 := regexp.MustCompile("http://www.nicovideo.jp/watch/([a-z0-9]+)")
	id := reg1.FindStringSubmatch(url)
	if len(id) > 1 {
		return `	<script type="text/javascript" src="http://ext.nicovideo.jp/thumb_watch/` + id[1] + `"></script>`
	}

	reg2 := regexp.MustCompile("https?://github.com/([^/]+)/([^/]+)")
	id = reg2.FindStringSubmatch(url)
	if len(id) > 2 {
		return `<div class="github-card" data-user="` + id[1] + `" data-repo="` + id[2] + `"></div>
<script src="http://cdn.jsdelivr.net/github-cards/latest/widget.js"></script>`
	}

	images := []string{"jpeg", "jpg", "gif", "png"}
	for _, img := range images {
		if strings.HasSuffix(url, img) {
			return `<img src="/x.gif" data-lazyimg data-src="` + url + `" height="210" alt="" />`
		}
	}
	return ""
}

//getJSON get json and converts map[string]interface{} from url by using GET.
func getJSON(url string) (map[string]interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	defer Fclose(resp.Body)
	js, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	var m map[string]interface{}
	if errr := json.Unmarshal(js, &m); errr != nil {
		log.Print(errr)
		return nil, errr
	}
	return m, err
}

//oEmbedURL returns url for embed by using oEmbed.
func oEmbedURL(url string) string {
	for _, p := range prov {
		match := false
		if p.regProviderURL != nil {
			match = p.regProviderURL.MatchString(url)
		}
		for _, e := range p.Endpoints {
			for _, s := range e.regSchemes {
				if s == nil {
					continue
				}
				match = match || s.MatchString(url)
			}
			if !match {
				continue
			}
			log.Println("geting embed url from", e.URL)
			m, err := getJSON(e.URL + "?url=" + url + "&format=json")
			if err != nil {
				log.Print(err)
				continue
			}
			if html, ok := m["html"].(string); ok {
				return html
			}
		}
	}
	return ""
}

//EmbedURL gets the url for embeding by using oEmbed API.
func EmbedURL(url string) string {
	if e := miscURL(url); e != "" {
		return e
	}
	return oEmbedURL(url)
}

//HasExt returns true if fname has prefix and not secret.
func HasExt(fname, suffix string) bool {
	return strings.HasSuffix(fname, "."+suffix) && (!strings.HasPrefix(fname, ".") || strings.HasPrefix(fname, "_"))
}

//Emoji converts :hoe: to a png pic link.
func Emoji(str string) string {
	for _, v := range emojis {
		match := false
		if str == v.Shortname {
			match = true
		}
		for _, alias := range v.AliasesASCII {
			if str == alias {
				match = true
			}
		}
		if match {
			return `<img height="25" src="http://cdn.jsdelivr.net/emojione/assets/png/` + v.Unicode + `.png">`
		}
	}
	return str
}
