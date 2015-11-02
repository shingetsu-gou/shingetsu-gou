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

package thread

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/node"
)

//UpdateQueConfig is a config of UpdateQue.
type UpdateQueConfig struct {
	RecentList  *RecentList
	NodeManager *node.Manager
}

//UpdateQue is for telling updates of records.
//it records hash of updated records for 1 hour not to tell again.
type UpdateQue struct {
	*UpdateQueConfig
	mutex   sync.Mutex
	updated map[[16]byte]time.Time
}

//NewUpdateQue makes and returns UpdateQue obj.
func NewUpdateQue(cfg *UpdateQueConfig) *UpdateQue {
	return &UpdateQue{
		UpdateQueConfig: cfg,
		updated:         make(map[[16]byte]time.Time),
	}
}

//UpdateNodes do doUpdateNode for each records using related nodes.
//if success to doUpdateNode, add node to updatelist and recentlist and
//removes the record from queue.
func (u *UpdateQue) UpdateNodes(rec *Record, n *node.Node) {
	if u.doUpdateNode(rec, n) {
		u.RecentList.Append(rec)
		u.RecentList.Sync()
	}
}

//deleteOldUpdated removes old updated records from updated map.
func (u *UpdateQue) deleteOldUpdated() {
	const oldUpdated = time.Hour

	for k, v := range u.updated {
		if time.Now().After(v.Add(oldUpdated)) {
			delete(u.updated, k)
		}
	}
}

//doUpdateNode broadcast and get data for each new records.
//if can get data (even if spam) return true, if fails to get, return false.
//if no fail, broadcast updates to node in cache and added n to nodelist and searchlist.
func (u *UpdateQue) doUpdateNode(rec *Record, n *node.Node) bool {
	errGet := errors.New("cannot get data")

	u.mutex.Lock()
	u.deleteOldUpdated()
	if _, exist := u.updated[rec.hash()]; exist {
		log.Println("already broadcasted", rec.ID)
		return true
	}
	u.updated[rec.hash()] = time.Now()
	u.mutex.Unlock()

	ca := NewCache(rec.Datfile)
	var err error
	switch {
	case !ca.Exists(), n == nil:
		log.Println("no cache, only broadcast updates.")
		u.NodeManager.TellUpdate(ca.Datfile, rec.Stamp, rec.ID, n)
		return true
	case ca.HasRecord():
		log.Println("cache and records exists, get data from node n.")
		err = ca.GetData(rec.Stamp, rec.ID, n)
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
		log.Println("marked spam")
		return true
	default:
		log.Println("telling update")
		u.NodeManager.TellUpdate(ca.Datfile, rec.Stamp, rec.ID, nil)
		u.NodeManager.Join(n)
		return true
	}
}
