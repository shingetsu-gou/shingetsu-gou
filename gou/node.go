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
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)


func urlopen(url string, timeout time.Duration) ([]string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", version)

	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	var lines []string
	err = eachIOLine(resp.Body, func(line string, i int) error {
		strings.TrimSpace(line)
		lines = append(lines, line)
		return nil
	})
	return lines, err
}

type node struct {
	nodestr string
}

func newNode(nodestr string) *node {
	n := &node{}
	if nodestr == "" {
		log.Fatal("nodestr must not empty")
	}
	nodestr = strings.TrimSpace(nodestr)
	if match, err := regexp.MatchString("\\d+/[^: ]+$", nodestr); !match || err != nil {
		log.Fatal("bad format", err)
	}
	n.nodestr = strings.Replace(nodestr, "+", "/", -1)
	return n
}

func makeNode(host, path string, port int) *node {
	n := &node{}
	n.nodestr = host + ":" + strconv.Itoa(port) + strings.Replace(path, "+", "/", -1)
	return n
}

func (n *node) toxstring() string {
	return strings.Replace(n.nodestr, "/", "+", -1)
}

func (n *node) talk(message string) ([]string, error) {
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
		return nil, err
	}
	return res, nil
}

func (n *node) ping() (string, bool) {
	res, err := n.talk("/ping")
	if err == nil && res[0] == "PONG" && len(res) == 2 {
		return res[1], true
	}
	log.Println("/ping", n.nodestr, "error")
	return "", false
}

func (n *node) isAllowed() bool {
	if !nodeAllow.check(n.nodestr) && nodeDeny.check(n.nodestr) {
		return false
	}
	return true
}

func (n *node) join() (bool, *node) {
	if n.isAllowed() {
		return false, nil
	}
	path := strings.Replace(serverCgi, "/", "+", -1)
	port := strconv.Itoa(defaultPort)
	res, err := n.talk("/join/" + dnsname + ":" + port + path)
	if err != nil {
		return false, nil
	}
	return (res[0] == "WELCOME"), newNode(res[1])
}

func (n *node) getNode() *node {
	res, err := n.talk("/node")
	if err != nil {
		log.Println("/node", n.nodestr, "error")
		return nil
	}
	return newNode(res[0])
}

func (n *node) bye() bool {
	path := strings.Replace(serverCgi, "/", "+", -1)
	port := strconv.Itoa(defaultPort)
	res, err := n.talk("/bye/" + dnsname + ":" + port + path)
	if err != nil {
		log.Println("/bye", n.nodestr, "error")
		return false
	}
	return (res[0] == "BYEBYE")
}

type rawNodeList struct {
	filepath string
	nodes    []*node
}

func newRawNodeList(filepath string) *rawNodeList {
	r := &rawNodeList{filepath: filepath}

	err := eachLine(filepath, func(line string, i int) error {
		n := newNode(line)
		r.nodes = append(r.nodes, n)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return r
}

func (t *rawNodeList) Len() int {
	return len(t.nodes)
}
func (t *rawNodeList) Swap(i, j int) {
	t.nodes[i], t.nodes[j] = t.nodes[j], t.nodes[i]
}

func (t *rawNodeList) getNodestrSlice() []string {
	result := make([]string, len(t.nodes))
	for i, v := range t.nodes {
		result[i] = v.nodestr
	}
	return result
}

func (t *rawNodeList) sync() {
	err := writeSlice(t.filepath, t.getNodestrSlice())
	if err != nil {
		log.Println(err)
	}
}

func (t *rawNodeList) random() *node {
	return t.nodes[rand.Intn(len(t.nodes))]
}

func (t *rawNodeList) append(n *node) {
	if n.isAllowed() && !hasString(t.getNodestrSlice(), n.nodestr) {
		t.nodes = append(t.nodes, n)
	}
}

func (t *rawNodeList) extend(ns []*node) {
	for _, n := range ns {
		t.append(n)
	}
}

func (t *rawNodeList) hasNode(n *node) bool {
	return t.findNode(n) != -1
}

func (t *rawNodeList) findNode(n *node) int {
	return findString(t.getNodestrSlice(), n.nodestr)
}

func (t *rawNodeList) removeNode(n *node) bool {
	if i := findString(t.getNodestrSlice(), n.nodestr); i >= 0 {
		t.nodes = append(t.nodes[:i], t.nodes[i:]...)
		return true
	}
	return false
}

type NodeList struct {
	*rawNodeList
}

func newNodeList() *NodeList {
	r := newRawNodeList(nodeFile)
	nl := &NodeList{rawNodeList: r}
	return nl
}

func (nl *NodeList) initialize() {
	var inode *node
	for _, i := range initNode.data {
		inode = newNode(i)
		if _, ok := inode.ping(); ok {
			nl.join(inode)
			break
		}
	}
	my := nl.myself()
	if my != nil && nl.hasNode(my) {
		nl.removeNode(my)
	}
	if nl.Len() == 0 {
		return
	}
	done := make(map[string]int)

	for {
		if nl.Len() == 0 {
			break
		}
		nn := nl.random()
		newN := nn.getNode()
		if _, exist := done[newN.nodestr]; newN != nil && !exist {
			nl.join(newN)
			done[newN.nodestr] = 1
		}
		done[nn.nodestr]++
		if done[nn.nodestr] > retry && nl.Len() >= defaultNodes {
			break
		}
	}
	if nl.Len() > defaultNodes {
		inode.bye()
		nl.removeNode(inode)
	} else {
		if nl.Len() <= 1 {
			log.Println("few linked nodes")
		}
		for nl.Len() > defaultNodes {
			nn := nl.random()
			nn.bye()
			nl.removeNode(nn)
		}
	}
}

func (nl *NodeList) myself() *node {
	if dnsname == "" {
		return makeNode(dnsname, serverCgi, defaultPort)
	}
	for _, n := range nl.rawNodeList.nodes {
		if host, ok := n.ping(); ok {
			return makeNode(host, serverCgi, defaultPort)
		}
		log.Println("myself() failed at", n.nodestr)
	}
	log.Println("myself() failed")
	return nil
}

func (nl *NodeList) pingAll() {
	for _, n := range nl.rawNodeList.nodes {
		if _, ok := n.ping(); !ok {
			nl.removeNode(n)
		}
	}
}

func (nl *NodeList) join(n *node) bool {
	flag := false
	for count := 0; count < retryJoin && len(nl.nodes) < defaultNodes; count++ {
		welcome, extnode := n.join()
		if welcome && extnode == nil {
			nl.append(n)
			return true
		}
		if welcome {
			nl.append(n)
			n = extnode
			flag = true
		} else {
			nl.removeNode(n)
			return flag
		}
	}
	return flag
}

func (nl *NodeList) rejoin(searchlist *SearchList) {
	for _, n := range searchlist.nodes {
		if len(nl.nodes) >= defaultNodes {
			return
		}
		if nl.hasNode(n) {
			continue
		}
		if _, ok := n.ping(); !ok || !nl.join(n) {
			searchlist.removeNode(n)
			searchlist.sync()
		} else {
			nl.append(n)
			nl.sync()
		}
	}
	if len(nl.nodes) <= 1 {
		log.Println("Warning: Few linked nodes")
	}
}

func (nl *NodeList) tellUpdate(c *cache, stamp int64, id string, node *node) {
	var tellstr string
	switch {
	case node != nil:
		tellstr = node.toxstring()
	case dnsname != "":
		tellstr = nl.myself().toxstring()
	default:
		tellstr = ":" + strconv.Itoa(defaultPort) + strings.Replace(serverCgi, "/", "+", -1)
	}
	arg := strings.Join([]string{"/update/", c.datfile, strconv.FormatInt(stamp, 10), id, tellstr}, "/")
	go broadcast(arg, c)
}

func broadcast(msg string, c *cache) {
	for _, n := range c.node.nodes {
		if _, ok := n.ping(); ok || nodeList.findNode(n) != -1 {
			_, err := n.talk(msg)
			if err != nil {
				log.Println(err)
			}
		} else {
			c.node.removeNode(n)
			c.node.sync()
		}
	}
	for _, n := range nodeList.nodes {
		if c.node.findNode(n) == -1 {
			_, err := n.talk(msg)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

type LookupTable struct {
	tosave   bool
	tieddict map[string]*rawNodeList
}

func newLookupTable() *LookupTable {
	r := &LookupTable{
		tosave:   false,
		tieddict: make(map[string]*rawNodeList),
	}
	err := eachKeyValueLine(lookup, func(key string, value []string, i int) error {
		nl := &rawNodeList{nodes: make([]*node, 0)}
		for _, v := range value {
			nl.append(newNode(v))
		}
		r.tieddict[key] = nl
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return r
}
func (lt *LookupTable) Len() int {
	return len(lt.tieddict)
}

func (lt *LookupTable) Get(i string, def []*node) []*node {
	if v, exist := lt.tieddict[i]; exist {
		return v.nodes
	}
	return def
}

func (lt *LookupTable) stringMap() map[string][]string {
	result := make(map[string][]string)
	for k, v := range lt.tieddict {
		result[k] = v.getNodestrSlice()
	}
	return result
}

func (lt *LookupTable) sync(force bool) {
	if lt.tosave || force {
		err := writeMap(lookup, lt.stringMap())
		if err != nil {
			log.Println(err)
		}
	}
}

func (lt *LookupTable) add(datfile string, n *node) {
	if ns, exist := lt.tieddict[datfile]; exist {
		ns.append(n)
		lt.tosave = true
	}
}

func (lt *LookupTable) remove(datfile string, n *node) {
	if ns, exist := lt.tieddict[datfile]; exist {
		ns.removeNode(n)
		lt.tosave = true
	}
}

func (lt *LookupTable) clear() {
	lt.tieddict = make(map[string]*rawNodeList)
}

type SearchList struct {
	*rawNodeList
}

func newSearchList() *SearchList {
	r := newRawNodeList(searchFile)
	return &SearchList{rawNodeList: r}
}

func (sl *SearchList) join(n *node) {
	if !sl.hasNode(n) {
		sl.append(n)
	}
}

func (sl *SearchList) search(c *cache, myself *node, nodes []*node) *node {
	nl := &rawNodeList{nodes: make([]*node, 0)}
	if nodes != nil {
		nl.extend(nodes)
	}
	shuffle(nl)
	count := 0
	for _, n := range nl.nodes {
		if (myself != nil && n.nodestr == myself.nodestr) || n.isAllowed() {
			continue
		}
		count++
		res, err := n.talk("/have" + c.datfile)
		if err == nil && len(res) > 0 && res[0] == "YES" {
			sl.sync()
			lookupTable.add(c.datfile, n)
			lookupTable.sync(false)
			return n
		}
		if _, ok := n.ping(); !ok {
			sl.removeNode(n)
			c.node.removeNode(n)
		}
		if rl, exist := lookupTable.tieddict[c.datfile]; exist {
			rl.removeNode(n)
		}
		if count > searchDepth {
			break
		}
	}
	sl.sync()
	if count <= 1 {
		log.Println("Warning: Search nodes are null.")
	}
	return nil
}
