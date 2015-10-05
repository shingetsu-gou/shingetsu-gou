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
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"log"
	"mime"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

func md5digest(dat string) string {
	sum := md5.Sum([]byte(dat))
	return hex.EncodeToString(sum[:])
}

func strEncode(query string) string {
	str := url.QueryEscape(query)
	return strings.Replace(str, "~", "%7E", -1)
}

func escapeSpace(msg string) string {
	msg = strings.Replace(msg, "  ", "&nbsp;&nbsp;", -1)
	msg = strings.Replace(msg, "<br> ", "<br>&nbsp;", -1)
	reg := regexp.MustCompile("^ ")
	msg = string(reg.ReplaceAllString(msg, "&nbsp;"))
	reg = regexp.MustCompile(" $")
	msg = string(reg.ReplaceAllString(msg, "&nbsp;"))
	msg = strings.Replace(msg, "<br>", "<br />\n", -1)
	return msg
}

func cgiEscape(msg string, quote bool) string {
	msg = strings.Replace(msg, "&", "&amp;", -1)
	msg = strings.Replace(msg, "<", "&lt", -1)
	msg = strings.Replace(msg, ">", "&gt;", -1)
	if quote {
		msg = strings.Replace(msg, "\"", "&guote;", -1)
	}
	return msg
}

func escape(msg string) string {
	if msg == "" {
		return ""
	}
	msg = strings.Replace(msg, "&", "&amp;", -1)
	reg := regexp.MustCompile("&amp;(#\\d+|#[Xx][0-9A-Fa-f]+|[A-Za-z0-9]+);")
	msg = string(reg.ReplaceAllString(msg, "&\\1;"))
	msg = strings.Replace(msg, "<", "&lt", -1)
	msg = strings.Replace(msg, ">", "&gt;", -1)
	msg = strings.Replace(msg, "\r", "", -1)
	msg = strings.Replace(msg, "\n", "<br>", -1)
	return msg
}

func strDecode(query string) string {
	str, err := url.QueryUnescape(query)
	if err != nil {
		return ""
	}
	return str
}

//from spam.py

func spamCheck(recstr string) bool {
	if cached_rule == nil {
		cached_rule = newRegexpList(spam_list)
	} else {
		cached_rule.update()
	}
	return cached_rule.check(recstr)
}

//fsdiff checks a difference between file and string.
//Return same data or not.
func fsdiff(f, s string) bool {
	cont, err := ioutil.ReadFile(f)
	if err != nil {
		log.Println(err)
		return false
	}
	if string(cont) == s {
		return true
	}
	return false
}

//from attachutil.py

//Type of path is same as mimetype or not.
func isValidImage(mimetype, path string) bool {
	ext := filepath.Ext(path)
	if ext == "" {
		return false
	}
	realType := mime.TypeByExtension(ext)
	if realType == mimetype {
		return true
	}
	if realType == "image/jpeg" && mimetype == "image/pjpeg" {
		return true
	}
	return false
}

//from mch/util.py

func saveTag(ca *cache, userTag string) {
	ca.tags.update([]string{userTag})
	ca.tags.sync()
	utl := newUserTagList()
	utl.sync()
}

func getBoard(url string) string {
	reg := regexp.MustCompile("/2ch_([^/]+)/")
	m := reg.FindStringSubmatch(url)
	if m == nil {
		return ""
	}
	board, _ := fileDecode("dummy_" + m[1])
	return board
}


//from title.py


//Encode for filename.
//    >>> file_encode('foo', 'a')
//    'foo_61'
//    >>> file_encode('foo', '~')
//    'foo_7E'
func fileEncode(t, query string) string {
	return t + "_" + strings.ToUpper(hex.EncodeToString([]byte(query)))
}

//Decode file type.
//    >>> file_decode_type('thread_41')
//    'thread'
func fileDecode(query string) (string, string) {
	strs := strings.Split(query, "_")
	if len(strs) < 2 {
		return "", ""
	}
	b, err := hex.DecodeString(strs[1])
	sb := string(b)
	if err != nil {
		log.Println("illegal file name", query)
		sb = ""
	}
	return strs[0], sb
}

//not implement except 'asis'
func fileHash(query string) string {
	return query
}
