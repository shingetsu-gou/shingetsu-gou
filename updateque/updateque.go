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

package updateque

import (
	"log"
	"sync"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/node"
	"github.com/shingetsu-gou/shingetsu-gou/node/manager"
	"github.com/shingetsu-gou/shingetsu-gou/recentlist"
	"github.com/shingetsu-gou/shingetsu-gou/record"
	"github.com/shingetsu-gou/shingetsu-gou/thread"
)

//UpdateQue is for telling updates of records.
//it records hash of updated records for 1 hour not to tell again.
var mutex sync.Mutex
var updated = make(map[[16]byte]time.Time)

//UpdateNodes do doUpdateNode for each records using related nodes.
//if success to doUpdateNode, add node to updatelist and recentlist and
//removes the record from queue.
func UpdateNodes(rec *record.Record, n *node.Node) {
	if doUpdateNode(rec, n) {
		recentlist.Append(rec.Head)
		if cfg.HeavyMoon {
			if ca := thread.NewCache(rec.Datfile); !ca.Exists() {
				ca.Subscribe()
			}
		}
	}
}

//deleteOldUpdated removes old updated records from updated map.
func deleteOldUpdated() {
	const oldUpdated = time.Hour

	for k, v := range updated {
		if time.Now().After(v.Add(oldUpdated)) {
			delete(updated, k)
		}
	}
}

//RecordChannel is for informing record was gotten.
type RecordChannel struct {
	*record.Head
	Ch    chan struct{}
	mutex sync.RWMutex
}

//regist registers record to RecordChannel.
func (r *RecordChannel) register(h *record.Head) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.Ch == nil {
		r.Ch = make(chan struct{})
	}
	r.Head = h
}

//Inform call channel if specifiedd condition is met.
func (r *RecordChannel) Inform(datfile, id string, begin, end int64) {
	r.mutex.Lock()
	if r.Head != nil && r.Datfile == datfile && begin <= r.Stamp && r.Stamp <= end && r.ID == id {
		r.Head = nil
		r.mutex.Unlock()
		r.Ch <- struct{}{}
		return
	}
	r.mutex.Unlock()
}

//wait waits for channel or 1 minutex.
func (r *RecordChannel) wait() bool {
	select {
	case <-r.Ch:
		return true
	case <-time.After(time.Minute):
		return false
	}
}

//UpdatedRecord is concerned record.
var UpdatedRecord RecordChannel

//doUpdateNode broadcast and get data for each new records.
//if can get data (even if spam) return true, if fails to get, return false.
//if no fail, broadcast updates to node in cache and added n to nodelist and searchlist.
func doUpdateNode(rec *record.Record, n *node.Node) bool {
	mutex.Lock()
	deleteOldUpdated()
	if _, exist := updated[rec.Hash()]; exist {
		log.Println("already broadcasted", rec.ID)
		mutex.Unlock()
		return true
	}
	updated[rec.Hash()] = time.Now()
	mutex.Unlock()

	ca := thread.NewCache(rec.Datfile)
	var err error
	if !ca.Exists() || n == nil {
		log.Println("no cache or updates by myself, broadcast updates.")
		UpdatedRecord.register(rec.Head)
		manager.TellUpdate(ca.Datfile, rec.Stamp, rec.ID, n)
		if UpdatedRecord.wait() || n != nil {
			log.Println(rec.ID, "was gotten or don't have the record")
		} else {
			log.Println(rec.ID, "was NOT gotten, will call updates later")
			go func() {
				time.Sleep(10 * time.Minute)
				UpdateNodes(rec, n)
			}()
		}
		return true
	}
	log.Println("cache exists. get record from node n.")
	err = rec.GetData(n)
	switch err {
	case cfg.ErrGet:
		log.Println("could not get")
		return false
	case cfg.ErrSpam:
		log.Println("marked spam")
		return true
	default:
		log.Println("telling update")
		manager.TellUpdate(ca.Datfile, rec.Stamp, rec.ID, nil)
		manager.Join(n)
		return true
	}
}
