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

package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shingetsu-gou/go-nat"
	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/cgi"
	"github.com/shingetsu-gou/shingetsu-gou/node"
	"github.com/shingetsu-gou/shingetsu-gou/node/manager"
	"github.com/shingetsu-gou/shingetsu-gou/recentlist"
	"github.com/shingetsu-gou/shingetsu-gou/record"
	"github.com/shingetsu-gou/shingetsu-gou/tag/user"
	"github.com/shingetsu-gou/shingetsu-gou/thread"
	"github.com/shingetsu-gou/shingetsu-gou/updateque"
)

//Setup setups handlers for server.cgi
func Setup(s *cgi.LoggingServeMux) {
	s.RegistCompressHandler(cfg.ServerURL+"/ping", doPing)
	s.RegistCompressHandler(cfg.ServerURL+"/node", doNode)
	s.RegistCompressHandler(cfg.ServerURL+"/join/", doJoin)
	s.RegistCompressHandler(cfg.ServerURL+"/bye/", doBye)
	s.RegistCompressHandler(cfg.ServerURL+"/have/", doHave)
	s.RegistCompressHandler(cfg.ServerURL+"/get/", doGetHead)
	s.RegistCompressHandler(cfg.ServerURL+"/head/", doGetHead)
	s.RegistCompressHandler(cfg.ServerURL+"/update/", doUpdate)
	s.RegistCompressHandler(cfg.ServerURL+"/recent/", doRecent)
	s.RegistCompressHandler(cfg.ServerURL+"/", doMotd)

}

//doPing just resopnse PONG with remote addr.
func doPing(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Header)
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprint(w, "PONG\n"+host+"\n")
}

//doNode returns one of nodelist. if nodelist.len=0 returns one of initNode.
func doNode(w http.ResponseWriter, r *http.Request) {
	if manager.ListLen() > 0 {
		fmt.Fprintln(w, manager.GetNodestrSliceInTable("")[0])
	} else {
		fmt.Fprintln(w, node.InitNode.GetData()[0])
	}
}

//doJoin adds node specified in url to searchlist and nodelist.
//if nodelist>#defaultnode removes and says bye one node in nodelist and returns welcome its ip:port.
func doJoin(w http.ResponseWriter, r *http.Request) {
	s, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	host, path, port := s.extractHost("join")
	host = s.remoteIP(host)
	if host == "" {
		return
	}
	n, err := node.MakeNode(host, path, port)
	if err != nil || !n.IsAllowed() {
		return
	}
	if _, err := n.Ping(); err != nil {
		return
	}
	suggest := manager.ReplaceNodeInList(n)
	if suggest == nil {
		fmt.Fprintln(s.WR, "WELCOME")
		return
	}
	fmt.Fprintf(s.WR, "WELCOME\n%s\n", suggest.Nodestr)
}

//doBye  removes from nodelist and says bye to the node specified in url.
func doBye(w http.ResponseWriter, r *http.Request) {
	s, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	host, path, port := s.extractHost("bye")
	host = s.checkRemote(host)
	if host == "" {
		return
	}
	n, err := node.MakeNode(host, path, port)
	if err == nil {
		manager.RemoveFromList(n)
	}
	fmt.Fprintln(s.WR, "BYEBYE")
}

//doHave checks existance of cache whose name is specified in url.
func doHave(w http.ResponseWriter, r *http.Request) {
	s, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	reg := regexp.MustCompile("^have/([0-9A-Za-z_]+)$")
	m := reg.FindStringSubmatch(s.Path())
	if m == nil {
		fmt.Fprintln(w, "NO")
		log.Println("illegal url")
		return
	}
	ca := thread.NewCache(m[1])
	if ca.HasRecord() {
		fmt.Fprintln(w, "YES")
	} else {
		fmt.Fprintln(w, "NO")
	}
}

//doUpdate adds remote node to searchlist and lookuptable with datfile specified in url.
//if stamp is in range of defaultUpdateRange adds to updateque.
func doUpdate(w http.ResponseWriter, r *http.Request) {
	s, err := new(w, r)
	if err != nil {
		log.Println(err)
		log.Println("failed to create cgi struct")
		return
	}
	reg := regexp.MustCompile(`^update/(\w+)/(\d+)/(\w+)/([^\+]*)(\+.*)`)
	m := reg.FindStringSubmatch(s.Path())
	if m == nil {
		log.Println("illegal url")
		return
	}
	datfile, stamp, id, hostport, path := m[1], m[2], m[3], m[4], m[5]
	host, portstr, err := net.SplitHostPort(hostport)
	if err != nil {
		log.Println(err)
		return
	}
	port, err := strconv.Atoi(portstr)
	if err != nil {
		log.Println(err)
		return
	}
	host = s.remoteIP(host)
	if host == "" {
		log.Println("host is null")
		return
	}

	n, err := node.MakeNode(host, path, port)
	if err != nil || !n.IsAllowed() {
		log.Println("detects spam")
		return
	}
	manager.AppendToTable(datfile, n)
	nstamp, err := strconv.ParseInt(stamp, 10, 64)
	if err != nil {
		log.Println(err)
		return
	}

	if !recentlist.IsInUpdateRange(nstamp) {
		return
	}
	rec := record.New(datfile, id, nstamp)
	go updateque.UpdateNodes(rec, n)
	fmt.Fprintln(w, "OK")
}

//doRecent renders records whose timestamp is in range of one specified in url.
func doRecent(w http.ResponseWriter, r *http.Request) {
	s, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	reg := regexp.MustCompile("^recent/?([-0-9A-Za-z/]*)$")
	m := reg.FindStringSubmatch(s.Path())
	if m == nil {
		log.Println("illegal url")
		return
	}
	stamp := m[1]
	last := time.Now().Unix() + cfg.RecentRange
	begin, end, _ := s.parseStamp(stamp, last)
	for _, i := range recentlist.GetRecords() {
		if begin > i.Stamp || i.Stamp > end {
			continue
		}
		ca := thread.NewCache(i.Datfile)
		cont := fmt.Sprintf("%d<>%s<>%s", i.Stamp, i.ID, i.Datfile)
		if user.Len(ca.Datfile) > 0 {
			cont += "<>tag:" + user.String(ca.Datfile)
		}
		_, err := fmt.Fprintf(w, "%s\n", cont)
		if err != nil {
			log.Println(err)
		}
	}
}

//doMotd simply renders motd file.
func doMotd(w http.ResponseWriter, r *http.Request) {
	f, err := ioutil.ReadFile(cfg.Motd())
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintf(w, string(f))
}

//doGetHead renders records contents(get) or id+timestamp(head) who has id and
// whose stamp is in range of one specified by url.
func doGetHead(w http.ResponseWriter, r *http.Request) {
	s, err := new(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	reg := regexp.MustCompile("^(get|head|removed)/([0-9A-Za-z_]+)/?([-0-9A-Za-z/]*)$")
	m := reg.FindStringSubmatch(s.Path())
	if m == nil {
		log.Println("illegal url", s.Path())
		return
	}
	method, datfile, stamp := m[1], m[2], m[3]
	ca := thread.NewCache(datfile)
	begin, end, id := s.parseStamp(stamp, math.MaxInt32)
	var recs record.Map
	if method == "removed" {
		recs = ca.LoadRecords(record.Removed)
	} else {
		recs = ca.LoadRecords(record.Alive)
	}
	for _, r := range recs {
		if r.InRange(begin, end, id) {
			if method == "get" {
				if err := r.Load(); err != nil {
					log.Println(err)
					continue
				}
				fmt.Fprintln(s.WR, r.Recstr())
				continue
			}
			fmt.Fprintln(s.WR, strings.Replace(r.Idstr(), "_", "<>", -1))
		}
	}
	if method == "get" {
		updateque.UpdatedRecord.Inform(datfile, id, begin, end)
	}
}

//serverCGI is for server.cgi handler.
type serverCGI struct {
	*cgi.CGI
}

//new set content-type to text and  returns serverCGI obj.
func new(w http.ResponseWriter, r *http.Request) (*serverCGI, error) {
	c, err := cgi.NewCGI(w, r)
	if err != nil {
		return nil, err
	}
	a := serverCGI{
		CGI: c,
	}

	if w != nil {
		w.Header().Set("Content-Type", "text/plain")
	}

	return &a, nil
}

//remoteIP returns host if host!=""
//else returns remoteaddr
func (s *serverCGI) remoteIP(host string) string {
	if host != "" {
		return host
	}
	remoteAddr, _, err := net.SplitHostPort(s.Req.RemoteAddr)
	if err != nil {
		log.Println(err)
		return ""
	}
	if !isGlobal(remoteAddr) {
		log.Println(remoteAddr, "is local IP")
		return ""
	}
	return remoteAddr
}

func isGlobal(remoteAddr string) bool {
	ip := net.ParseIP(remoteAddr)
	if ip == nil {
		log.Println(remoteAddr, "has illegal format")
		return false
	}
	return nat.IsGlobalIP(ip) != ""
}

//checkRemote returns remoteaddr
//if host is specified returns remoteaddr if host==remoteaddr.
func (s *serverCGI) checkRemote(host string) string {
	remoteAddr, _, err := net.SplitHostPort(s.Req.RemoteAddr)
	if err != nil {
		log.Println(err)
		return ""
	}
	if host == "" {
		return remoteAddr
	}
	ipaddr, err := net.LookupIP(host)
	if err != nil {
		log.Println(err)
		return ""
	}
	for _, ipa := range ipaddr {
		if ipa.String() == remoteAddr {
			return remoteAddr
		}
	}
	return ""
}

//makeNode makes and returns node obj from /method/ip:port.
func (s *serverCGI) extractHost(method string) (string, string, int) {
	reg := regexp.MustCompile("^" + method + `/([^\+]*)(\+.*)`)
	m := reg.FindStringSubmatch(s.Path())
	if m == nil {
		log.Println("illegal url")
		return "", "", 0
	}
	path := m[2]
	host, portstr, err := net.SplitHostPort(m[1])
	if err != nil {
		log.Println(err)
		return "", "", 0
	}
	port, err := strconv.Atoi(portstr)
	if err != nil {
		log.Println(err)
		return "", "", 0
	}
	return host, path, port
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
