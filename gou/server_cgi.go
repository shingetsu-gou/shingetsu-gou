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
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//ServerURL is the url to server.cgi
const ServerURL = "/server.cgi"

//serverSetup setups handlers for server.cgi
func serverSetup(s *loggingServeMux, cfg *Config, g *Global) {
	s.registCompressHandler(ServerURL+"/ping", doPing)
	s.registCompressHandler(ServerURL+"/node", doNode(cfg, g))
	s.registCompressHandler(ServerURL+"/join/", doJoin(cfg, g))
	s.registCompressHandler(ServerURL+"/bye/", doBye(cfg, g))
	s.registCompressHandler(ServerURL+"/have/", doHave(cfg, g))
	s.registCompressHandler(ServerURL+"/get/", doGetHead(cfg, g))
	s.registCompressHandler(ServerURL+"/head/", doGetHead(cfg, g))
	s.registCompressHandler(ServerURL+"/update/", doUpdate(cfg, g))
	s.registCompressHandler(ServerURL+"/recent/", doRecent(cfg, g))
	s.registCompressHandler(ServerURL+"/", doMotd(cfg, g))
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
func doNode(cfg *Config,gl *Global) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if gl.NodeManager.listLen() > 0 {
			fmt.Fprintln(w, gl.NodeManager.getNodestrSliceInTable("")[0])
		} else {
			fmt.Fprintln(w, gl.InitNode.data[0])
		}
	}
}

//doJoin adds node specified in url to searchlist and nodelist.
//if nodelist>#defaultnode removes and says bye one node in nodelist and returns welcome its ip:port.
func doJoin(cfg *Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := newServerCGI(w, r, cfg)
		defer s.close()
		if err != nil {
			log.Println(err)
			return
		}
		n := s.makeNode("join")
		if n == nil {
			return
		}
		if !n.isAllowed() {
			return
		}
		if _, err := n.ping(); err != nil {
			return
		}
		suggest := nodeManager.replaceNodeInList(n)
		if suggest == nil {
			fmt.Fprintln(s.wr, "WELCOME")
			return
		}
		fmt.Fprintf(s.wr, "WELCOME\n%s\n", suggest.nodestr)
	}
}

//doBye  removes from nodelist and says bye to the node specified in url.
func doBye(cfg *Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := newServerCGI(w, r, cfg)
		defer s.close()
		if err != nil {
			log.Println(err)
			return
		}
		n := s.makeNode("bye")
		if n == nil {
			return
		}

		nodeManager.removeFromList(n)
		fmt.Fprintln(s.wr, "BYEBYE")
	}
}

//doHave checks existance of cache whose name is specified in url.
func doHave(cfg *Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := newServerCGI(w, r, cfg)
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
		ca := newCache(m[1])
		if ca.hasRecord() {
			fmt.Fprintln(w, "YES")
		} else {
			fmt.Fprintln(w, "NO")
		}
	}
}

//doUpdate adds remote node to searchlist and lookuptable with datfile specified in url.
//if stamp is in range of defaultUpdateRange adds to updateque.
func doUpdate(cfg *Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := newServerCGI(w, r, cfg)
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

		n := makeNode(host, path, port)
		if !n.isAllowed() {
			log.Println("detects spam")
			return
		}
		nodeManager.appendToTable(datfile, n)
		nodeManager.sync()
		nstamp, err := strconv.ParseInt(stamp, 10, 64)
		if err != nil {
			log.Println(err)
			return
		}

		if !isInUpdateRange(nstamp) {
			return
		}
		rec := newRecord(datfile, stamp+"_"+id)
		go que.updateNodes(rec, n)
		fmt.Fprintln(w, "OK")
	}
}

//doRecent renders records whose timestamp is in range of one specified in url.
func doRecent(cfg *Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := newServerCGI(w, r, cfg)
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
		last := time.Now().Unix() + s.recentRange
		begin, end, _ := s.parseStamp(stamp, last)
		for _, i := range recentList.infos {
			if begin > i.Stamp || i.Stamp > end {
				return
			}
			ca := newCache(i.datfile)
			cont := fmt.Sprintf("%d<>%s<>%s", i.Stamp, i.ID, i.datfile)
			if ca.lenTags() > 0 {
				cont += "<>tag:" + ca.tagString()
			}
			_, err := fmt.Fprintf(w, "%s\n", cont)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

//doMotd simply renders motd file.
func doMotd(cfg *Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := newServerCGI(w, r, cfg)
		defer s.close()
		if err != nil {
			log.Println(err)
			return
		}
		f, err := ioutil.ReadFile(s.motd)
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Fprintf(w, string(f))
	}
}

//doGetHead renders records contents(get) or id+timestamp(head) who has id and
// whose stamp is in range of one specified by url.
func doGetHead(cfg *Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := newServerCGI(w, r, cfg)
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
		ca := newCache(datfile)
		begin, end, id := s.parseStamp(stamp, ca.readInfo().stamp)
		recs := ca.loadRecords()
		for _, r := range recs {
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
}

//serverCGI is for server.cgi handler.
type serverCGI struct {
	*cgi
}

//newServerCGI set content-type to text and  returns serverCGI obj.
func newServerCGI(w http.ResponseWriter, r *http.Request, cfg *Config) (serverCGI, error) {
	c := serverCGI{
		cgi: newCGI(w, r, cfg),
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
func (s *serverCGI) makeNode(method string) *node {
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
