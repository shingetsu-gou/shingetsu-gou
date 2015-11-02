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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/node"
	"github.com/shingetsu-gou/shingetsu-gou/thread"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//ServerURL is the url to server.cgi
const ServerURL = "/server.cgi"

//ServerSetup setups handlers for server.cgi
func ServerSetup(s *LoggingServeMux) {
	s.RegistCompressHandler(ServerURL+"/ping", doPing)
	s.RegistCompressHandler(ServerURL+"/node", doNode)
	s.RegistCompressHandler(ServerURL+"/join/", doJoin)
	s.RegistCompressHandler(ServerURL+"/bye/", doBye)
	s.RegistCompressHandler(ServerURL+"/have/", doHave)
	s.RegistCompressHandler(ServerURL+"/get/", doGetHead)
	s.RegistCompressHandler(ServerURL+"/head/", doGetHead)
	s.RegistCompressHandler(ServerURL+"/update/", doUpdate)
	s.RegistCompressHandler(ServerURL+"/recent/", doRecent)
	s.RegistCompressHandler(ServerURL+"/", doMotd)
}

//doPing just resopnse PONG with remote addr.
func doPing(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprint(w, "PONG\n"+host+"\n")
}

//doNode returns one of nodelist. if nodelist.len=0 returns one of initNode.
func doNode(w http.ResponseWriter, r *http.Request) {
	s, err := newServerCGI(w, r)
	defer s.close()
	if err != nil {
		log.Println(err)
		return
	}
	if s.NodeManager.ListLen() > 0 {
		fmt.Fprintln(w, s.NodeManager.GetNodestrSliceInTable("")[0])
	} else {
		fmt.Fprintln(w, s.InitNode.GetData()[0])
	}
}

//doJoin adds node specified in url to searchlist and nodelist.
//if nodelist>#defaultnode removes and says bye one node in nodelist and returns welcome its ip:port.
func doJoin(w http.ResponseWriter, r *http.Request) {
	s, err := newServerCGI(w, r)
	defer s.close()
	if err != nil {
		log.Println(err)
		return
	}
	n := s.makeNode("join")
	if n == nil {
		return
	}
	if !n.IsAllowed() {
		return
	}
	if _, err := n.Ping(); err != nil {
		return
	}
	suggest := s.NodeManager.ReplaceNodeInList(n)
	if suggest == nil {
		fmt.Fprintln(s.wr, "WELCOME")
		return
	}
	fmt.Fprintf(s.wr, "WELCOME\n%s\n", suggest.Nodestr)
}

//doBye  removes from nodelist and says bye to the node specified in url.
func doBye(w http.ResponseWriter, r *http.Request) {
	s, err := newServerCGI(w, r)
	defer s.close()
	if err != nil {
		log.Println(err)
		return
	}
	n := s.makeNode("bye")
	if n == nil {
		return
	}

	s.NodeManager.RemoveFromList(n)
	fmt.Fprintln(s.wr, "BYEBYE")
}

//doHave checks existance of cache whose name is specified in url.
func doHave(w http.ResponseWriter, r *http.Request) {
	s, err := newServerCGI(w, r)
	defer s.close()
	if err != nil {
		log.Println(err)
		return
	}
	reg := regexp.MustCompile("^have/([0-9A-Za-z_]+)$")
	m := reg.FindStringSubmatch(s.path())
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
	s, err := newServerCGI(w, r)
	defer s.close()
	if err != nil {
		log.Println(err)
		log.Println("failed to create cgi struct")
		return
	}
	reg := regexp.MustCompile(`^update/(\w+)/(\d+)/(\w+)/([^:]*):(\d+)(.*)`)
	m := reg.FindStringSubmatch(s.path())
	if m == nil {
		log.Println("illegal url")
		return
	}
	datfile, stamp, id, host, path := m[1], m[2], m[3], m[4], m[6]
	port, err := strconv.Atoi(m[5])
	if err != nil {
		log.Println(err)
		return
	}
	host = s.getRemoteHostname(host)
	if host == "" {
		log.Println("host is null")
		return
	}

	n := node.MakeNode(host, path, port)
	if !n.IsAllowed() {
		log.Println("detects spam")
		return
	}
	s.NodeManager.AppendToTable(datfile, n)
	s.NodeManager.Sync()
	nstamp, err := strconv.ParseInt(stamp, 10, 64)
	if err != nil {
		log.Println(err)
		return
	}

	if !thread.IsInUpdateRange(nstamp) {
		return
	}
	rec := thread.NewRecord(datfile, stamp+"_"+id)
	go s.UpdateQue.UpdateNodes(rec, n)
	fmt.Fprintln(w, "OK")
}

//doRecent renders records whose timestamp is in range of one specified in url.
func doRecent(w http.ResponseWriter, r *http.Request) {
	s, err := newServerCGI(w, r)
	defer s.close()
	if err != nil {
		log.Println(err)
		return
	}
	reg := regexp.MustCompile("^recent/?([-0-9A-Za-z/]*)$")
	m := reg.FindStringSubmatch(s.path())
	if m == nil {
		log.Println("illegal url")
		return
	}
	stamp := m[1]
	last := time.Now().Unix() + s.RecentRange
	begin, end, _ := s.parseStamp(stamp, last)
	for _, i := range s.RecentList.GetRecords() {
		if begin > i.Stamp || i.Stamp > end {
			return
		}
		ca := thread.NewCache(i.Datfile)
		cont := fmt.Sprintf("%d<>%s<>%s", i.Stamp, i.ID, i.Datfile)
		if ca.LenTags() > 0 {
			cont += "<>tag:" + ca.TagString()
		}
		_, err := fmt.Fprintf(w, "%s\n", cont)
		if err != nil {
			log.Println(err)
		}
	}

}

//doMotd simply renders motd file.
func doMotd(w http.ResponseWriter, r *http.Request) {
	s, err := newServerCGI(w, r)
	defer s.close()
	if err != nil {
		log.Println(err)
		return
	}
	f, err := ioutil.ReadFile(s.Motd)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintf(w, string(f))
}

//doGetHead renders records contents(get) or id+timestamp(head) who has id and
// whose stamp is in range of one specified by url.
func doGetHead(w http.ResponseWriter, r *http.Request) {
	s, err := newServerCGI(w, r)
	defer s.close()
	if err != nil {
		log.Println(err)
		return
	}
	reg := regexp.MustCompile("^(get|head)/([0-9A-Za-z_]+)/?([-0-9A-Za-z/]*)$")
	m := reg.FindStringSubmatch(s.path())
	if m == nil {
		log.Println("illegal url", s.path())
		return
	}
	method, datfile, stamp := m[1], m[2], m[3]
	ca := thread.NewCache(datfile)
	begin, end, id := s.parseStamp(stamp, ca.ReadInfo().Stamp)
	recs := ca.LoadRecords()
	for _, r := range recs {
		if begin <= r.Stamp && r.Stamp <= end && (id == "" || strings.HasSuffix(r.Idstr(), id)) {
			if method == "get" {
				err := r.Load()
				if err != nil {
					log.Println(err)
				}
				fmt.Fprintf(s.wr, r.Recstr())
			} else {
				fmt.Fprintln(s.wr, strings.Replace(r.Idstr(), "_", "<>", -1))
			}
		}
	}
}

//ServerCfg is config for serverCGI struct.
//must set beforehand.
var ServerCfg *ServerConfig

//ServerConfig is config for serverCGI struct.
type ServerConfig struct {
	RecentRange int64
	Motd        string
	NodeManager *node.Manager
	InitNode    *util.ConfList
	UpdateQue   *thread.UpdateQue
	RecentList  *thread.RecentList
}

//serverCGI is for server.cgi handler.
type serverCGI struct {
	*ServerConfig
	*cgi
}

//newServerCGI set content-type to text and  returns serverCGI obj.
func newServerCGI(w http.ResponseWriter, r *http.Request) (serverCGI, error) {
	if ServerCfg == nil {
		log.Fatal("must set ServerCfg")
	}
	c := serverCGI{
		ServerConfig: ServerCfg,
		cgi:          newCGI(w, r),
	}
	if c.cgi == nil {
		return c, errors.New("cannot make CGI")
	}
	w.Header().Set("Content-Type", "text/plain")

	return c, nil
}

//getRemoteHostname returns remoteaddr
//if host is specified returns remoteaddr if host==remoteaddr.
func (s *serverCGI) getRemoteHostname(host string) string {
	remoteAddr, _, err := net.SplitHostPort(s.req.RemoteAddr)
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
			return host
		}
	}
	return ""
}

//makeNode makes and returns node obj from /method/ip:port.
func (s *serverCGI) makeNode(method string) *node.Node {
	reg := regexp.MustCompile("^" + method + `/([^:]*):(\d+)(.*)`)
	m := reg.FindStringSubmatch(s.path())
	if m == nil {
		log.Println("illegal url")
		return nil
	}
	path := m[3]
	host := s.getRemoteHostname(m[1])
	if host == "" {
		log.Println("no host")
		return nil
	}
	port, err := strconv.Atoi(m[2])
	if err != nil {
		log.Println(err)
		return nil
	}
	return node.MakeNode(host, path, port)
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