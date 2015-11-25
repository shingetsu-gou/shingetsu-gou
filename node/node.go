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

package node

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shingetsu-gou/go-nat"
	"github.com/shingetsu-gou/http-relay"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

const (
	Disconnected = iota
	Port0
	Normal
)

//Myself contains my node info.
type Myself struct {
	ip           string
	internalPort int
	externalPort *int32
	Path         string
	ServerName   string
	relayServer  *Node
	serveHTTP    http.HandlerFunc
	mutex        sync.RWMutex
	enableNAT    bool
	status       int
}

//NewMyself returns Myself obj.
func NewMyself(internalPort int, path string, serverName string, serveHTTP http.HandlerFunc, enableNAT bool) *Myself {
	p := int32(internalPort)
	return &Myself{
		internalPort: internalPort,
		externalPort: &p,
		Path:         path,
		ServerName:   serverName,
		serveHTTP:    serveHTTP,
		enableNAT:    enableNAT,
	}
}

//resetPort sets externalPort to internalPort.
func (m *Myself) resetPort() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	p := int32(m.internalPort)
	m.externalPort = &p
}

//IsRelayed returns true is relayed.
func (m *Myself) IsRelayed() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.status == Port0 && m.relayServer != nil
}

//GetStatus returns status.
func (m *Myself) GetStatus() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.status
}

//setStatus set connection status.
func (m *Myself) setStatus(stat int) {
	m.mutex.Lock()
	m.status = stat
	m.mutex.Unlock()
}

//IPPortPath returns node ojb contains ip:port/path.
func (m *Myself) IPPortPath() *Node {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	port := int(*m.externalPort)
	n, err := MakeNode(m.ip, m.Path, port)
	if err != nil {
		log.Fatal(err)
	}
	return n
}

//toNode converts myself to *Node.
func (m *Myself) toNode() *Node {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	serverName := m.ServerName
	if m.ServerName == "" {
		serverName = m.ip
	}
	if m.relayServer != nil {
		serverName = m.relayServer.Nodestr + "/relay/" + serverName
	}
	n, err := newNode(fmt.Sprintf("%s:%d%s", serverName, *m.externalPort, m.Path))
	if err != nil {
		log.Fatal(err)
	}
	return n
}

//Nodestr returns nodestr.
func (m *Myself) Nodestr() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.toNode().Nodestr
}

//toxstring returns /->+ nodestr.
func (m *Myself) toxstring() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.toNode().toxstring()
}

//setIP set my IP.
func (m *Myself) setIP(ip string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if ip := net.ParseIP(ip); ip == nil {
		log.Println("ip", ip, "is illegal format")
		return
	}
	m.ip = ip
}

//RelayServer returns nodestr of relay server.
func (m *Myself) RelayServer() string {
	return m.relayServer.Nodestr
}

//tryRelay tries to relay myself for each nodes.
//This returns true if success.
func (m *Myself) tryRelay(nodes []*Node) {
	go func() {
		closed := make(chan struct{})
		for {
			/*
				n, _ := newNode("192.168.1.23:8000/server.cgi")
				nodes = append(nodes, n)
			*/
			success := false
			for _, n := range nodes {
				if n.cannotRelay {
					continue
				}
				log.Println("trying to connect to relay server", n.Nodestr)
				url := "ws://" + n.Nodestr + "/request_relay/"
				err := relay.HandleClient(url, m.serveHTTP, closed, func(r *http.Request) {
					//nothing to do for now
				})
				if err != nil {
					n.cannotRelay = true
					log.Println(err)
				} else {
					success = true
					m.mutex.Lock()
					/*
						n, err = MakeNode("123.230.131.165", "/server.cgi", 8000)
						log.Println(err)
					*/
					m.relayServer = n
					m.mutex.Unlock()
					<-closed
					m.relayServer = nil
				}
			}
			if !success {
				log.Println("cannot find relay server,sleeping...")
				time.Sleep(10 * time.Minute)
			}
		}
	}()
}

//proxyURL returns url for proxy if relayed.
func (m *Myself) proxyURL(path string) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if m.relayServer == nil {
		return path
	}
	/*
		ssss, err := MakeNode("192.168.1.23", "/server.cgi", 8000)
		log.Println(err)
		return ssss.Nodestr + "/proxy/" + path
	*/
	return m.relayServer.Nodestr + "/proxy/" + path
}

//useUPnP gets external port by upnp and return external port.
//returns defaultPort if failed.
func (m *Myself) useUPnP() bool {
	if !m.enableNAT {
		return false
	}
	nt, err := nat.NewNetStatus()
	if err != nil {
		log.Println(err)
		return false
	} else {
		ma, err := nt.LoopPortMapping("tcp", m.internalPort, "shingetsu-gou", 10*time.Minute)
		if err != nil {
			log.Println(err)
		} else {
			m.externalPort = ma.ExternalPort
		}
		return true
	}
}

//NodeCfg is a global stuf for Node struct. it must be set before using it.
var NodeCfg *NodeConfig

//NodeConfig contains params for Node struct.
type NodeConfig struct {
	Myself    *Myself
	NodeAllow *util.RegexpList
	NodeDeny  *util.RegexpList
	Version   string
}

//Node represents node info.
type Node struct {
	*NodeConfig
	Nodestr     string
	cannotRelay bool
}

//NewNode checks nodestr format and returns node obj.
func newNode(nodestr string) (*Node, error) {
	if NodeCfg == nil {
		log.Fatal("must set NodeCfg")
	}
	nodestr = strings.TrimSpace(nodestr)
	if nodestr == "" {
		err := errors.New("nodestr is empty")
		log.Println(err)
		return nil, err
	}
	if match, err := regexp.MatchString(`\d+/[^: ]+$`, nodestr); !match || err != nil {
		err := errors.New(fmt.Sprintln("bad format", err, nodestr))
		return nil, err
	}
	n := &Node{
		NodeConfig: NodeCfg,
		Nodestr:    strings.Replace(nodestr, "+", "/", -1),
	}
	return n, nil
}

//urlopen retrievs html data from url
func (n *Node) urlopen(url string, timeout time.Duration, fn func(string) error) error {
	ua := "shinGETsuPlus/0.8alpha (Gou/" + n.Version + ")"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", ua)

	transport := http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, timeout)
		},
	}

	client := http.Client{
		Transport: &transport,
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return err
	}
	err = util.EachIOLine(resp.Body, func(line string, i int) error {
		return fn(line)
	})
	return err
}

//Equals return true is Nodestr is equal.
func (n *Node) Equals(nn *Node) bool {
	if nn == nil {
		return false
	}
	return n.Nodestr == nn.Nodestr
}

//MakeNode makes node from host info.
func MakeNode(host, path string, port int) (*Node, error) {
	nodestr := net.JoinHostPort(host, strconv.Itoa(port)) + strings.Replace(path, "+", "/", -1)
	return newNode(nodestr)
}

//toxstring covnerts Nodestr to saku node format.
func (n *Node) toxstring() string {
	return strings.Replace(n.Nodestr, "/", "+", -1)
}

//Talk talks with n with the message and returns data.
func (n *Node) Talk(message string, proxy bool, fn func(string) error) ([]string, error) {
	const defaultTimeout = time.Minute // Seconds; Timeout for TCP
	var res []string
	if fn == nil {
		fn = func(line string) error {
			res = append(res, line)
			return nil
		}
	}

	if !strings.HasPrefix(message, "/") {
		message = "/" + message
	}
	if n == nil {
		err := errors.New("n==nil")
		log.Println(err)
		return nil, err
	}
	msg := "http://" + n.Nodestr + message
	if proxy {
		msg = "http://" + n.Myself.proxyURL(n.Nodestr+message)
	}
	log.Println("Talk:", msg)
	err := n.urlopen(msg, defaultTimeout, fn)
	if err != nil {
		log.Println(msg, err)
	}
	return res, err
}

//Ping pings to n and return response.
func (n *Node) Ping() (string, error) {
	res, err := n.Talk("/ping", false, nil)
	if err != nil {
		log.Println("/ping", n.Nodestr, err)
		return "", err
	}
	if res[0] == "PONG" && len(res) == 2 {
		log.Println("ponged,i am", res[1])
		n.Myself.setIP(res[1])
		return res[1], nil
	}
	log.Println("/ping", n.Nodestr, "error")
	return "", errors.New("connected,but not ponged")
}

//IsAllowed returns fase if n is not allowed and denied.
func (n *Node) IsAllowed() bool {
	if !n.NodeAllow.Check(n.Nodestr) && n.NodeDeny.Check(n.Nodestr) {
		return false
	}
	return true
}

//join requests n to join me and return true and other node name if success.
func (n *Node) join() (*Node, error) {
	if !n.IsAllowed() {
		err := errors.New(fmt.Sprintln(n.Nodestr, "is not allowd"))
		return nil, err
	}
	res, err := n.Talk("/join/"+n.Myself.toxstring(), true, nil)
	if err != nil {
		return nil, err
	}
	log.Println("response of join:", res)
	switch len(res) {
	case 0:
		return nil, errors.New("illegal response")
	case 1:
		if res[0] != "WELCOME" {
			return nil, errors.New("not welcomed")
		}
		return nil, nil
	}
	nn, err := newNode(res[1])
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if res[0] != "WELCOME" {
		err = errors.New("not welcomed")
	}
	return nn, err
}

//getNode requests n to pass me another node info and returns another node.
func (n *Node) getNode() (*Node, error) {
	res, err := n.Talk("/node", false, nil)
	if err != nil {
		err := errors.New(fmt.Sprintln("/node", n.Nodestr, "error"))
		return nil, err
	}
	return newNode(res[0])
}

//bye says goodbye to n and returns true if success.
func (n *Node) bye() bool {
	res, err := n.Talk("/bye/"+n.Myself.Nodestr(), true, nil)
	if err != nil {
		log.Println("/bye", n.Nodestr, "error")
		return false
	}
	return len(res) > 0 && (res[0] == "BYEBYE")
}

//getherNodes gethers nodes from n.
func (n *Node) getherNodes() []*Node {
	ns := map[string]*Node{
		n.Nodestr: n,
	}
	for i := 0; i < 10; i++ {
		var mutex sync.Mutex
		var wg sync.WaitGroup
		for _, nn := range ns {
			wg.Add(1)
			go func(nn *Node) {
				defer wg.Done()
				newN, err := nn.getNode()
				if err != nil {
					log.Println(err)
					return
				}
				mutex.Lock()
				ns[newN.Nodestr] = newN
				mutex.Unlock()
			}(nn)
		}
		log.Println(i)
		wg.Wait()
	}
	nss := make([]*Node, len(ns))
	var i int
	for _, nn := range ns {
		nss[i] = nn
		i++
	}
	return nss
}

//Slice is slice of node.
type Slice []*Node

//Len returns size of nodes.
func (ns Slice) Len() int {
	return len(ns)
}

//Swap swaps nodes order.
func (ns Slice) Swap(i, j int) {
	ns[i], ns[j] = ns[j], ns[i]
}

//Has returns true if ns has n.
func (ns Slice) Has(n *Node) bool {
	return util.HasString(ns.getNodestrSlice(), n.Nodestr)
}

//getNodestrSlice returns slice of Nodestr of nodes.
func (ns Slice) getNodestrSlice() []string {
	s := make([]string, ns.Len())
	for i, v := range ns {
		s[i] = v.Nodestr
	}
	return s
}

//tomap returns map[nodestr]struct{}{} for searching a node.
func (ns Slice) toMap() map[string]struct{} {
	m := make(map[string]struct{})
	for _, nn := range ns {
		m[nn.Nodestr] = struct{}{}
	}
	return m
}

//uniq solidate the slice.
func (ns Slice) uniq() Slice {
	m := make(map[string]struct{})
	ret := make([]*Node, 0, ns.Len())

	for _, n := range ns {
		if _, exist := m[n.Nodestr]; !exist {
			m[n.Nodestr] = struct{}{}
			ret = append(ret, n)
		}
	}
	return ret
}

//Extend make a new nodeslice including specified slices.
func (ns Slice) Extend(a Slice) Slice {
	nn := make([]*Node, ns.Len()+a.Len())
	copy(nn, ns)
	copy(nn[ns.Len():], a)
	return nn
}
