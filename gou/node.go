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
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//urlopen retrievs html data from url
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
	if err != nil {
		log.Println(err)
		return nil, err
	}
	var lines []string
	err = eachIOLine(resp.Body, func(line string, i int) error {
		strings.TrimSpace(line)
		lines = append(lines, line)
		return nil
	})
	return lines, err
}

//node represents node info.
type node struct {
	nodestr string
}

//newNode checks nodestr format and returns node obj.
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
	n.nodestr = host + ":" + strconv.Itoa(port) + strings.Replace(path, "+", "/", -1)
	return n
}

//toxstring covnerts nodestr to saku node format.
func (n *node) toxstring() string {
	return strings.Replace(n.nodestr, "/", "+", -1)
}

//talk talks with n with the message and returns data.
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
		return res[1], nil
	}
	log.Println("/ping", n.nodestr, "error")
	return "", errors.New("connected,but not ponged")
}

//isAllow returns fase if n is not allowed and denied.
func (n *node) isAllowed() bool {
	if !nodeAllow.check(n.nodestr) && nodeDeny.check(n.nodestr) {
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
	path := strings.Replace(serverURL, "/", "+", -1)
	port := strconv.Itoa(ExternalPort)
	res, err := n.talk("/join/" + dnsname + ":" + port + path)
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
	path := strings.Replace(serverURL, "/", "+", -1)
	port := strconv.Itoa(ExternalPort)
	res, err := n.talk("/bye/" + dnsname + ":" + port + path)
	if err != nil {
		log.Println("/bye", n.nodestr, "error")
		return false
	}
	return (res[0] == "BYEBYE")
}

//rawNodeList is base class representing list of nodes.
type rawNodeList struct {
	filepath string
	nodes    []*node
}

//newRawNodeList read the file and returns rawNodeList obj .
func newRawNodeList(filepath string) *rawNodeList {
	r := &rawNodeList{filepath: filepath}
	if !IsDir(filepath) {
		return r
	}
	err := eachLine(filepath, func(line string, i int) error {
		n := newNode(line)
		r.nodes = append(r.nodes, n)
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	return r
}

//Len returns size of nodes.
func (t *rawNodeList) Len() int {
	return len(t.nodes)
}

//Swap swaps nodes order.
func (t *rawNodeList) Swap(i, j int) {
	t.nodes[i], t.nodes[j] = t.nodes[j], t.nodes[i]
}

//getNodestrSlice returns nodestr slice of nodes.
func (t *rawNodeList) getNodestrSlice() []string {
	result := make([]string, len(t.nodes))
	for i, v := range t.nodes {
		result[i] = v.nodestr
	}
	return result
}

//sync saves nodestr to filepath.
func (t *rawNodeList) sync() {
	err := writeSlice(t.filepath, t.getNodestrSlice())
	if err != nil {
		log.Println(err)
	}
}

//random select one node randomly.
func (t *rawNodeList) random() *node {
	return t.nodes[rand.Intn(len(t.nodes))]
}

//append add node n if it is allowd and list doesn't have it.
func (t *rawNodeList) append(n *node) {
	if n.isAllowed() && !hasString(t.getNodestrSlice(), n.nodestr) {
		t.nodes = append(t.nodes, n)
	}
}

//extend adds slice of nodes with check.
func (t *rawNodeList) extend(ns []*node) {
	for _, n := range ns {
		t.append(n)
	}
}

//hasNode returns true if nodelist has n.
func (t *rawNodeList) hasNode(n *node) bool {
	return t.findNode(n) != -1
}

//findNode returns location of node n, or -1 if not exist.
func (t *rawNodeList) findNode(n *node) int {
	return findString(t.getNodestrSlice(), n.nodestr)
}

//removeNode removes node n and return true if exists.
//or returns false if not exists.
func (t *rawNodeList) removeNode(n *node) bool {
	if i := findString(t.getNodestrSlice(), n.nodestr); i >= 0 {
		t.nodes = append(t.nodes[:i], t.nodes[i+1:]...)
		return true
	}
	return false
}

//NodeList represents adjacent node list.
type NodeList struct {
	*rawNodeList
}

//newNodeList reads the file and returns NodeList obj
func newNodeList() *NodeList {
	r := newRawNodeList(nodeFile)
	nl := &NodeList{rawNodeList: r}
	return nl
}

//moreNodes gets another node info from each nodes in nodelist.
func (nl *NodeList) moreNodes() {
	my := nl.myself()
	if my != nil && nl.hasNode(my) {
		nl.removeNode(my)
	}
	done := make(map[string]int)
	for nl.Len() != 0 && nl.Len() < defaultNodes {
		nn := nl.random()
		newN := nn.getNode()
		if newN != nil {
			if _, exist := done[newN.nodestr]; !exist {
				nl.join(newN)
				done[newN.nodestr] = 1
			}
		}
		done[nn.nodestr]++
		if done[nn.nodestr] > retry {
			break
		}
	}
}

//initialize pings one of initNode except myself and added it if success,
//and get another node info from each nodes in nodelist.
//if can get sufficent nodes, removes initNode.
//after that if over sufficient nodes, removes random nodes from nodelist.
func (nl *NodeList) initialize() {
	var inode *node
	for _, i := range initNode.data {
		inode = newNode(i)
		if _, err := inode.ping(); err == nil {
			nl.join(inode)
			break
		}
	}
	nl.moreNodes()
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

//myself makes mynode info from dnsname.
//if dnsname is empty ping to a node in nodelist and get info of myself.
func (nl *NodeList) myself() *node {
	if dnsname != "" {
		return makeNode(dnsname, serverURL, ExternalPort)
	}
	for _, n := range nl.rawNodeList.nodes {
		if host, err := n.ping(); err == nil {
			return makeNode(host, serverURL, ExternalPort)
		}
		log.Println("myself() failed at", n.nodestr)
	}
	log.Println("myself() failed")
	return nil
}

//pingAll pings to all nodes in nodelist.
//if ng, removes from nodelist.
func (nl *NodeList) pingAll() {
	for _, n := range nl.rawNodeList.nodes {
		if _, err := n.ping(); err == nil {
			nl.removeNode(n)
		}
	}
}

//join tells n to join and adds n to nodelist if welcomed.
//if n returns another nodes, repleats it and return true..
//removes fron nodelist if not welcomed and return false.
func (nl *NodeList) join(n *node) bool {
	flag := false
	if nl.hasNode(n) {
		return false
	}
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

//rejoin add nodes in searchlist if ping is ok and len(nodelist)<defaultNodes
//and doesn't have it's node.
//if ping is ng, removes node from searchlist.
func (nl *NodeList) rejoin(searchlist *SearchList) {
	for _, n := range searchlist.nodes {
		if len(nl.nodes) >= defaultNodes {
			return
		}
		if nl.hasNode(n) {
			continue
		}
		if _, err := n.ping(); err == nil || !nl.join(n) {
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

//tellUpdate makes mynode info from node or dnsname or ip addr,
//and broadcast the updates of record id=id in cache c.datfile with stamp.
func (nl *NodeList) tellUpdate(c *cache, stamp int64, id string, node *node) {
	var tellstr string
	switch {
	case node != nil:
		tellstr = node.toxstring()
	case dnsname != "":
		tellstr = nl.myself().toxstring()
	default:
		tellstr = ":" + strconv.Itoa(ExternalPort) + strings.Replace(serverURL, "/", "+", -1)
	}
	arg := strings.Join([]string{"/update/", c.Datfile, strconv.FormatInt(stamp, 10), id, tellstr}, "/")
	broadcast(arg, c)
}

//broadcast broadcsts msg to nodes which has info of cache c  if ping is ok or is in nodelist.
//and also broadcasts to nodes in nodelist.
//if ping is ng or nodelist has n , remove n from nodes in cache.
func broadcast(msg string, c *cache) {
	for _, n := range c.node.nodes {
		if _, err := n.ping(); err == nil || nodeList.hasNode(n) {
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
		if !c.node.hasNode(n) {
			_, err := n.talk(msg)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

//LookupTable represents map datfile to it's source node list.
type LookupTable struct {
	tosave      bool
	rawnodelist map[string]*rawNodeList
}

//newLookupTable read the file and returns LookupTable obj.
func newLookupTable() *LookupTable {
	r := &LookupTable{
		tosave:      false,
		rawnodelist: make(map[string]*rawNodeList),
	}
	err := eachKeyValueLine(lookup, func(key string, value []string, i int) error {
		nl := &rawNodeList{}
		for _, v := range value {
			nl.append(newNode(v))
		}
		r.rawnodelist[key] = nl
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	return r
}

//Len returns size of rawnodelist.
func (lt *LookupTable) Len() int {
	return len(lt.rawnodelist)
}

//Get returns rawnodelist associated with datfile
//if not found return def
func (lt *LookupTable) get(datfile string, def []*node) []*node {
	if v, exist := lt.rawnodelist[datfile]; exist {
		return v.nodes
	}
	return def
}

//stringmap returns map of k=datfile, v=nodestr of rawnodelist.
func (lt *LookupTable) stringMap() map[string][]string {
	result := make(map[string][]string)
	for k, v := range lt.rawnodelist {
		result[k] = v.getNodestrSlice()
	}
	return result
}

//sync saves  k=datfile, v=nodestr map to the file.
func (lt *LookupTable) sync(force bool) {
	if lt.tosave || force {
		err := writeMap(lookup, lt.stringMap())
		if err != nil {
			log.Println(err)
		}
	}
}

//add associates node n to datfile and stores it.
func (lt *LookupTable) add(datfile string, n *node) {
	if ns, exist := lt.rawnodelist[datfile]; exist {
		ns.append(n)
		lt.tosave = true
	}
}

//remove removes n from key=datfile rawnodelist.
func (lt *LookupTable) remove(datfile string, n *node) {
	if ns, exist := lt.rawnodelist[datfile]; exist {
		ns.removeNode(n)
		lt.tosave = true
	}
}

//clear removes rawnodelist.
func (lt *LookupTable) clear() {
	lt.rawnodelist = make(map[string]*rawNodeList)
}

//SearchList represents nodes list for searching.
type SearchList struct {
	*rawNodeList
}

//newSearchList read the file and returns SearchList obj.
func newSearchList() *SearchList {
	r := newRawNodeList(searchFile)
	return &SearchList{rawNodeList: r}
}

//join adds node n if list doesn't have it.
func (sl *SearchList) join(n *node) {
	if !sl.hasNode(n) {
		sl.append(n)
	}
}

//search checks one allowed nodes which selected randomly from nodes has the datfile record.
//if not found,n is removed from lookuptable. also if not pingable  removes n from searchlist and cache c.
//if found, n is added to lookuptable.
func (sl *SearchList) search(c *cache, myself *node, nodes []*node) *node {
	nl := &rawNodeList{}
	if nodes != nil {
		nl.extend(nodes)
	}
	nl.extend(sl.nodes)
	shuffle(nl)
	count := 0
	for _, n := range nl.nodes {
		if n.equals(myself) || !n.isAllowed() {
			continue
		}
		count++
		res, err := n.talk("/have/" + c.Datfile)
		if err == nil && res[0] == "YES" {
			sl.sync()
			lookupTable.add(c.Datfile, n)
			lookupTable.sync(false)
			return n
		}
		if _, err := n.ping(); err != nil {
			sl.removeNode(n)
			c.node.removeNode(n)
		}
		lookupTable.remove(c.Datfile, n)
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
