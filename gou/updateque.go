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
	"sync"
	"time"
)

type updateQue struct {
	mutex   sync.Mutex
	updated map[[16]byte]time.Time
}

func newUpdateQue() *updateQue {
	return &updateQue{
		updated: make(map[[16]byte]time.Time),
	}
}

//run do doUpdateNode for each records using related nodes.
//if success to doUpdateNode, add node to updatelist and recentlist and
//removes the record from queue.
func (u *updateQue) updateNodes(rec *record, n *node) {
	log.Println("updating", rec)
	if u.doUpdateNode(rec, n) {
		recentList.append(rec)
		recentList.sync()
	}
}

//deleteOldUpdated removes old updated records from updated map.
func (u *updateQue) deleteOldUpdated() {
	for k, v := range u.updated {
		if v.After(time.Now().Add(oldUpdated)) {
			delete(u.updated, k)
		}
	}
}

//doUpdateNode broadcast and get data for each new records.
//if can get data (even if spam) return true, if fails to get, return false.
//if no fail, broadcast updates to node in cache and added n to nodelist and searchlist.
func (u *updateQue) doUpdateNode(rec *record, n *node) bool {
	u.mutex.Lock()
	if _, exist := u.updated[rec.hash()]; exist {
		return true
	}
	u.deleteOldUpdated()
	u.updated[rec.hash()] = time.Now()
	u.mutex.Unlock()

	ca := newCache(rec.datfile)
	var err error
	switch {
	case !ca.Exists(), n == nil:
		log.Println("no cache, only broadcast updates.")
		nodeManager.tellUpdate(ca, rec.Stamp, rec.ID, n)
		return true
	case ca.Len() > 0:
		log.Println("cache and records exists, get data from node n.")
		err = ca.getData(rec.Stamp, rec.ID, n)
	default:
		log.Println("cache exists ,but no records. get data with range.")
		ca.getWithRange(n)
		if flagGot := rec.Exists(); !flagGot {
			err = errGet
		}
	}
	switch err {
	case errGet:
		log.Println("could not get")
		return false
	case errSpam:
		log.Println("makred spam")
		return true
	default:
		log.Println("telling update")
		nodeManager.tellUpdate(ca, rec.Stamp, rec.ID, nil)
		nodeManager.join(n)
		return true
	}
}
