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

	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/myself"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//Node represents node info.
type Node struct {
	Nodestr string
}

//MustNew makes node slice from names.
func MustNew(names []string) []*Node {
	ns := make([]*Node, len(names))
	for i, nn := range names {
		nn, err := New(nn)
		if err != nil {
			log.Fatal(err)
		}
		ns[i] = nn
	}
	return ns
}

//New checks nodestr format and returns node obj.
func New(nodestr string) (*Node, error) {
	nodestr = strings.TrimSpace(nodestr)
	if nodestr == "" {
		err := errors.New("nodestr is empty")
		log.Println(err)
		return nil, err
	}
	if match, err := regexp.MatchString(`\d+/[^: ]+$`, nodestr); !match || err != nil {
		errr := errors.New(fmt.Sprintln("bad format", err, nodestr))
		return nil, errr
	}
	n := &Node{
		Nodestr: strings.Replace(nodestr, "+", "/", -1),
	}
	return n, nil
}

//urlopen retrievs html data from url
func (n *Node) urlopen(url string, timeout time.Duration, fn func(string) error) error {
	ua := "shinGETsuPlus/1.0alpha (Gou/" + cfg.Version + ")"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
		return err
	}
	req.Header.Set("User-Agent", ua)

	transport := http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			con, errr := net.DialTimeout(network, addr, timeout)
			if errr != nil {
				return nil, errr
			}
			errr = con.SetDeadline(time.Now().Add(time.Minute))
			return con, errr
		},
	}

	client := http.Client{
		Transport: &transport,
		Timeout:   timeout,
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
	return New(nodestr)
}

//Toxstring covnerts Nodestr to saku node format.
func (n *Node) Toxstring() string {
	return strings.Replace(n.Nodestr, "/", "+", -1)
}

//Talk talks with n with the message and returns data.
func (n *Node) Talk(message string, fn func(string) error) ([]string, error) {
	const defaultTimeout = 15 * time.Second // Seconds; Timeout for TCP
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

	log.Println("Talk:", msg)
	err := n.urlopen(msg, defaultTimeout, fn)
	if err != nil {
		log.Println(msg, err)
	}
	return res, err
}

//Ping pings to n and return response.
func (n *Node) Ping() (string, error) {
	res, err := n.Talk("/ping", nil)
	if err != nil {
		log.Println("/ping", n.Nodestr, err)
		return "", err
	}
	if len(res) == 2 && res[0] == "PONG" {
		log.Println("ponged,i am", res[1])
		myself.SetIP(res[1])
		return res[1], nil
	}
	log.Println("/ping", n.Nodestr, "error")
	return "", errors.New("connected,but not ponged")
}

//IsAllowed returns fase if n is not allowed and denied.
func (n *Node) IsAllowed() bool {
	nodeAllow := util.NewRegexpList(cfg.NodeAllowFile)
	nodeDeny := util.NewRegexpList(cfg.NodeDenyFile)

	if !nodeAllow.Check(n.Nodestr) && nodeDeny.Check(n.Nodestr) {
		return false
	}
	return true
}

//Join requests n to Join me and return true and other node name if success.
func (n *Node) Join() (*Node, error) {
	if !n.IsAllowed() {
		err := errors.New(fmt.Sprintln(n.Nodestr, "is not allowd"))
		return nil, err
	}
	res, err := n.Talk("/join/"+Me(true).Toxstring(), nil)
	if err != nil {
		return nil, err
	}
	log.Println(n.Nodestr, "response of Join:", res)
	switch len(res) {
	case 0:
		return nil, errors.New("illegal response")
	case 1:
		if res[0] != "WELCOME" {
			return nil, errors.New("not welcomed")
		}
		return nil, nil
	}
	nn, err := New(res[1])
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
	res, err := n.Talk("/node", nil)
	if err != nil {
		err := errors.New(fmt.Sprintln("/node", n.Nodestr, "error"))
		return nil, err
	}
	if len(res) == 0 {
		return nil, errors.New("no response")
	}
	return New(res[0])
}

//Bye says goodBye to n and returns true if success.
func (n *Node) Bye() bool {
	res, err := n.Talk("/bye/"+Me(true).Toxstring(), nil)
	if err != nil {
		log.Println("/bye", n.Nodestr, "error")
		return false
	}
	return len(res) > 0 && (res[0] == "BYEBYE")
}

//GetherNodes gethers nodes from n.
func (n *Node) GetherNodes() []*Node {
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
				NewN, err := nn.getNode()
				if err != nil {
					log.Println(err)
					return
				}
				mutex.Lock()
				ns[NewN.Nodestr] = NewN
				mutex.Unlock()
			}(nn)
		}
		wg.Wait()
		log.Println("iteration", i, ",# of nodes:", len(ns))
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

//NewSlice  makes node slice from names.
func NewSlice(names []string) Slice {
	var ns Slice
	for _, nn := range names {
		nn, err := New(nn)
		if err != nil {
			log.Println(err)
			continue
		}
		ns = append(ns, nn)
	}
	return ns
}

//Len returns size of nodes.
func (ns Slice) Len() int {
	return len(ns)
}

//Has returns true if ns has n.
func (ns Slice) Has(n *Node) bool {
	return util.HasString(ns.GetNodestrSlice(), n.Nodestr)
}

//GetNodestrSlice returns slice of Nodestr of nodes.
func (ns Slice) GetNodestrSlice() []string {
	s := make([]string, ns.Len())
	for i, v := range ns {
		s[i] = v.Nodestr
	}
	return s
}

//ToMap returns map[nodestr]struct{}{} for searching a node.
func (ns Slice) ToMap() map[string]struct{} {
	m := make(map[string]struct{})
	for _, nn := range ns {
		m[nn.Nodestr] = struct{}{}
	}
	return m
}

//Uniq solidate the slice.
func (ns Slice) Uniq() Slice {
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

//Extend make a New nodeslice including specified slices.
func (ns Slice) Extend(a Slice) Slice {
	nn := make(Slice, ns.Len()+a.Len())
	copy(nn, ns)
	copy(nn[ns.Len():], a)
	return nn.Uniq()
}

// Me converts myself to *Node.
func Me(servernameIfExist bool) *Node {
	ip, port := myself.GetIPPort()
	var serverName string
	if servernameIfExist {
		serverName = cfg.ServerName
	}
	if serverName == "" {
		serverName = ip
	}

	n, err := New(fmt.Sprintf("%s:%d%s", serverName, port, cfg.ServerURL))
	if err != nil {
		log.Fatal(err)
	}
	return n
}
