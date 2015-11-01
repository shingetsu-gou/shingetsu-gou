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
	"bytes"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/shingetsu-gou/shingetsu-gou/mch"
	"github.com/shingetsu-gou/shingetsu-gou/thread"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//mchSetup setups handlers for 2ch interface.
func MchSetup(s *loggingServeMux) {
	log.Println("start 2ch interface")
	rtr := mux.NewRouter()

	registToRouter(rtr, "/2ch/", boardApp)
	registToRouter(rtr, "/2ch/dat/{datkey:[^\\.]+}.dat", threadApp)
	registToRouter(rtr, "/2ch/{board:[^/]+}/subject.txt", subjectApp)
	registToRouter(rtr, "/2ch/subject.txt", subjectApp)
	registToRouter(rtr, "/2ch/{board:[^/]+}/head.txt", headApp)
	registToRouter(rtr, "/2ch/head.txt", headApp)
	s.Handle("/2ch/", handlers.CompressHandler(rtr))

	s.RegistCompressHandler("/test/bbs.cgi", postCommentApp)
}

//boardApp just calls boardApp(), only print title.
func boardApp(w http.ResponseWriter, r *http.Request) {
	a, err := newMchCGI(w, r)
	defer a.close()
	if err != nil {
		log.Println(err)
		return
	}
	a.boardApp()
}

//threadApp renders dat files(record data) in the thread.
func threadApp(w http.ResponseWriter, r *http.Request) {
	a, err := newMchCGI(w, r)
	defer a.close()
	if err != nil {
		log.Println(err)
		return
	}
	m := mux.Vars(r)
	board := m["board"]
	if board == "" {
		board = "2ch"
	}
	a.threadApp(board, m["datkey"])
}

//subjectApp renders time-subject lines of the thread.
func subjectApp(w http.ResponseWriter, r *http.Request) {
	a, err := newMchCGI(w, r)
	defer a.close()
	if err != nil {
		log.Println(err)
		return
	}
	m := mux.Vars(r)
	a.subjectApp(m["board"])
}

//postCommentApp posts one record to the thread.
func postCommentApp(w http.ResponseWriter, r *http.Request) {
	a, err := newMchCGI(w, r)
	defer a.close()
	if err != nil {
		log.Println(err)
		return
	}
	a.postCommentApp()
}

//headApp just renders motd.
func headApp(w http.ResponseWriter, r *http.Request) {
	a, err := newMchCGI(w, r)
	defer a.close()
	if err != nil {
		log.Println(err)
		return
	}
	a.headApp()
}

var MchCfg *MchConfig

type MchConfig struct {
	Motd         string
	RecentList   *thread.RecentList
	DatakeyTable *mch.DatakeyTable
	UpdateQue    *thread.UpdateQue
}

//mchCGI is a class for renderring pages of 2ch interface .
type mchCGI struct {
	*cgi
	*MchConfig
}

//newMchCGI returns mchCGI obj if visitor  is allowed.
//if not allowed print 403.
func newMchCGI(w http.ResponseWriter, r *http.Request) (mchCGI, error) {
	c := mchCGI{
		cgi:       NewCGI(w, r),
		MchConfig: MchCfg,
	}
	defer c.close()
	if c.cgi == nil || !c.checkVisitor() {
		w.WriteHeader(403)
		fmt.Fprintf(w, "403 Forbidden")
		return c, errors.New("403 forbidden")
	}

	return c, nil
}

//serveContent serves str as content with name=name(only used suffix to determine
//data type),time=t after converted cp932. ServeContent is used to make clients possible
//to use range request.
func (m *mchCGI) serveContent(name string, t time.Time, str string) {
	br := bytes.NewReader([]byte(util.ToSJIS(str)))
	http.ServeContent(m.wr, m.req, name, t, br)
}

//boardApp just renders title stripped from url.
func (m *mchCGI) boardApp() {
	l := m.req.FormValue("Accept-Language")
	if l == "" {
		l = "ja"
	}
	message := searchMessage(l, m.FileDir)
	m.wr.Header().Set("Content-Type", "text/html; charset=Shift_JIS")
	board := util.Escape(util.GetBoard(m.path()))
	text := ""
	if board != "" {
		text = fmt.Sprintf("%s - %s - %s", message["logo"], message["description"], board)
	} else {
		text = fmt.Sprintf("%s - %s", message["logo"], message["description"])
	}

	htmlStr := fmt.Sprintf(
		`<!DOCTYPE html>
        <html><head>
        <meta http-equiv="content-type" content="text/html; charset=Shift_JIS">
        <title>%s</title>
        <meta name="description" content="%s">
        </head><body>
        <h1>%s</h1>
        </body></html>`,
		text, text, text)
	m.serveContent("a.html", time.Time{}, htmlStr)
}

//threadApp load thread.Cache specified in the url and returns dat file
//listing records. if thread.Cache len=0 or for each refering the thread.Cache 4 times
//reloads thread.Cache fron network.
func (m *mchCGI) threadApp(board, datkey string) {
	m.wr.Header().Set("Content-Type", "text/plain; charset=Shift_JIS")
	n, err := strconv.ParseInt(datkey, 10, 64)
	if err != nil {
		log.Println(err)
		return
	}
	key := m.DatakeyTable.GetFilekey(n)
	if err != nil {
		m.wr.WriteHeader(404)
		fmt.Fprintf(m.wr, "404 Not Found")
	}
	data := thread.NewCache(key)
	i := data.ReadInfo()

	if m.checkGetCache() {
		if data.Exists() || i.Len == 0 {
			data.GetCache()
		} else {
			go data.GetCache()
		}
	}

	if !data.Exists() {
		m.wr.WriteHeader(404)
		fmt.Fprintf(m.wr, "404 Not Found")
	}
	thread := m.DatakeyTable.MakeDat(data, board, m.req.Host)
	str := strings.Join(thread, "\n")
	m.serveContent("a.txt", time.Unix(i.Stamp, 0), str)
}

//makeSubjectCachelist returns thread.Caches in all thread.Cache and in recentlist sorted by recent stamp.
//if board is specified,  returns thread.Caches whose tagstr=board.
func (m *mchCGI) makeSubjectCachelist(board string) []*thread.Cache {
	cl := thread.NewCacheList()
	seen := make([]string, cl.Len())
	for i, c := range cl.Caches {
		seen[i] = c.Datfile
	}
	for _, rec := range m.RecentList.GetRecords() {
		if !util.HasString(seen, rec.Datfile) {
			seen = append(seen, rec.Datfile)
			c := thread.NewCache(rec.Datfile)
			cl.Append(c)
		}
	}
	var result []*thread.Cache
	for _, c := range cl.Caches {
		result = append(result, c)
	}
	sort.Sort(sort.Reverse(thread.SortByRecentStamp{result}))
	if board == "" {
		return result
	}
	var result2 []*thread.Cache
	for _, c := range result {
		if c.HasTag(board) {
			result2 = append(result2, c)
		}
	}
	return result2
}

//subjectApp makes list of records title from thread.Caches whose tag is same as one stripped from url.
func (m *mchCGI) subjectApp(board string) {
	var boardEncoded, boardName string
	if board != "" {
		boardEncoded = util.StrDecode(board)
	}
	if boardEncoded != "" {
		boardName = util.FileDecode("dummy_" + boardEncoded)
	}
	subject, lastStamp := m.makeSubject(boardName)
	m.wr.Header().Set("Content-Type", "text/plain; charset=Shift_JIS")
	m.serveContent("a.txt", time.Unix(lastStamp, 0), strings.Join(subject, "\n"))
}

//makeSubject makes subject.txt(list of records title) from thread.Caches with tag=board.
func (m *mchCGI) makeSubject(board string) ([]string, int64) {
	loadFromNet := m.checkGetCache()
	var subjects []string
	cl := m.makeSubjectCachelist(board)
	var lastStamp int64
	for _, c := range cl {
		i := c.ReadInfo()
		if !loadFromNet && i.Len == 0 {
			continue
		}
		if lastStamp < i.Stamp {
			lastStamp = i.Stamp
		}
		key, err := m.DatakeyTable.GetDatkey(c.Datfile)
		if err != nil {
			log.Println(err)
			continue
		}
		titleStr := util.FileDecode(c.Datfile)
		if titleStr == "" {
			continue
		}
		titleStr = strings.Trim(titleStr, "\r\n")
		subjects = append(subjects, fmt.Sprintf("%d.dat<>%s (%d)",
			key, titleStr, i.Len))
	}
	return subjects, lastStamp
}

//headApp renders motd(terms of service).
func (m *mchCGI) headApp() {
	m.wr.Header().Set("Content-Type", "text/plain; charset=Shift_JIS")
	var body string
	err := util.EachLine(m.Motd, func(line string, i int) error {
		line = strings.TrimSpace(line)
		body += line + "<br>\n"
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	m.serveContent("a.txt", time.Time{}, body)
}

var (
	errSpamM = errors.New("this is spam")
)

//postComment creates a record from args and adds it to thread.Cache.
//also adds tag if not tag!=""
func (m *mchCGI) postComment(threadKey, name, mail, body, passwd, tag string) error {
	stamp := time.Now().Unix()
	recbody := make(map[string]string)
	recbody["body"] = html.EscapeString(body)
	recbody["name"] = html.EscapeString(name)
	recbody["mail"] = html.EscapeString(mail)

	c := thread.NewCache(threadKey)
	rec := thread.NewRecord(c.Datfile, "")
	rec.Build(stamp, recbody, passwd)
	if rec.IsSpam() {
		return errSpamM
	}
	rec.Sync()
	if tag != "" {
		c.SetTags([]string{tag})
		c.SyncTag()
	}
	go m.UpdateQue.UpdateNodes(rec, nil)
	return nil
}

//errorResp render erro page with cp932 code.
func (m *mchCGI) errorResp(msg string, info map[string]string) {
	m.wr.Header().Set("Content-Type", "text/html; charset=Shift_JIS")
	info["message"] = msg
	m.Htemplate.RenderTemplate("2ch_error", info, m.wr)
}

//getCP932 returns form value of key with cp932 code.
func (m *mchCGI) getCP932(key string) string {
	return util.FromSJIS(m.req.FormValue(key))
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
		key = util.FileEncode("thread", info["subject"])
	} else {
		n, err := strconv.ParseInt(info["key"], 10, 64)
		if err != nil {
			m.errorResp(err.Error(), info)
			return ""
		}
		key = m.DatakeyTable.GetFilekey(n)
	}

	switch {
	case info["body"] == "":
		m.errorResp("本文がありません.", info)
		return ""
	case thread.NewCache(key).Exists(), m.hasAuth():
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
		tag = util.FileDecode("dummy_" + ma[1])
	}
	table := mch.NewResTable(thread.NewCache(key))
	reg = regexp.MustCompile(">>([1-9][0-9]*)")
	body := reg.ReplaceAllStringFunc(info["body"], func(str string) string {
		noStr := reg.FindStringSubmatch(str)[1]
		no, err := strconv.Atoi(noStr)
		if err != nil {
			log.Fatal(err)
		}
		return ">>" + table.Num2id[no]
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
		util.ToSJIS(`<html lang="ja"><head><meta http-equiv="Content-Type" content="text/html"><title>書きこみました。</title></head><body>書きこみが終わりました。<br><br></body></html>`))
}
