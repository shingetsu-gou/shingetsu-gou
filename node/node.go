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
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//Version , same as one in main package.
var Version string

//urlopen retrievs html data from url
func urlopen(url string, timeout time.Duration) ([]string, error) {
	ua := "shinGETsu/0.7 (Gou/" + Version + ")"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", ua)

	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	var lines []string
	err = util.EachIOLine(resp.Body, func(line string, i int) error {
		strings.TrimRight(line, "\r\n")
		lines = append(lines, line)
		return nil
	})
	return lines, err
}

//Myself contains my node info.
type Myself struct {
	IP         string
	Port       int
	Path       string
	ServerName string
	mutex      sync.RWMutex
}

//IpPortPath returns node ojb contains ip:port/path.
func (m *Myself) IPPortPath() *Node {
	return MakeNode(m.IP, m.Path, m.Port)
}

//toNode converts myself to *Node.
func (m *Myself) toNode() *Node {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if m.ServerName != "" {
		return MakeNode(m.ServerName, m.Path, m.Port)
	}
	return MakeNode("", m.Path, m.Port)
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

//NodeCfg is a global stuf for Node struct. it must be set before using it.
var NodeCfg *NodeConfig

//NodeConfig contains params for Node struct.
type NodeConfig struct {
	Myself    *Myself
	NodeAllow *util.RegexpList
	NodeDeny  *util.RegexpList
}

//Node represents node info.
type Node struct {
	*NodeConfig
	Nodestr string
	mutex   sync.RWMutex
}

//NewNode checks nodestr format and returns node obj.
func NewNode(nodestr string) *Node {
	if NodeCfg == nil {
		log.Fatal("must set NodeCfg")
	}
	nodestr = strings.TrimSpace(nodestr)
	if nodestr == "" {
		log.Printf("nodestr is empty")
		return nil
	}
	if match, err := regexp.MatchString(`\d+/[^: ]+$`, nodestr); !match || err != nil {
		log.Println("bad format", err, nodestr)
		return nil
	}
	n := &Node{
		NodeConfig: NodeCfg,
		Nodestr:    strings.Replace(nodestr, "+", "/", -1),
	}
	return n
}

//equals return true is Nodestr is equal.
func (n *Node) equals(nn *Node) bool {
	if nn == nil {
		return false
	}
	return n.Nodestr == nn.Nodestr
}

//MakeNode makes node from host info.
func MakeNode(host, path string, port int) *Node {
	nodestr := net.JoinHostPort(host, strconv.Itoa(port)) + strings.Replace(path, "+", "/", -1)
	return NewNode(nodestr)
}

//toxstring covnerts Nodestr to saku node format.
func (n *Node) toxstring() string {
	return strings.Replace(n.Nodestr, "/", "+", -1)
}

//Talk talks with n with the message and returns data.
func (n *Node) Talk(message string) ([]string, error) {
	const defaultTimeout = 20 * time.Second // Seconds; Timeout for TCP

	const getTimeout = 20 * time.Second // Seconds; Timeout for /get

	if !strings.HasPrefix(message, "/") {
		message = "/" + message
	}
	var timeout time.Duration
	if !strings.HasPrefix(message, "/get") {
		timeout = getTimeout
	} else {
		timeout = defaultTimeout
	}
	if n == nil {
		panic("nil!")
	}

	message = "http://" + n.Nodestr + message
	log.Println("Talk:", message)
	res, err := urlopen(message, timeout)
	if err != nil {
		log.Println(message, err)
	}
	return res, err
}

//Ping pings to n and return response.
func (n *Node) Ping() (string, error) {
	res, err := n.Talk("/ping")
	if err != nil {
		log.Println("/ping", n.Nodestr, err)
		return "", err
	}
	if res[0] == "PONG" && len(res) == 2 {
		log.Println("ponged,i am", res[1])
		n.Myself.mutex.Lock()
		defer n.Myself.mutex.Unlock()
		n.Myself.IP = res[1]
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
func (n *Node) join() (bool, *Node) {
	if !n.IsAllowed() {
		log.Println(n.Nodestr, "is not allowd")
		return false, nil
	}
	res, err := n.Talk("/join/" + n.Myself.toxstring())
	if err != nil {
		return false, nil
	}
	log.Println(res)
	switch len(res) {
	case 0:
		return false, nil
	case 1:
		return res[0] == "WELCOME", nil
	}
	return (res[0] == "WELCOME"), NewNode(res[1])
}

//getNode request n to pass me another node info and returns another node.
func (n *Node) getNode() *Node {
	res, err := n.Talk("/node")
	if err != nil {
		log.Println("/node", n.Nodestr, "error")
		return nil
	}
	return NewNode(res[0])
}

//bye says goodbye to n and returns true if success.
func (n *Node) bye() bool {
	res, err := n.Talk("/bye/" + n.Myself.Nodestr())
	if err != nil {
		log.Println("/bye", n.Nodestr, "error")
		return false
	}
	return (res[0] == "BYEBYE")
}

type nodeSlice []*Node

//Len returns size of nodes.
func (ns nodeSlice) Len() int {
	return len(ns)
}

//Swap swaps nodes order.
func (ns nodeSlice) Swap(i, j int) {
	ns[i], ns[j] = ns[j], ns[i]
}

//getNodestrSlice returns slice of Nodestr of nodes.
func (ns nodeSlice) getNodestrSlice() []string {
	s := make([]string, ns.Len())
	for i, v := range ns {
		s[i] = v.Nodestr
	}
	return s
}

//tomap returns map[nodestr]struct{}{} for searching a node.
func (ns nodeSlice) toMap() map[string]struct{} {
	m := make(map[string]struct{})
	for _, nn := range ns {
		m[nn.Nodestr] = struct{}{}
	}
	return m
}

//uniq solidate the slice.
func (ns nodeSlice) uniq() nodeSlice {
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

//extend make a new nodeslice including specified slices.
func (ns nodeSlice) extend(a nodeSlice) nodeSlice {
	nn := make([]*Node, ns.Len()+a.Len())
	copy(nn, ns)
	copy(nn[ns.Len():], a)
	return nn
}
