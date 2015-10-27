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
	"strconv"
	"strings"
	"sync"
)

//NodeManager represents map datfile to it's source node list.
type NodeManager struct {
	isDirty bool
	nodes   map[string]nodeSlice //map[""] is nodelist
	mutex   sync.RWMutex
}

//newLookupTable read the file and returns LookupTable obj.
func newNodeManager() *NodeManager {
	r := &NodeManager{
		nodes: make(map[string]nodeSlice),
	}
	err := eachKeyValueLine(lookup, func(key string, value []string, i int) error {
		var nl nodeSlice
		for _, v := range value {
			nl = append(nl, newNode(v))
		}
		r.nodes[key] = nl
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	return r
}

//appendToList add node n to nodelist if it is allowd and list doesn't have it.
func (lt *NodeManager) getFromList(n int) *node {
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()
	if lt.listLen() == 0 {
		return nil
	}
	return lt.nodes[""][0]
}

//FileLen returns # of datfile.
func (lt *NodeManager) FileLen() int {
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()
	return len(lt.nodes) - 1
}

//nodeLen returns size of all nodes.
func (lt *NodeManager) nodeLen() int {
	ns := lt.getAllNodes()
	return ns.Len()
}

//listLen returns size of nodelist.
func (lt *NodeManager) listLen() int {
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()
	return len(lt.nodes[""])
}

//getNodestr returns nodestr of all nodes.
func (lt *NodeManager) getNodestrSlice() []string {
	return lt.getAllNodes().getNodestrSlice()
}

//getAllNodes returns all nodes in table.
func (lt *NodeManager) getAllNodes() nodeSlice {
	var n nodeSlice
	n = make([]*node, lt.nodeLen())
	i := 0
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()
	for _, v := range lt.nodes {
		for _, node := range v {
			n[i] = node
			i++
		}
	}
	return n.uniq()
}

//getNodestr returns nodestr of all nodes.
func (lt *NodeManager) getNodestrSliceInTable(datfile string) []string {
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()
	n := lt.nodes["datfile"]
	return n.getNodestrSlice()
}

//random selects #n node randomly except exclude nodes.
func (lt *NodeManager) random(exclude nodeSlice, num int) []*node {
	all := lt.getAllNodes()
	if exclude != nil {
		for i, n := range all {
			if exclude.has(n) {
				all, all[len(all)-1] = append(all[:i], all[i+1:]...), nil
			}
		}
	}
	r := make([]*node, num)
	rs := rand.Perm(all.Len() - 1)
	for i := 0; i < num; i++ {
		r[i] = all[rs[i]]
	}
	return r
}

//appendToTable add node n to table if it is allowd and list doesn't have it.
func (lt *NodeManager) appendToTable(datfile string, n *node) {
	lt.mutex.RLock()
	l := len(lt.nodes[datfile])
	lt.mutex.RUnlock()
	if ((datfile != "" && l < shareNodes) || (datfile == "" && l < defaultNodes)) &&
		n.isAllowed() && !lt.hasNodeInTable(datfile, n) {
		lt.mutex.Lock()
		lt.isDirty = true
		lt.nodes[datfile] = append(lt.nodes[datfile], n)
		lt.mutex.Unlock()
	}
}

//extendTable adds slice of nodes with check.
func (lt *NodeManager) extendToTable(datfile string, ns []*node) {
	if ns == nil {
		return
	}
	for _, n := range ns {
		lt.appendToTable(datfile, n)
	}
}

//appendToList add node n to nodelist if it is allowd and list doesn't have it.
func (lt *NodeManager) appendToList(n *node) {
	lt.appendToTable("", n)
}

//extendToList adds node slice to nodelist.
func (lt *NodeManager) extendToList(ns []*node) {
	lt.extendToTable("", ns)
}

//hasNode returns true if nodelist in all tables has n.
func (lt *NodeManager) hasNode(n *node) bool {
	return len(lt.findNode(n)) > 0
}

//findNode returns datfile of node n, or -1 if not exist.
func (lt *NodeManager) findNode(n *node) []string {
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()
	var r []string
	for k := range lt.nodes {
		if lt.hasNodeInTable(k, n) {
			r = append(r, k)
		}
	}
	return r
}

//hasNodeInTable returns true if nodelist has n.
func (lt *NodeManager) hasNodeInTable(datfile string, n *node) bool {
	return lt.findNodeInTable(datfile, n) != -1
}

//findNode returns location of node n, or -1 if not exist.
func (lt *NodeManager) findNodeInTable(datfile string, n *node) int {
	return findString(lt.getNodestrSliceInTable(datfile), n.nodestr)
}

//removeFromTable removes node n and return true if exists.
//or returns false if not exists.
func (lt *NodeManager) removeFromTable(datfile string, n *node) bool {
	lt.mutex.Lock()
	defer lt.mutex.Unlock()
	if i := findString(lt.nodes[datfile].getNodestrSlice(), n.nodestr); i >= 0 {
		lt.nodes[datfile] = append(lt.nodes[datfile][:i], lt.nodes[datfile][i+1:]...)
		lt.isDirty = true
		return true
	}
	return false
}

//removeFromList removes node n from nodelist and return true if exists.
//or returns false if not exists.
func (lt *NodeManager) removeFromList(n *node) bool {
	return lt.removeFromTable("", n)
}

//removeNode removes node n from all tables and return true if exists.
//or returns false if not exists.
func (lt *NodeManager) removeFromAllTable(n *node) bool {
	del := false
	lt.mutex.RLock()
	for k := range lt.nodes {
		defer lt.mutex.RUnlock()
		del = del || lt.removeFromTable(k, n)
	}
	return del
}

//moreNodes gets another node info from each nodes in nodelist.
func (lt *NodeManager) moreNodes() {
	no := 0
	count := 0
	all := lt.getAllNodes()
	for lt.nodeLen() < defaultNodes {
		nn := all[no]
		newN := nn.getNode()
		lt.join(newN)
		if count++; count > retry {
			count = 0
			if no++; no >= len(all) {
				return
			}
		}
	}
}

//initialize pings one of initNode except myself and added it if success,
//and get another node info from each nodes in nodelist.
func (lt *NodeManager) initialize() {
	if lt.listLen() > defaultNodes {
		return
	}
	for _, i := range initNode.data {
		inode := newNode(i)
		if _, err := inode.ping(); err == nil {
			lt.join(inode)
			break
		}
	}
	if myself != nil {
		lt.removeFromAllTable(myself)
	}
	if lt.nodeLen() > 0 {
		lt.moreNodes()
	}
	if lt.nodeLen() <= 1 {
		log.Println("few linked nodes")
	}
	lt.sync()
}

//join tells n to join and adds n to nodelist if welcomed.
//if n returns another nodes, repeats it and return true..
//removes fron nodelist if not welcomed and return false.
func (lt *NodeManager) join(n *node) bool {
	if n == nil {
		return false
	}
	flag := false
	if lt.hasNode(n) {
		return false
	}
	for count := 0; count < retryJoin && lt.nodeLen() < defaultNodes; count++ {
		welcome, extnode := n.join()
		if welcome && extnode == nil {
			lt.appendToList(n)
			return true
		}
		if welcome {
			lt.appendToList(n)
			n = extnode
			flag = true
		} else {
			lt.removeFromTable("", n)
			return flag
		}
	}
	return flag
}

//tellUpdate makes mynode info from node or dnsname or ip addr,
//and broadcast the updates of record id=id in cache c.datfile with stamp.
func (lt *NodeManager) tellUpdate(c *cache, stamp int64, id string, node *node) {
	var tellstr string
	switch {
	case node != nil:
		tellstr = node.toxstring()
	case dnsname != "":
		tellstr = myself.toxstring()
	default:
		tellstr = ":" + strconv.Itoa(ExternalPort) + strings.Replace(serverURL, "/", "+", -1)
	}
	msg := strings.Join([]string{"/update", c.Datfile, strconv.FormatInt(stamp, 10), id, tellstr}, "/")

	lt.mutex.Lock()
	ns := lt.nodes[c.Datfile].extend(lt.nodes[""])
	lt.mutex.Unlock()

	for _, n := range ns {
		_, err := n.talk(msg)
		if err != nil {
			log.Println(err)
		}
	}
}

//Get returns rawnodelist associated with datfile
//if not found return def
func (lt *NodeManager) get(datfile string, def []*node) []*node {
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()
	if v, exist := lt.nodes[datfile]; exist {
		nodes := make([]*node, v.Len())
		copy(nodes, v)
		return nodes
	}
	return def
}

//stringmap returns map of k=datfile, v=nodestr of rawnodelist.
func (lt *NodeManager) stringMap() map[string][]string {
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()
	result := make(map[string][]string)
	for k, v := range lt.nodes {
		if k == "" {
			continue
		}
		result[k] = v.getNodestrSlice()
	}
	return result
}

//sync saves  k=datfile, v=nodestr map to the file.
func (lt *NodeManager) sync() {
	if lt.isDirty {
		m := lt.stringMap()
		fmutex.Lock()
		defer fmutex.Unlock()
		err := writeMap(lookup, m)
		if err != nil {
			log.Println(err)
		} else {
			lt.mutex.Lock()
			lt.isDirty = false
			lt.mutex.Unlock()
		}
	}
}

//search checks one allowed nodes which selected randomly from nodes has the datfile record.
//if not found,n is removed from lookuptable. also if not pingable  removes n from searchlist and cache c.
//if found, n is added to lookuptable.
func (lt *NodeManager) search(c *cache, nodes []*node) *node {
	lt.mutex.RLock()
	ns := lt.nodes[c.Datfile].extend(nodes)
	lt.mutex.RUnlock()
	if ns.Len() < shareNodes {
		ns = ns.extend(lt.random(ns, shareNodes-ns.Len()))
	}
	count := 0
	for _, n := range ns {
		if n.equals(myself) || !n.isAllowed() {
			continue
		}
		res, err := n.talk("/have/" + c.Datfile)
		if err == nil && len(res) > 0 && res[0] == "YES" {
			lt.appendToTable(c.Datfile, n)
			lt.sync()
			return n
		}
		lt.removeFromTable(c.Datfile, n)
		if count++; count > searchDepth {
			break
		}
	}
	if count <= 1 {
		log.Println("Warning: Search nodes are null.")
	}
	return nil
}

//rejoin add nodes in searchlist if ping is ok and len(nodelist)<defaultNodes
//and doesn't have it's node.
//if ping is ng, removes node from searchlist.
func (lt *NodeManager) rejoin() {
	all := lt.getAllNodes()
	for _, n := range all {
		if lt.listLen() >= defaultNodes {
			return
		}
		lt.mutex.RLock()
		has := lt.nodes[""].has(n)
		lt.mutex.RUnlock()
		if has {
			continue
		}
		if _, err := n.ping(); err == nil || !lt.join(n) {
			lt.removeFromAllTable(n)
			lt.sync()
		} else {
			lt.appendToList(n)
		}
	}
	if lt.listLen() <= 1 {
		log.Println("Warning: Few linked nodes")
	}
}

//pingAll pings to all nodes in nodelist.
//if ng, removes from nodelist.
func (lt *NodeManager) pingAll() {
	lt.mutex.RLock()
	for _, n := range lt.nodes[""] {
		lt.mutex.RUnlock()
		if _, err := n.ping(); err != nil {
			lt.removeFromAllTable(n)
		}
		lt.mutex.RLock()
	}
	lt.mutex.RUnlock()
}

//rejoinlist joins all node in nodelist.
func (lt *NodeManager) rejoinList() {
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()
	for _, n := range lt.nodes[""] {
		n.join()
	}
}
