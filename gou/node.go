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

var initNode = newConfList(initnodeList, defaultInitNode)
var nodeAllow = newRegexpList(nodeAllowFile)
var nodeDeny = newRegexpList(nodeDenyFile)

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
		strings.Trim(line, "\r\n")
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
	nodestr = strings.Trim(nodestr, " \n\r")
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
	tiedlist []*node
}

func newRawNodeList(filepath string, caching bool) *rawNodeList {
	r := &rawNodeList{
		filepath: filepath,
		tiedlist: make([]*node, 0),
	}
	err := eachLine(filepath, func(line string, i int) error {
		n := newNode(line)
		r.tiedlist = append(r.tiedlist, n)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return r
}

func (t *rawNodeList) stringSlice() []string {
	r := make([]string, len(t.tiedlist))
	for i, v := range t.tiedlist {
		r[i] = v.nodestr
	}
	return r
}

func (t *rawNodeList) Len() int {
	return len(t.tiedlist)
}
func (t *rawNodeList) Swap(i, j int) {
	t.tiedlist[i], t.tiedlist[j] = t.tiedlist[j], t.tiedlist[i]
}

func (t *rawNodeList) Get(i int) string {
	return t.tiedlist[i].nodestr
}

func (t *rawNodeList) sync() {
	err := writeSlice(t.filepath, t)
	if err != nil {
		log.Println(err)
	}
}

func (t *rawNodeList) random() *node {
	return t.tiedlist[rand.Intn(len(t.tiedlist))]
}

func (t *rawNodeList) append(n *node) {
	if n.isAllowed() && !hasString(t, n.nodestr) {
		t.tiedlist = append(t.tiedlist, n)
	}
}

func (t *rawNodeList) extend(ns []*node) {
	for _, n := range ns {
		t.append(n)
	}
}

func (t *rawNodeList) has(n *node) bool {
	return t.find(n) != -1
}

func (t *rawNodeList) find(n *node) int {
	return findString(t, n.nodestr)
}

func (t *rawNodeList) remove(n *node) bool {
	if i := findString(t, n.nodestr); i >= 0 {
		t.tiedlist = append(t.tiedlist[:i], t.tiedlist[i:]...)
		return true
	}
	return false
}

type nodeList struct {
	*rawNodeList
}

func newNodeList() *nodeList {
	r := &rawNodeList{
		filepath: nodeFile,
		tiedlist: make([]*node, 0),
		//caching:true
	}
	nl := &nodeList{rawNodeList: r}
	return nl
}

func (nl *nodeList) initialize() {
	var inode *node
	for _, i := range initNode.data {
		inode = newNode(i)
		if _, ok := inode.ping(); ok {
			nl.join(inode)
			break
		}
	}
	my := nl.myself()
	if my != nil && nl.has(my) {
		nl.remove(my)
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
		nl.remove(inode)
	} else {
		if nl.Len() <= 1 {
			log.Println("few linked nodes")
		}
		for nl.Len() > defaultNodes {
			nn := nl.random()
			nn.bye()
			nl.remove(nn)
		}
	}
}

func (nl *nodeList) myself() *node {
	if dnsname == "" {
		return makeNode(dnsname, serverCgi, defaultPort)
	}
	for _, n := range nl.rawNodeList.tiedlist {
		if host, ok := n.ping(); ok {
			return makeNode(host, serverCgi, defaultPort)
		}
		log.Println("myself() failed at", n.nodestr)
	}
	log.Println("myself() failed")
	return nil
}

func (nl *nodeList) pingAll() {
	for _, n := range nl.rawNodeList.tiedlist {
		if _, ok := n.ping(); !ok {
			nl.remove(n)
		}
	}
}

func (nl *nodeList) join(n *node) bool {
	flag := false
	for count := 0; count < retryJoin && len(nl.tiedlist) < defaultNodes; count++ {
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
			nl.remove(n)
			return flag
		}
	}
	return flag
}

func (nl *nodeList) rejoin(searchlist *searchList) {
	doJoin := false
	for _, n := range searchlist.tiedlist {
		if len(nl.tiedlist) >= defaultNodes {
			break
		}
		if nl.has(n) {
			continue
		}
		doJoin = true
		if _, ok := n.ping(); !ok || !nl.join(n) {
			searchlist.remove(n)
		}
	}
	if doJoin {
		searchlist.tiedlist = append(searchlist.tiedlist, nl.tiedlist...)
		searchlist.sync()
		nl.sync()
	}
	if len(nl.tiedlist) <= 1 {
		log.Println("Warning: Few linked nodes")
	}
}

func (nl *nodeList) tellUpdate(c *cache, stamp int64, id string, node *node) {
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
	nlist := newNodeList()
	for _, n := range c.node.tiedlist {
		if _, ok := n.ping(); ok || nlist.find(n) != -1 {
			_, err := n.talk(msg)
			if err != nil {
				log.Println(err)
			}
		} else {
			c.node.remove(n)
			c.node.sync()
		}
	}
	for _, n := range nlist.tiedlist {
		if c.node.find(n) == -1 {
			_, err := n.talk(msg)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

type lookupTable struct {
	tosave   bool
	tieddict map[string]*rawNodeList
}

func newLookupTable() *lookupTable {
	r := &lookupTable{
		tosave:   false,
		tieddict: make(map[string]*rawNodeList),
	}
	err := eachKeyValueLine(lookup, func(key string, value []string, i int) error {
		nl := &rawNodeList{tiedlist: make([]*node, 0)}
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
func (lt *lookupTable) Len() int {
	return len(lt.tieddict)
}

func (lt *lookupTable) Get(i string, def []*node) []*node {
	if v, exist := lt.tieddict[i]; exist {
		return v.tiedlist
	}
	return def
}

func (lt *lookupTable) stringMap() map[string][]string {
	result := make(map[string][]string)
	for k, v := range lt.tieddict {
		result[k] = v.stringSlice()
	}
	return result
}

func (lt *lookupTable) sync(force bool) {
	if lt.tosave || force {
		err := writeMap(lookup, lt.stringMap())
		if err != nil {
			log.Println(err)
		}
	}
}

func (lt *lookupTable) add(datfile string, n *node) {
	if ns, exist := lt.tieddict[datfile]; exist {
		ns.append(n)
		lt.tosave = true
	}
}

func (lt *lookupTable) remove(datfile string, n *node) {
	if ns, exist := lt.tieddict[datfile]; exist {
		ns.remove(n)
		lt.tosave = true
	}
}

func (lt *lookupTable) clear() {
	lt.tieddict = make(map[string]*rawNodeList)
}

type searchList struct {
	*rawNodeList
}

func newSearchList() *searchList {
	r := newRawNodeList(searchFile, true)
	return &searchList{rawNodeList: r}
}

func (sl *searchList) join(n *node) {
	if !sl.has(n) {
		sl.append(n)
	}
}

func (sl *searchList) search(c *cache, myself *node, nodes []*node) *node {
	nl := &rawNodeList{tiedlist: make([]*node, 0)}
	if nodes != nil {
		nl.extend(nodes)
	}
	shuffle(nl)
	count := 0
	for _, n := range nl.tiedlist {
		if (myself != nil && n.nodestr == myself.nodestr) || n.isAllowed() {
			continue
		}
		count++
		tbl := newLookupTable()
		res, err := n.talk("/have" + c.datfile)
		if err == nil && len(res) > 0 && res[0] == "YES" {
			sl.sync()
			tbl.add(c.datfile, n)
			tbl.sync(false)
			return n
		}
		if _, ok := n.ping(); !ok {
			sl.remove(n)
			c.node.remove(n)
		}
		if rl, exist := tbl.tieddict[c.datfile]; exist {
			rl.remove(n)
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
