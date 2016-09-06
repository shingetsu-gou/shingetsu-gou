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
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"mime"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

//MD5digest returns hex string of md5sum
func MD5digest(dat string) string {
	sum := md5.Sum([]byte(dat))
	return hex.EncodeToString(sum[:])
}

//StrEncode returns enscaped string for url , including "~"
func StrEncode(query string) string {
	str := url.QueryEscape(query)
	str = strings.Replace(str, "~", "%7E", -1)
	str = strings.Replace(str, "+", "%20", -1)
	return str
}

//EscapeSpace converts spaces into html space.
func EscapeSpace(msg string) string {
	msg = strings.Replace(msg, "  ", "&nbsp;&nbsp;", -1)
	msg = strings.Replace(msg, "<br> ", "<br>&nbsp;", -1)
	if len(msg) > 0 && msg[0] == ' ' {
		msg = "&nbsp;" + msg[1:]
	}
	if len(msg) > 0 && msg[len(msg)-1] == ' ' {
		msg = msg[:len(msg)-1] + "&nbsp;"
	}
	msg = strings.Replace(msg, "<br>", "<br />\n", -1)
	return msg
}

//Escape is like a html.escapestring, except &#xxxx and \n
func Escape(msg string) string {
	msg = strings.Replace(msg, "&", "&amp;", -1)
	reg := regexp.MustCompile(`&amp;(#\d+|#[Xx][0-9A-Fa-f]+|[A-Za-z0-9]+);`)
	msg = reg.ReplaceAllString(msg, "&$1;")
	msg = strings.Replace(msg, "<", "&lt;", -1)
	msg = strings.Replace(msg, ">", "&gt;", -1)
	msg = strings.Replace(msg, "\r", "", -1)
	msg = strings.Replace(msg, "\n", "<br>", -1)
	return msg
}

//StrDecode decode from url query
func StrDecode(query string) string {
	str, err := url.QueryUnescape(query)
	if err != nil {
		return ""
	}
	return str
}

//from attachutil.py

//IsValidImage checks type of path is same as mimetype or not.
func IsValidImage(mimetype, path string) bool {
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

//GetBoard returns decoded board name.
func GetBoard(url string) string {
	reg := regexp.MustCompile(`/2ch_([^/]+)/`)
	m := reg.FindStringSubmatch(url)
	if m == nil {
		return ""
	}
	return FileDecode("dummy_" + m[1])
}

//Datestr2ch converts unixtime str ecpochStr to the certain format string.
//e.g. 2006/01/02(日) 15:04:05.99
func Datestr2ch(epoch int64) string {
	t := time.Unix(epoch, 0)
	d := t.Format("2006/01/02(%s) 15:04:05.99")
	wdays := []string{"日", "月", "火", "水", "木", "金", "土"}
	return fmt.Sprintf(d, wdays[t.Weekday()])
}

//from title.py

//FileEncode encodes filename.
//    >>> file_encode('foo', 'a')
//    'foo_61'
//    >>> file_encode('foo', '~')
//    'foo_7E'
func FileEncode(t, query string) string {
	return t + "_" + strings.ToUpper(hex.EncodeToString([]byte(query)))
}

//FileDecode decodes filename.
//    >>> file_decode('foo_7E')
//    '~'
func FileDecode(query string) string {
	strs := strings.Split(query, "_")
	if len(strs) < 2 {
		return ""
	}
	b, err := hex.DecodeString(strs[1])
	if err != nil {
		//		log.Println("illegal file name", query, err)
		return ""
	}
	return string(b)
}
