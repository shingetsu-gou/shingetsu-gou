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

package manager

import (
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/myself"
	"github.com/shingetsu-gou/shingetsu-gou/node"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

const (
	defaultNodes = 5 // Nodes keeping in node list
	shareNodes   = 5 // Nodes having the file
)

//Manager represents the map that maps datfile to it's source node list.
var isDirty bool
var nodes = make(map[string]node.Slice) //map[""] is nodelist
var mutex sync.RWMutex

//init read the file and returns NodeManager obj.
func init() {
	err := util.EachKeyValueLine(cfg.Lookup(), func(key string, value []string, i int) error {
		var nl node.Slice
		for _, v := range value {
			if v == "" {
				continue
			}
			nn, err := node.New(v)
			if err != nil {
				log.Println("line", i, "in lookup.txt,err=", err, v)
				continue
			}
			nl = append(nl, nn)
		}
		nodes[key] = nl
		return nil
	})
	if err != nil {
		log.Println(err)
	}
}

//getFromList returns number=n in the nodelist.
func getFromList(n int) *node.Node {
	mutex.RLock()
	defer mutex.RUnlock()
	if ListLen() == 0 {
		return nil
	}
	return nodes[""][n]
}

//NodeLen returns size of all nodes.
func NodeLen() int {
	ns := getAllNodes()
	return ns.Len()
}

//ListLen returns size of nodelist.
func ListLen() int {
	mutex.RLock()
	defer mutex.RUnlock()
	return len(nodes[""])
}

//GetNodestrSlice returns Nodestr of all nodes.
func GetNodestrSlice() []string {
	return getAllNodes().GetNodestrSlice()
}

//getAllNodes returns all nodes in table.
func getAllNodes() node.Slice {
	var n node.Slice
	mutex.RLock()
	defer mutex.RUnlock()
	for _, v := range nodes {
		n = append(n, v...)
	}
	return n.Uniq()
}

//GetNodestrSliceInTable returns Nodestr slice of nodes associated datfile thread.
func GetNodestrSliceInTable(datfile string) []string {
	mutex.RLock()
	defer mutex.RUnlock()
	n := nodes[datfile]
	return n.GetNodestrSlice()
}

//Random selects # of min(all # of nodes,n) nodes randomly except exclude nodes.
func Random(exclude node.Slice, num int) []*node.Node {
	all := getAllNodes()
	if exclude != nil {
		cand := make([]*node.Node, 0, len(all))
		m := exclude.ToMap()
		for _, n := range all {
			if _, exist := m[n.Nodestr]; !exist {
				cand = append(cand, n)
			}
		}
		all = cand
	}
	n := all.Len()
	if num < n && num != 0 {
		n = num
	}
	r := make([]*node.Node, n)
	rs := rand.Perm(all.Len())
	for i := 0; i < n; i++ {
		r[i] = all[rs[i]]
	}
	return r
}

//AppendToTable add node n to table if it is allowd and list doesn't have it.
func AppendToTable(datfile string, n *node.Node) {
	mutex.RLock()
	l := len(nodes[datfile])
	mutex.RUnlock()
	if ((datfile != "" && l < shareNodes) || (datfile == "" && l < defaultNodes)) &&
		n != nil && n.IsAllowed() && !hasNodeInTable(datfile, n) {
		mutex.Lock()
		isDirty = true
		nodes[datfile] = append(nodes[datfile], n)
		mutex.Unlock()
	}
}

//extendTable adds slice of nodes with check.
func extendToTable(datfile string, ns []*node.Node) {
	if ns == nil {
		return
	}
	for _, n := range ns {
		AppendToTable(datfile, n)
	}
}

//appendToList add node n to nodelist if it is allowd and list doesn't have it.
func appendToList(n *node.Node) {
	AppendToTable("", n)
}

//ReplaceNodeInList removes one node and say bye to the node and add n in nodelist.
//if len(node)>defaultnode
func ReplaceNodeInList(n *node.Node) *node.Node {
	mutex.RLock()
	l := len(nodes[""])
	mutex.RUnlock()
	if !n.IsAllowed() || hasNodeInTable("", n) {
		return nil
	}
	var old *node.Node
	if l >= defaultNodes {
		old = getFromList(0)
		RemoveFromList(old)
		old.Bye()
	}
	appendToList(n)
	return old
}

//extendToList adds node slice to nodelist.
func extendToList(ns []*node.Node) {
	extendToTable("", ns)
}

//hasNode returns true if nodelist in all tables has n.
func hasNode(n *node.Node) bool {
	return len(findNode(n)) > 0
}

//findNode returns datfile of node n, or -1 if not exist.
func findNode(n *node.Node) []string {
	mutex.RLock()
	defer mutex.RUnlock()
	var r []string
	for k := range nodes {
		if hasNodeInTable(k, n) {
			r = append(r, k)
		}
	}
	return r
}

//hasNodeInTable returns true if nodelist has n.
func hasNodeInTable(datfile string, n *node.Node) bool {
	return findNodeInTable(datfile, n) != -1
}

//findNode returns location of node n, or -1 if not exist.
func findNodeInTable(datfile string, n *node.Node) int {
	return util.FindString(GetNodestrSliceInTable(datfile), n.Nodestr)
}

//RemoveFromTable removes node n and return true if exists.
//or returns false if not exists.
func RemoveFromTable(datfile string, n *node.Node) bool {
	mutex.Lock()
	defer mutex.Unlock()
	i := 0
	if n != nil {
		i = util.FindString(nodes[datfile].GetNodestrSlice(), n.Nodestr)
	} else {
		for ii, nn := range nodes[datfile] {
			if nn == nil {
				i = ii
				break
			}
		}
	}
	if i >= 0 {
		ln := len(nodes[datfile])
		nodes[datfile], nodes[datfile][ln-1] = append(nodes[datfile][:i], nodes[datfile][i+1:]...), nil
		isDirty = true
		return true
	}
	return false
}

//RemoveFromList removes node n from nodelist and return true if exists.
//or returns false if not exists.
func RemoveFromList(n *node.Node) bool {
	return RemoveFromTable("", n)
}

//RemoveFromAllTable removes node n from all tables and return true if exists.
//or returns false if not exists.
func RemoveFromAllTable(n *node.Node) bool {
	del := false
	mutex.RLock()
	for k := range nodes {
		mutex.RUnlock()
		del = del || RemoveFromTable(k, n)
		mutex.RLock()
	}
	mutex.RUnlock()
	return del
}

//Initialize pings one of initNode except myself and added it if success,
//and get another node info from each nodes in nodelist.
func Initialize(allnodes node.Slice) {
	inodes := allnodes
	if len(allnodes) > defaultNodes {
		inodes = inodes[:defaultNodes]
	}
	var wg sync.WaitGroup
	pingOK := make([]*node.Node, 0, len(inodes))
	var mutex sync.Mutex
	for i := 0; i < len(inodes) && ListLen() < defaultNodes; i++ {
		wg.Add(1)
		go func(inode *node.Node) {
			defer wg.Done()
			if _, err := inode.Ping(); err == nil {
				mutex.Lock()
				pingOK = append(pingOK, inode)
				mutex.Unlock()
				Join(inode)
			}
		}(inodes[i])
	}
	wg.Wait()

	log.Println("# of nodelist:", ListLen())
	if ListLen() == 0 {
		myself.SetStatus(cfg.Port0)
		for _, p := range pingOK {
			appendToList(p)
		}
	}
}

//Join tells n to join and adds n to nodelist if welcomed.
//if n returns another nodes, repeats it and return true..
//removes fron nodelist if not welcomed and return false.
func Join(n *node.Node) bool {
	const retryJoin = 2 // Times; Join network
	if n == nil {
		return false
	}
	flag := false
	if hasNodeInTable("", n) || node.Me(false).Nodestr == n.Nodestr {
		return false
	}
	for count := 0; count < retryJoin && ListLen() < defaultNodes; count++ {
		extnode, err := n.Join()
		if err == nil && extnode == nil {
			appendToList(n)
			return true
		}
		if err == nil {
			appendToList(n)
			n = extnode
			flag = true
		} else {
			RemoveFromTable("", n)
			return flag
		}
	}
	return flag
}

//TellUpdate makes mynode info from node or dnsname or ip addr,
//and broadcast the updates of record id=id in cache c.datfile with stamp.
func TellUpdate(datfile string, stamp int64, id string, n *node.Node) {
	const updateNodes = 10

	tellstr := node.Me(true).Toxstring()
	if n != nil {
		tellstr = n.Toxstring()
	}
	msg := strings.Join([]string{"/update", datfile, strconv.FormatInt(stamp, 10), id, tellstr}, "/")

	ns := Get(datfile, nil)
	ns = ns.Extend(Get("", nil))
	ns = ns.Extend(Random(ns, updateNodes))
	log.Println("telling #", len(ns))
	for _, n := range ns {
		_, err := n.Talk(msg, nil)
		if err != nil {
			log.Println(err)
		}
	}
}

//Get returns rawnodelist associated with datfile
//if not found returns def
func Get(datfile string, def node.Slice) node.Slice {
	mutex.RLock()
	defer mutex.RUnlock()
	if v, exist := nodes[datfile]; exist {
		nodes := make([]*node.Node, v.Len())
		copy(nodes, v)
		return nodes
	}
	return def
}

//stringMap returns map of k=datfile, v=Nodestr of rawnodelist.
func stringMap() map[string][]string {
	mutex.RLock()
	defer mutex.RUnlock()
	result := make(map[string][]string)
	for k, v := range nodes {
		if k == "" {
			continue
		}
		result[k] = v.GetNodestrSlice()
	}
	return result
}

//Sync saves  k=datfile, v=Nodestr map to the file.
func Sync() {
	mutex.RLock()
	isDirtyB := isDirty
	mutex.RUnlock()
	if isDirtyB {
		m := stringMap()
		cfg.Fmutex.Lock()
		defer cfg.Fmutex.Unlock()
		err := util.WriteMap(cfg.Lookup(), m)
		if err != nil {
			log.Println(err)
		} else {
			mutex.Lock()
			isDirty = false
			mutex.Unlock()
		}
	}
}

//NodesForGet returns nodes which has datfile cache , and that extends nodes to #searchDepth .
func NodesForGet(datfile string, searchDepth int) node.Slice {
	var ns, ns2 node.Slice
	ns = ns.Extend(Get(datfile, nil))
	ns = ns.Extend(Get("", nil))
	ns = ns.Extend(Random(ns, 0))

	for _, n := range ns {
		if !n.Equals(node.Me(true)) && n.IsAllowed() {
			ns2 = append(ns2, n)
		}
	}
	if ns2.Len() > searchDepth {
		ns2 = ns2[:searchDepth]
	}
	return ns2
}
