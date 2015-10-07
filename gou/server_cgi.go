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
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
)

func serverSetup(s *http.ServeMux) {
	s.Handle("/server.cgi/ping", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newServerCGI(w, r); a != nil {
			a.doPing()
		}
	})))
	s.Handle("/server.cgi/node", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newServerCGI(w, r); a != nil {
			a.doNode()
		}
	})))
	s.Handle("/server.cgi/join", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newServerCGI(w, r); a != nil {
			a.doJoin()
		}
	})))
	s.Handle("/server.cgi/bye", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newServerCGI(w, r); a != nil {
			a.doBye()
		}
	})))
	s.Handle("/server.cgi/have", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newServerCGI(w, r); a != nil {
			a.doHave()
		}
	})))
	s.Handle("/server.cgi/get", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newServerCGI(w, r); a != nil {
			a.doGetHead()
		}
	})))
	s.Handle("/server.cgi/head", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newServerCGI(w, r); a != nil {
			a.doGetHead()
		}
	})))
	s.Handle("/server.cgi/update", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newServerCGI(w, r); a != nil {
			a.doUpdate()
		}
	})))
	s.Handle("/server.cgi/recent", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newServerCGI(w, r); a != nil {
			a.doRecent()
		}
	})))
	s.Handle("/server.cgi/", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := newServerCGI(w, r); a != nil {
			a.doMotd()
		}
	})))
}

type serverCGI struct {
	*cgi
}

func newServerCGI(w http.ResponseWriter, r *http.Request) *serverCGI {
	c := newCGI(w, r)
	if c == nil {
		return nil
	}
	w.Header().Set("Content-Type", "text/plain")

	if r.Method != "GET" && r.Method != "HEAD" {
		w.Header().Set("Content-Type", "text/plain")
	}

	return &serverCGI{
		c,
	}
}

func (s *serverCGI) doPing() {
	fmt.Fprint(s.wr, "PONG\n"+s.req.RemoteAddr+"\n")
}

func (s *serverCGI) doNode() {
	if nodeList.Len() > 0 {
		fmt.Fprintln(s.wr, nodeList.nodes[0].nodestr)
	} else {
		fmt.Fprintln(s.wr, nodeList.random().nodestr)
	}
}

func (s *serverCGI) getRemoteHostname(host string) string {
	remoteAddr := strings.Split(s.req.RemoteAddr, ":")[0]
	if host == "" {
		return remoteAddr
	}
	ipaddr, err := net.LookupIP(host)
	if err != nil {
		return ""
	}
	for _, ipa := range ipaddr {
		if ipa.String() == remoteAddr {
			return host
		}
	}
	return ""
}

func (s *serverCGI) makeNode(method string) *node {
	reg := regexp.MustCompile("^/" + method + "/([^:]*):(\\d+)(.*)")
	m := reg.FindStringSubmatch(s.path)
	if m == nil {
		log.Println("illegal url")
		return nil
	}
	path := m[3]
	host := s.getRemoteHostname(m[1])
	if host == "" {
		return nil
	}
	port, err := strconv.Atoi(m[2])
	if err != nil {
		log.Println(err)
		return nil
	}
	return makeNode(host, path, port)
}

func (s *serverCGI) doJoin() {
	n := s.makeNode("join")
	if n == nil {
		return
	}
	if !nodeAllow.check(n.nodestr) && nodeDeny.check(n.nodestr) {
		return
	}
	if _, ok := n.ping(); !ok {
		return
	}
	if nodeList.Len() < defaultNodes {
		nodeList.append(n)
		nodeList.sync()
		searchList.append(n)
		searchList.sync()
		fmt.Fprintln(s.wr, "WELCOME")
		return
	}
	searchList.append(n)
	searchList.sync()
	suggest := nodeList.nodes[0]
	nodeList.removeNode(suggest)
	nodeList.sync()
	suggest.bye()
	fmt.Fprintf(s.wr, "WELCOME\n%s\n", suggest)

}

func (s *serverCGI) doBye() {
	n := s.makeNode("bye")
	if n == nil {
		return
	}

	if nodeList.removeNode(n) {
		nodeList.sync()
	}
	fmt.Fprintln(s.wr, "BYEBYE")
}

func (s *serverCGI) doHave() {
	reg := regexp.MustCompile("^/have/([0-9A-Za-z_]+)$")
	m := reg.FindStringSubmatch(s.path)
	if m == nil {
		log.Println("illegal url")
		return
	}
	ca := newCache(m[1])
	if ca.Len() > 0 {
		fmt.Fprintln(s.wr, "YES")
	} else {
		fmt.Fprintln(s.wr, "NO")
	}
}

func (s *serverCGI) doUpdate() {
	reg := regexp.MustCompile("^/update/(\\w+)/(\\d+)/(\\w+)/([^:]*):(\\d+)(.*)")
	m := reg.FindStringSubmatch(s.path)
	if m == nil || len(m) < 6 {
		log.Println("illegal url")
		return
	}
	datfile, stamp, id, host, path := m[0], m[1], m[2], m[3], m[5]
	port, err := strconv.Atoi(m[4])
	if err != nil {
		log.Println(err)
		return
	}
	host = s.getRemoteHostname(host)
	if host == "" {
		return
	}

	n := makeNode(host, path, port)
	if !nodeAllow.check(n.nodestr) && nodeDeny.check(n.nodestr) {
		return
	}
	searchList.append(n)
	searchList.sync()
	lookupTable.add(datfile, n)
	lookupTable.sync(false)
	now := time.Now().Unix()
	nstamp, err := strconv.ParseInt(stamp, 10, 64)
	if err != nil {
		log.Println(err)
		return
	}

	if nstamp < now-int64(defaultUpdateRange) || nstamp > now+int64(defaultUpdateRange) {
		return
	}
	rec := newRecord(datfile, stamp+"_"+id)
	if !updateList.hasRecord(rec) {
		queue.append(rec, n)
		go queue.run()
	}

}

func (s *serverCGI) doRecent() {
	reg := regexp.MustCompile("^/recent/?([-0-9A-Za-z/]*)$")
	m := reg.FindStringSubmatch(s.path)
	if m == nil {
		log.Println("illegal url")
		return
	}
	stamp := m[1]
	last := time.Now().Unix() + int64(recentRange)
	begin, end, _ := s.parseStamp(stamp, last)
	for _, i := range recentList.records {
		if begin <= i.stamp && i.stamp <= end {
			ca := newCache(i.datfile)
			var tagstr string
			if ca.tags != nil {
				tagstr = "tag:" + ca.tags.string()
			}
			line := fmt.Sprintf("%d<>%s<>%s%s\n", i.stamp, i.id, i.datfile, tagstr)
			fmt.Fprintf(s.wr, line)
		}
	}
}

func (s *serverCGI) doMotd() {
	f, err := ioutil.ReadFile(motd)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintf(s.wr, string(f))
}

func (s *serverCGI) parseStamp(stamp string, last int64) (int64, int64, string) {
	buf := strings.Split(stamp, "/")
	var id string
	if len(buf) > 1 {
		id = buf[1]
		stamp = buf[0]
	}
	nstamp, err := strconv.ParseInt(stamp, 10, 64)
	if err != nil {
		return 0, 0, ""
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, 0, ""
	}
	buf = strings.Split(stamp, "-")
	switch {
	case stamp == "", stamp == "-":
		return 0, last, id
	case strings.HasSuffix(stamp, "-"):
		return nstamp, last, id
	case len(buf) == 1:
		return nstamp, nstamp, id
	case buf[0] == "":
		return 0, nid, id
	default:
		return nstamp, nid, id
	}
}

func (s *serverCGI) doGetHead() {
	reg := regexp.MustCompile("/(get|head)/([0-9A-Za-z_]+)/([-0-9A-Za-z/]*)$")
	m := reg.FindStringSubmatch(s.path)
	if m == nil {
		log.Println("illegal url")
		return
	}
	method, datfile, stamp := m[1], m[2], m[3]
	ca := newCache(datfile)
	begin, end, id := s.parseStamp(stamp, ca.stamp)
	for _, r := range ca.recs {
		if begin <= r.stamp && r.stamp <= end && (id == "" || strings.HasSuffix(r.idstr, id)) {
			if method == "get" {
				err := r.load()
				if err != nil {
					log.Println(err)
				}
				fmt.Fprintf(s.wr, r.recstr)
				r.free()
			} else {
				fmt.Fprintln(s.wr, strings.Replace(r.idstr, "_", "<>", -1))
			}
		}
	}
}
