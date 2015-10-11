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

import "sync"

//updateQue contains update records which will be informed to other nodes
type updateQue struct {
	queue map[*record][]*node
	//	running bool
	mutex sync.Mutex
}

//newUpdateQue make updateQue object.
func newUpdateQue() *updateQue {
	u := &updateQue{
		queue: make(map[*record][]*node),
	}
	return u
}

//append adds a record and origina n to be broadcasted.
func (u *updateQue) append(rec *record, n *node) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	u.queue[rec] = append(u.queue[rec], n)
}

//run do doUpdateNode for each records using related nodes.
//if success to doUpdateNode, add node to updatelist and recentlist and
//removes the record from queue.
func (u *updateQue) run() {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	for rec, ns := range u.queue {
		for i, n := range ns {
			if u.doUpdateNode(rec, n) {
				delete(u.queue, rec)
				updateList.append(rec)
				updateList.sync()
				recentList.append(rec)
				recentList.sync()
				break
			}
			u.queue[rec] = append(u.queue[rec][:i], u.queue[rec][i:]...)
		}
	}
}

//doUpdateNode broadcast and get data for each new records.
//if can get data (even if spam) return true, if fails to get, return false.
//if no fail, broadcast updates to node in cache and added n to nodelist and searchlist.
func (u *updateQue) doUpdateNode(rec *record, n *node) bool {
	if updateList.hasInfo(rec) {
		return true
	}
	ca := newCache(rec.datfile)
	var err error
	switch {
	case !ca.Exists(), n == nil: //no cache, only broadcast updates.
		nodeList.tellUpdate(ca, rec.Stamp, rec.ID, n)
		return true
	case ca.Len() > 0: //cache and records exists, get data from node n.
		err = ca.getData(rec.Stamp, rec.ID, n)
	default: //cache exists ,but no records. get data with range.
		ca.getWithRange(n)
		if flagGot := rec.Exists(); !flagGot {
			err = errGet
		}
	}
	switch err {
	case errGet:
		return false
	case errSpam:
		return true
	default:
		nodeList.tellUpdate(ca, rec.Stamp, rec.ID, nil)
		nodeList.join(n)
		nodeList.sync()
		searchList.join(n)
		searchList.sync()
		return true
	}
}
