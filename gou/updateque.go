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
	"strconv"
	"strings"
	"sync"
)

type updateQue struct {
	queue   map[string][]*node
	running bool
	mutex   sync.Mutex
}

func newUpdateQue() *updateQue {
	u := &updateQue{
		queue: make(map[string][]*node),
	}
	return u
}

func (u *updateQue) append(datfile string, stamp int64, id string, n *node) {
	key := strings.Join([]string{strconv.FormatInt(stamp, 10), id, datfile}, "<>")
	if u.queue[key] == nil {
		u.queue[key] = make([]*node, 0)
	}
	u.queue[key] = append(u.queue[key], n)
}

func (u *updateQue) run() {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	u.running = true
	for updateid := range u.queue {
		u.doUpdate(updateid)
	}
	u.running = false
}

func (u *updateQue) doUpdateNode(rec *record, n *node) bool {
	ul := newUpdateList("", -1)
	if ul.has(rec) {
		return true
	}
	ca := newCache(rec.datfile, nil, nil)
	nl := newNodeList()
	sl := newSearchList()
	var flagGot, flagSpam bool
	switch {
	case !ca.exists():
		if sl.Len() < searchDepth {
			nl.tellUpdate(ca, rec.stamp, rec.id, n)
		}
		return true
	case n == nil:
	case ca.Len() > 0:
		flagGot, flagSpam = ca.getData(rec.stamp, rec.id, n)
	default:
		ca.getWithRange(n)
		flagGot = rec.exists()
		flagSpam = false
	}
	if n == nil {
		nl.tellUpdate(ca, rec.stamp, rec.id, n)
		return true
	}
	if flagGot {
		if !flagSpam {
			nl.tellUpdate(ca, rec.stamp, rec.id, nil)
		}
		if !nl.has(n) && nl.Len() < defaultNodes {
			nl.join(n)
			nl.sync()
		}
		sl = newSearchList()
		if !sl.has(n) {
			sl.join(n)
			sl.sync()
		}
		return true
	}
	return false
}

func (u *updateQue) doUpdate(updateid string) {
	if _, exist := u.queue[updateid]; !exist {
		return
	}
	ids := strings.Split(updateid, "<>")
	if len(ids) < 3 {
		log.Println("illegal format")
		return
	}
	rec := newRecord(ids[2], ids[0]+"_"+ids[1])
	for i, n := range u.queue[updateid] {
		if u.doUpdateNode(rec, n) {
			delete(u.queue, updateid)
			ul := newUpdateList("", -1)
			ul.append(rec)
			ul.sync()
			rl := newRecentList()
			rl.append(rec)
			rl.sync()
			return
		}
		u.queue[updateid] = append(u.queue[updateid][:i], u.queue[updateid][i:]...)

	}
}
