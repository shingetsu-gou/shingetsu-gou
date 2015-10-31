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
	"errors"
	"fmt"
	"html"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	errSpamM = errors.New("this is spam")
)

//postComment creates a record from args and adds it to cache.
//also adds tag if not tag!=""
func (m *mchCGI) postComment(threadKey, name, mail, body, passwd, tag string) error {
	stamp := time.Now().Unix()
	recbody := make(map[string]string)
	recbody["body"] = html.EscapeString(body)
	recbody["name"] = html.EscapeString(name)
	recbody["mail"] = html.EscapeString(mail)

	c := newCache(threadKey, m.Config, m.Global)
	rec := newRecord(c.Datfile, "", m.Config)
	rec.build(stamp, recbody, passwd)
	if rec.isSpam() {
		return errSpamM
	}
	rec.sync()
	if tag != "" {
		c.setTags([]string{tag})
		c.syncTag()
	}
	go m.UpdateQue.updateNodes(rec, nil)
	return nil
}

//errorResp render erro page with cp932 code.
func (m *mchCGI) errorResp(msg string, info map[string]string) {
	m.wr.Header().Set("Content-Type", "text/html; charset=Shift_JIS")
	info["message"] = msg
	m.Htemplate.renderTemplate("2ch_error", info, m.wr)
}

//getCP932 returns form value of key with cp932 code.
func (m *mchCGI) getCP932(key string) string {
	return fromSJIS(m.req.FormValue(key))
}

//getcommentData returns comment data with map in cp932 code.
func (m *mchCGI) getCommentData() map[string]string {
	mail := m.getCP932("mail")
	if strings.ToLower(mail) == "sage" {
		mail = ""
	}
	return map[string]string{
		"subject": m.getCP932("subject"),
		"name":    m.getCP932("FROM"),
		"mail":    mail,
		"body":    m.getCP932("MESSAGE"),
		"key":     m.getCP932("key"),
	}
}

func (m *mchCGI) checkInfo(info map[string]string) string {
	key := ""
	if info["subject"] != "" {
		key = fileEncode("thread", info["subject"])
	} else {
		var err error
		key, err = m.datakeyTable.getFilekey(info["key"])
		if err != nil {
			m.errorResp(err.Error(), info)
			return ""
		}
	}

	switch {
	case info["body"] == "":
		m.errorResp("本文がありません.", info)
		return ""
	case newCache(key, m.Config, m.Global).Exists(), m.hasAuth():
	case info["subject"] != "":
		m.errorResp("掲示版を作る権限がありません", info)
		return ""
	default:
		m.errorResp("掲示版がありません", info)
		return ""
	}

	if info["subject"] == "" && key == "" {
		m.errorResp("フォームが変です.", info)
		return ""
	}
	return key
}

//postCommentApp
func (m *mchCGI) postCommentApp() {

	if m.req.Method != "POST" {
		m.wr.Header().Set("Content-Type", "text/plain")
		m.wr.WriteHeader(404)
		fmt.Fprintf(m.wr, "404 Not Found")
		return
	}
	info := m.getCommentData()
	info["host"] = m.req.Host
	key := m.checkInfo(info)
	if key == "" {
		return
	}

	referer := m.getCP932("Referer")
	reg := regexp.MustCompile("/2ch_([^/]+)/")
	var tag string
	if ma := reg.FindStringSubmatch(referer); ma != nil && m.hasAuth() {
		tag = fileDecode("dummy_" + ma[1])
	}
	table := newResTable(newCache(key, m.Config, m.Global))
	reg = regexp.MustCompile(">>([1-9][0-9]*)")
	body := reg.ReplaceAllStringFunc(info["body"], func(str string) string {
		noStr := reg.FindStringSubmatch(str)[1]
		no, err := strconv.Atoi(noStr)
		if err != nil {
			log.Fatal(err)
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
	if passwd != "" && !m.isAdmin() {
		m.errorResp("自ノード以外で署名機能は使えません", info)
	}
	err := m.postComment(key, name, info["mail"], body, passwd, tag)
	if err == errSpamM {
		m.errorResp("スパムとみなされました", info)
	}
	m.wr.Header().Set("Content-Type", "text/html; charset=Shift_JIS")
	fmt.Fprintln(m.wr,
		toSJIS(`<html lang="ja"><head><meta http-equiv="Content-Type" content="text/html"><title>書きこみました。</title></head><body>書きこみが終わりました。<br><br></body></html>`))
}
