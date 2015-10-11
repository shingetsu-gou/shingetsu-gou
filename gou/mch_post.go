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
	"html"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/axgle/mahonia"
)

func (m *mchCGI) postComment(threadKey, name, mail, body, passwd, tag string) error {
	stamp := time.Now().Unix()
	recbody := make(map[string]string)
	recbody["body"] = html.EscapeString(body)
	recbody["name"] = html.EscapeString(name)
	recbody["mail"] = html.EscapeString(mail)

	c := newCache(threadKey)
	rec := newRecord(c.Datfile, "")
	rec.build(stamp, recbody, passwd)
	if spamCheck(rec.recstr()) {
		return errSpam
	}
	c.addData(rec)
	c.syncStatus()
	if tag != "" {
		saveTag(c, tag)
	}
	queue.append(rec, nil)
	queue.run()
	return nil
}

func (m *mchCGI) errorResp(msg string, info map[string]string) string {
	info["message"] = msg
	str := executeTemplate("2ch_error", info)
	m.wr.Header().Set("Content-Type", "text/html; charset=Shift_JIS")
	return mahonia.NewEncoder("cp932").ConvertString(str)
}

func (m *mchCGI) getCP932(key string) string {
	return mahonia.NewDecoder("cp932").ConvertString(m.req.FormValue(key))
}

func (m *mchCGI) getCommentData() map[string]string {
	mail := m.getCP932("mail")
	if strings.ToLower(mail) == "sage" {
		mail = ""
	}
	return map[string]string{
		"subject": m.getCP932("subject"),
		"from":    m.getCP932("FROM"),
		"mail":    mail,
		"message": m.getCP932("MESSAGE"),
		"key":     m.getCP932("key"),
	}
}

func (m *mchCGI) postCommentApp() string {
	if m.req.Method != "POST" {
		m.wr.Header().Set("Content-Type", "text/plain")
		m.wr.WriteHeader(404)
		return "404 Not Found"
	}
	info := m.getCommentData()
	info["host"] = m.req.URL.Host

	if info["body"] == "" {
		return m.errorResp("本文がありません.", info)
	}

	key := ""
	if info["subject"] != "" {
		key = fileEncode("thread", info["subject"])
	} else {
		var err error
		key, err = dataKeyTable.getFilekey(info["key"])
		if err != nil {
			return ""
		}
	}
	hasAuth := m.isAdmin || m.isFriend
	referer := m.getCP932("Referer")
	reg := regexp.MustCompile("/2ch_([^/]+)/")
	var tag string
	if m := reg.FindStringSubmatch(referer); m != nil && hasAuth {
		tag = fileDecode("dummy_" + m[1])
	}

	switch {
	case newCache(key).Exists():
	case hasAuth:
	case info["subject"] != "":
		return m.errorResp("掲示版を作る権限がありません", info)
	default:
		return m.errorResp("掲示版がありません", info)
	}

	if info["subject"] == "" && key == "" {
		return m.errorResp("フォームが変です.", info)
	}

	table := newResTable(newCache(key))
	reg = regexp.MustCompile(">>([1-9][0-9]*)")
	body := reg.ReplaceAllStringFunc(info["body"], func(noStr string) string {
		no, err := strconv.Atoi(noStr)
		if err != nil {
			log.Println(err)
			return ""
		}
		return ">>" + table.num2id[no]
	})

	name := info["name"]
	var passwd string
	if strings.ContainsRune(name, '#') {
		ary := strings.Split(name, "#")
		name = ary[0]
		passwd = ary[1]
	}
	if passwd != "" && !m.isAdmin {
		return m.errorResp("自ノード以外で署名機能は使えません", info)
	}
	err := m.postComment(key, name, info["mail"], body, passwd, tag)
	if err == errSpam {
		return m.errorResp("スパムとみなされました", info)
	}
	m.wr.Header().Set("Content-Type", "text/html; charset=Shift_JIS")
	successMsg := `<html lang="ja"><head><meta http-equiv="Content-Type" content="text/html"><title>書きこみました。</title></head><body>書きこみが終わりました。<br><br></body></html>`

	return mahonia.NewDecoder("cp932").ConvertString(successMsg)
}
