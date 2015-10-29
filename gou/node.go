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
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	myself       *node
	externalPort int
)

//urlopen retrievs html data from url
func urlopen(url string, timeout time.Duration) ([]string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", getVersion())

	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	var lines []string
	err = eachIOLine(resp.Body, func(line string, i int) error {
		strings.TrimRight(line, "\r\n")
		lines = append(lines, line)
		return nil
	})
	return lines, err
}

type nodeConfig struct {
	serverName string
	nodeAllow  *regexpList
	nodeDeny   *regexpList
}

func newNodeConfig(cfg *Config) *nodeConfig {
	return &nodeConfig{
		serverName: cfg.ServerName,
		nodeAllow:  newRegexpList(cfg.NodeAllowFile),
		nodeDeny:   newRegexpList(cfg.NodeDenyFile),
	}
}

//node represents node info.
type node struct {
	*nodeConfig
	nodestr string
}

//newNode checks nodestr format and returns node obj.
func newNode(nodestr string) *node {
	nodestr = strings.TrimSpace(nodestr)
	if nodestr == "" {
		log.Printf("nodestr is empty")
		return nil
	}
	if match, err := regexp.MatchString(`\d+/[^: ]+$`, nodestr); !match || err != nil {
		log.Println("bad format", err)
		return nil
	}
	n := &node{}
	n.nodestr = strings.Replace(nodestr, "+", "/", -1)
	return n
}

//equals return true is nodestr is equal.
func (n *node) equals(nn *node) bool {
	if nn == nil {
		return false
	}
	return n.nodestr == nn.nodestr
}

//makeNode makes node from host info.
func makeNode(host, path string, port int) *node {
	n := &node{}
	n.nodestr = net.JoinHostPort(host, strconv.Itoa(port)) + strings.Replace(path, "+", "/", -1)
	return n
}

//toxstring covnerts nodestr to saku node format.
func (n *node) toxstring() string {
	return strings.Replace(n.nodestr, "/", "+", -1)
}

//talk talks with n with the message and returns data.
func (n *node) talk(message string) ([]string, error) {
	const defaultTimeout = 20 * time.Second // Seconds; Timeout for TCP

	const getTimeout = 2 * time.Minute // Seconds; Timeout for /get

	if !strings.HasPrefix(message, "/") {
		message = "/" + message
	}
	var timeout time.Duration
	if !strings.HasPrefix(message, "/get") {
		timeout = getTimeout
	} else {
		timeout = defaultTimeout
	}

	message = "http://" + n.nodestr + message
	log.Println("talk:", message)
	res, err := urlopen(message, timeout)
	if err != nil {
		log.Println(message, err)
	}
	return res, err
}

//ping pings to n and return response.
func (n *node) ping() (string, error) {
	res, err := n.talk("/ping")
	if err != nil {
		log.Println("/ping", n.nodestr, err)
		return "", err
	}
	if res[0] == "PONG" && len(res) == 2 {
		log.Println("ponged,i am", res[1])
		if n.serverName != "" {
			myself = makeNode(n.serverName, ServerURL, externalPort)
		} else {
			myself = newNode(res[1])
		}
		return res[1], nil
	}
	log.Println("/ping", n.nodestr, "error")
	return "", errors.New("connected,but not ponged")
}

//isAllow returns fase if n is not allowed and denied.
func (n *node) isAllowed() bool {
	if !n.nodeAllow.check(n.nodestr) && n.nodeDeny.check(n.nodestr) {
		return false
	}
	return true
}

//join requests n to join me and return true and other node name if success.
func (n *node) join() (bool, *node) {
	if !n.isAllowed() {
		log.Println(n.nodestr, "is not allowd")
		return false, nil
	}
	path := strings.Replace(ServerURL, "/", "+", -1)
	port := strconv.Itoa(externalPort)
	res, err := n.talk("/join/" + n.serverName + ":" + port + path)
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
	return (res[0] == "WELCOME"), newNode(res[1])
}

//getNode request n to pass me another node info and returns another node.
func (n *node) getNode() *node {
	res, err := n.talk("/node")
	if err != nil {
		log.Println("/node", n.nodestr, "error")
		return nil
	}
	return newNode(res[0])
}

//bye says goodbye to n and returns true if success.
func (n *node) bye() bool {
	path := strings.Replace(ServerURL, "/", "+", -1)
	port := strconv.Itoa(externalPort)
	res, err := n.talk("/bye/" + n.serverName + ":" + port + path)
	if err != nil {
		log.Println("/bye", n.nodestr, "error")
		return false
	}
	return (res[0] == "BYEBYE")
}

type nodeSlice []*node

//Len returns size of nodes.
func (ns nodeSlice) Len() int {
	return len(ns)
}

//Swap swaps nodes order.
func (ns nodeSlice) Swap(i, j int) {
	ns[i], ns[j] = ns[j], ns[i]
}

//getNodestrSlice returns slice of nodestr of nodes.
func (ns nodeSlice) getNodestrSlice() []string {
	s := make([]string, ns.Len())
	for i, v := range ns {
		s[i] = v.nodestr
	}
	return s
}

//has returns true if nodeslice has node n.
func (ns nodeSlice) has(n *node) bool {
	for _, nn := range ns {
		if n.nodestr == nn.nodestr {
			return true
		}
	}
	return false
}

//uniq solidate the slice.
func (ns nodeSlice) uniq() nodeSlice {
	for i, n := range ns {
		for j, nn := range ns[i+1:] {
			if n.equals(nn) {
				ns, ns[len(ns)-1] = append(ns[:j], ns[j+1:]...), nil
			}
		}
	}
	return ns
}

//extend make a new nodeslice including specified slices.
func (ns nodeSlice) extend(a nodeSlice) nodeSlice {
	nn := make([]*node, ns.Len()+a.Len())
	copy(nn, ns)
	copy(nn[ns.Len():], a)
	return nn
}
