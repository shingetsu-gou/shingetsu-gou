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

package mch

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/cgi"
	"github.com/shingetsu-gou/shingetsu-gou/mch"
	"github.com/shingetsu-gou/shingetsu-gou/mch/keylib"
	"github.com/shingetsu-gou/shingetsu-gou/record"
	"github.com/shingetsu-gou/shingetsu-gou/tag/user"
	"github.com/shingetsu-gou/shingetsu-gou/thread"
	"github.com/shingetsu-gou/shingetsu-gou/thread/download"
	"github.com/shingetsu-gou/shingetsu-gou/updateque"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//Setup setups handlers for 2ch interface.
func Setup(s *cgi.LoggingServeMux) {
	log.Println("start 2ch interface")
	rtr := mux.NewRouter()

	cgi.RegistToRouter(rtr, "/2ch/", boardApp)
	cgi.RegistToRouter(rtr, "/2ch/dat/{datkey:[^\\.]+}.dat", threadApp)
	cgi.RegistToRouter(rtr, "/2ch/{board:[^/]+}/subject.txt", subjectApp)
	cgi.RegistToRouter(rtr, "/2ch/subject.txt", subjectApp)
	cgi.RegistToRouter(rtr, "/2ch/{board:[^/]+}/head.txt", headApp)
	cgi.RegistToRouter(rtr, "/2ch/head.txt", headApp)
	s.Handle("/2ch/", handlers.CompressHandler(rtr))

	s.RegistCompressHandler("/test/bbs.cgi", postCommentApp)
}

//boardApp just calls boardApp(), only print title.
func boardApp(w http.ResponseWriter, r *http.Request) {
	a, err := newMchCGI(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	a.boardApp()
}

//threadApp renders dat files(record data) in the thread.
func threadApp(w http.ResponseWriter, r *http.Request) {
	a, err := newMchCGI(w, r)
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
	if err != nil {
		log.Println(err)
		return
	}
	a.postCommentApp()
}

//headApp just renders motd.
func headApp(w http.ResponseWriter, r *http.Request) {
	a, err := newMchCGI(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	a.headApp()
}

//mchCGI is a class for renderring pages of 2ch interface .
type mchCGI struct {
	*cgi.CGI
}

//newMchCGI returns mchCGI obj if visitor  is allowed.
//if not allowed print 403.
func newMchCGI(w http.ResponseWriter, r *http.Request) (*mchCGI, error) {
	c, err := cgi.NewCGI(w, r)
	if err != nil {
		w.WriteHeader(403)
		fmt.Fprintf(w, "403 Forbidden")
		return nil, err
	}
	a := mchCGI{
		CGI: c,
	}
	if !c.CheckVisitor() {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return nil, errors.New("403 forbidden")
	}

	return &a, nil
}

//serveContent serves str as content with name=name(only used suffix to determine
//data type),time=t after converted cp932. ServeContent is used to make clients possible
//to use range request.
func (m *mchCGI) serveContent(name string, t time.Time, str string) {
	br := bytes.NewReader([]byte(util.ToSJIS(str)))
	http.ServeContent(m.WR, m.Req, name, t, br)
}

//boardApp just renders title stripped from url.
func (m *mchCGI) boardApp() {
	l := m.Req.FormValue("Accept-Language")
	if l == "" {
		l = "ja"
	}
	msg := cgi.SearchMessage(l, cfg.FileDir)
	m.WR.Header().Set("Content-Type", "text/html; charset=Shift_JIS")
	board := util.Escape(util.GetBoard(m.Path()))
	text := ""
	if board != "" {
		text = fmt.Sprintf("%s - %s - %s", msg["logo"], msg["description"], board)
	} else {
		text = fmt.Sprintf("%s - %s", msg["logo"], msg["description"])
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
	m.WR.Header().Set("Content-Type", "text/plain; charset=Shift_JIS")
	n, err := strconv.ParseInt(datkey, 10, 64)
	if err != nil {
		log.Println(err)
		return
	}
	key := keylib.GetFilekey(n)
	if err != nil {
		m.WR.WriteHeader(404)
		fmt.Fprintf(m.WR, "404 Not Found")
		return
	}
	data := thread.NewCache(key)

	if !data.Exists() {
		m.WR.WriteHeader(404)
		fmt.Fprintf(m.WR, "404 Not Found")
		return
	}

	if m.CheckGetCache() {
		download.GetCache(true, data)
	}

	thread := keylib.MakeDat(data, board, m.Req.Host)
	str := strings.Join(thread, "\n") + "\n"
	m.serveContent("a.txt", time.Unix(data.Stamp(), 0), str)
}

//makeSubjectCachelist returns thread.Caches in all thread.Cache and in recentlist sorted by recent stamp.
//if board is specified,  returns thread.Caches whose tagstr=board.
func (m *mchCGI) makeSubjectCachelist(board string) []*thread.Cache {
	result := thread.MakeRecentCachelist()
	if board == "" {
		return result
	}
	var result2 []*thread.Cache
	for _, c := range result {
		if user.Has(c.Datfile, board) {
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
	m.WR.Header().Set("Content-Type", "text/plain; charset=Shift_JIS")
	m.serveContent("a.txt", time.Unix(lastStamp, 0), strings.Join(subject, "\n")+"\n")
}

//makeSubject makes subject.txt(list of records title) from thread.Caches with tag=board.
func (m *mchCGI) makeSubject(board string) ([]string, int64) {
	loadFromNet := m.CheckGetCache()
	var subjects []string
	cl := m.makeSubjectCachelist(board)
	var lastStamp int64
	for _, c := range cl {
		if !loadFromNet && c.Len() == 0 {
			continue
		}
		if lastStamp < c.Stamp() {
			lastStamp = c.Stamp()
		}
		key, err := keylib.GetDatkey(c.Datfile)
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
			key, titleStr, c.Len()))
	}
	return subjects, lastStamp
}

//headApp renders motd(terms of service).
func (m *mchCGI) headApp() {
	m.WR.Header().Set("Content-Type", "text/plain; charset=Shift_JIS")
	var body string
	err := util.EachLine(cfg.Motd(), func(line string, i int) error {
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
	rec := record.New(c.Datfile, "", 0)
	rec.Build(stamp, recbody, passwd)
	if rec.IsSpam() {
		return errSpamM
	}
	rec.Sync()
	if tag != "" {
		user.Set(c.Datfile, []string{tag})
	}
	go updateque.UpdateNodes(rec, nil)
	return nil
}

//errorResp render erro page with cp932 code.
func (m *mchCGI) errorResp(msg string, info map[string]string) {
	m.WR.Header().Set("Content-Type", "text/html; charset=Shift_JIS")
	info["message"] = msg
	cgi.TmpH.RenderTemplate("2ch_error", info, m.WR)
}

//getCP932 returns form value of key with cp932 code.
func (m *mchCGI) getCP932(key string) string {
	return util.FromSJIS(m.Req.FormValue(key))
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

//checkInfo checks posted info and returs thread name.
//if ok.
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
		key = keylib.GetFilekey(n)
	}

	switch {
	case info["body"] == "":
		m.errorResp("本文がありません.", info)
		return ""
	case thread.NewCache(key).Exists(), m.HasAuth():
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

//postCommentApp checks posted data and replaces >> links to html links,
//and  saves it as record.
func (m *mchCGI) postCommentApp() {
	if m.Req.Method != http.MethodPost {
		m.WR.Header().Set("Content-Type", "text/plain")
		m.WR.WriteHeader(404)
		fmt.Fprintf(m.WR, "404 Not Found")
		return
	}
	info := m.getCommentData()
	info["host"] = m.Req.Host
	key := m.checkInfo(info)
	if key == "" {
		return
	}

	referer := m.getCP932("Referer")
	reg := regexp.MustCompile("/2ch_([^/]+)/")
	var tag string
	if ma := reg.FindStringSubmatch(referer); ma != nil && m.HasAuth() {
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
	if passwd != "" && !m.IsAdmin() {
		m.errorResp("自ノード以外で署名機能は使えません", info)
	}
	err := m.postComment(key, name, info["mail"], body, passwd, tag)
	if err == errSpamM {
		m.errorResp("スパムとみなされました", info)
	}
	m.WR.Header().Set("Content-Type", "text/html; charset=Shift_JIS")
	fmt.Fprintln(m.WR,
		util.ToSJIS(`<html lang="ja"><head><meta http-equiv="Content-Type" content="text/html"><title>書きこみました。</title></head><body>書きこみが終わりました。<br><br></body></html>`))
}
