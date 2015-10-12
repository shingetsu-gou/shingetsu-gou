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
)

//serverSetup setups handlers for server.cgi
func serverSetup(s *http.ServeMux) {
	registCompressHandler(s, "/server.cgi/ping", doPing)
	registCompressHandler(s, "/server.cgi/node", doNode)
	registCompressHandler(s, "/server.cgi/join", doJoin)
	registCompressHandler(s, "/server.cgi/bye", doBye)
	registCompressHandler(s, "/server.cgi/have", doHave)
	registCompressHandler(s, "/server.cgi/get", doGetHead)
	registCompressHandler(s, "/server.cgi/head", doGetHead)
	registCompressHandler(s, "/server.cgi/update", doUpdate)
	registCompressHandler(s, "/server.cgi/recent", doRecent)
	registCompressHandler(s, "/server.cgi/", doMotd)
}

//doPing just resopnse PONG with remote addr.
func doPing(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	fmt.Fprint(w, "PONG\n"+r.RemoteAddr+"\n")
}

//doNode returns one of nodelist. if nodelist.len=0 returns one of initNode.
func doNode(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	if nodeList.Len() > 0 {
		fmt.Fprintln(w, nodeList.nodes[0].nodestr)
	} else {
		fmt.Fprintln(w, initNode.data[0])
	}
}

//doJoin adds node specified in url to searchlist and nodelist.
//if nodelist>#defaultnode removes and says bye one node in nodelist and returns welcome its ip:port.
func doJoin(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	s := newServerCGI(w, r)
	if s == nil {
		return
	}
	n := s.makeNode("join")
	if n == nil {
		return
	}
	if !n.isAllowed() {
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

//doBye  removes from nodelist and says bye to the node specified in url.
func doBye(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	s := newServerCGI(w, r)
	if s == nil {
		return
	}
	n := s.makeNode("bye")
	if n == nil {
		return
	}

	if nodeList.removeNode(n) {
		nodeList.sync()
	}
	fmt.Fprintln(s.wr, "BYEBYE")
}

//doHave checks existance of cache whose name is specified in url.
func doHave(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	s := newServerCGI(w, r)
	if s == nil {
		return
	}
	reg := regexp.MustCompile("^/have/([0-9A-Za-z_]+)$")
	m := reg.FindStringSubmatch(s.path)
	if m == nil {
		fmt.Fprintln(w, "NO")
		log.Println("illegal url")
		return
	}
	ca := newCache(m[1])
	if ca.Len() > 0 {
		fmt.Fprintln(w, "YES")
	} else {
		fmt.Fprintln(w, "NO")
	}
}

//doUpdate adds remote node to searchlist and lookuptable with datfile specified in url.
//if stamp is in range of defaultUpdateRange adds to updateque.
func doUpdate(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	s := newServerCGI(w, r)
	if s == nil {
		return
	}
	reg := regexp.MustCompile("^/update/(\\w+)/(\\d+)/(\\w+)/([^:]*):(\\d+)(.*)")
	m := reg.FindStringSubmatch(s.path)
	if m == nil {
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
	if !n.isAllowed() {
		return
	}
	searchList.append(n)
	searchList.sync()
	lookupTable.add(datfile, n)
	lookupTable.sync(false)
	now := time.Now()
	nstamp, err := strconv.ParseInt(stamp, 10, 64)
	if err != nil {
		log.Println(err)
		return
	}

	if nstamp < now.Add(-defaultUpdateRange).Unix() || nstamp > now.Add(defaultUpdateRange).Unix() {
		return
	}
	rec := newRecord(datfile, stamp+"_"+id)
	if !updateList.hasInfo(rec) {
		queue.append(rec, n)
	}
}

//doRecent renders records whose timestamp is in range of one specified in url.
func doRecent(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	s := newServerCGI(w, r)
	if s == nil {
		return
	}
	reg := regexp.MustCompile("^/recent/?([-0-9A-Za-z/]*)$")
	m := reg.FindStringSubmatch(s.path)
	if m == nil {
		log.Println("illegal url")
		return
	}
	stamp := m[1]
	last := time.Now().Unix() + recentRange
	begin, end, _ := s.parseStamp(stamp, last)
	for _, i := range recentList.infos {
		if begin > i.stamp || i.stamp > end {
			return
		}
		ca := newCache(i.datfile)
		var tagstr string
		if ca.tags != nil {
			tagstr = "tag:" + ca.tags.string()
		}
		_, err := fmt.Fprintf(w, "%d<>%s<>%s%s\n", i.stamp, i.id, i.datfile, tagstr)
		if err != nil {
			log.Println(err)
		}
	}
}

//doMotd simply renders motd file.
func doMotd(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	f, err := ioutil.ReadFile(motd)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintf(w, string(f))
}

//doGetHead renders records contents(get) or id+timestamp(head) who has id and
// whose stamp is in range of one specified by url.
func doGetHead(w http.ResponseWriter, r *http.Request) {
	<-connections
	defer func() {
		connections <- struct{}{}
	}()
	s := newServerCGI(w, r)
	if s == nil {
		return
	}
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
		if begin <= r.Stamp && r.Stamp <= end && (id == "" || strings.HasSuffix(r.Idstr(), id)) {
			if method == "get" {
				err := r.load()
				if err != nil {
					log.Println(err)
				}
				fmt.Fprintf(s.wr, r.recstr())
			} else {
				fmt.Fprintln(s.wr, strings.Replace(r.Idstr(), "_", "<>", -1))
			}
		}
	}
}

//serverCGI is for server.cgi handler.
type serverCGI struct {
	*cgi
}

//newServerCGI set content-type to text and  returns serverCGI obj.
func newServerCGI(w http.ResponseWriter, r *http.Request) *serverCGI {
	c := newCGI(w, r)
	if c == nil {
		return nil
	}
	w.Header().Set("Content-Type", "text/plain")

	return &serverCGI{
		c,
	}
}

//getRemoteHostname returns remoteaddr
//if host is specified returns remoteaddr if host==remoteaddr.
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

//makeNode makes and returns node obj from /method/ip:port.
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

//parseStamp parses format beginStamp - endStamp/id and returns them.
//if endStamp is not specified returns last as endStamp.
func (s *serverCGI) parseStamp(stamp string, last int64) (int64, int64, string) {
	buf := strings.Split(stamp, "/")
	var id string
	if len(buf) > 1 {
		id = buf[1]
		stamp = buf[0]
	}
	buf = strings.Split(stamp, "-")
	nstamps := make([]int64, len(buf))
	var err error
	for i, nstamp := range buf {
		if nstamp == "" {
			continue
		}
		nstamps[i], err = strconv.ParseInt(nstamp, 10, 64)
		if err != nil {
			return 0, 0, ""
		}
	}
	switch {
	case stamp == "", stamp == "-":
		return 0, last, id
	case strings.HasSuffix(stamp, "-"):
		return nstamps[0], last, id
	case len(buf) == 1:
		return nstamps[0], nstamps[0], id
	case buf[0] == "":
		return 0, nstamps[1], id
	default:
		return nstamps[0], nstamps[1], id
	}
}
