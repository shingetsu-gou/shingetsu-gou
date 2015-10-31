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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultUpdateRange = 24 * time.Hour // Seconds

//isInUpdateRange returns true if stamp is in updateRange.
func isInUpdateRange(nstamp int64) bool {
	now := time.Now()
	if now.Add(-defaultUpdateRange).Unix() < nstamp && nstamp < now.Add(defaultUpdateRange).Unix() {
		return true
	}
	return false
}

type RecentListConfig struct {
	recentRange       int64
	tagSize           int
	recent            string
	fmutex            *sync.RWMutex
	nodeManager       *NodeManager
	suggestedTagTable *SuggestedTagTable
}

//RecentList represents records list udpated by remote host and
//gotten by /gateway.cgi/recent
type RecentList struct {
	*RecentListConfig
	infos   recordHeads
	isDirty bool
	mutex   sync.RWMutex
}

//newRecentList load a file and create a RecentList obj.
func newRecentList(cfg *RecentListConfig) *RecentList {
	r := &RecentList{
		RecentListConfig: cfg,
	}
	r.loadFile()
	return r
}

//loadFile reads from file and add records.
func (r *RecentList) loadFile() {
	r.fmutex.RLock()
	defer r.fmutex.RUnlock()
	err := eachLine(r.recent, func(line string, i int) error {
		vr, err := newRecordHeadFromLine(line)
		if err == nil {
			r.mutex.Lock()
			r.infos = append(r.infos, vr)
			r.isDirty = true
			r.mutex.Unlock()
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}
}

//append add a infos generated from the record.
func (r *RecentList) newest(datfile string) *RecordHead {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	for _, v := range r.infos {
		if v.datfile == datfile {
			return v
		}
	}
	return nil
}

//append add a infos generated from the record.
func (r *RecentList) append(rec *record) {
	loc := r.find(rec)
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if loc >= 0 {
		if r.infos[loc].Stamp > rec.Stamp {
			return
		}
		r.infos[loc] = &rec.RecordHead
	}
	r.infos = append(r.infos, &rec.RecordHead)
	sort.Sort(r.infos)
	r.isDirty = true
}

//find finds records and returns index. returns -1 if not found.
func (r *RecentList) find(rec *record) int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	for i, v := range r.infos {
		if v.equals(&rec.RecordHead) {
			return i
		}
	}
	return -1
}

//hasRecord returns true if has record r.
func (r *RecentList) hasInfo(rec *record) bool {
	return r.find(rec) != -1
}

//remove removes info which is same as record r
func (r *RecentList) remove(rec *record) {
	if l := r.find(rec); l != -1 {
		r.mutex.Lock()
		defer r.mutex.Unlock()
		r.infos = append(r.infos[:l], r.infos[l+1:]...)
	}
}

//removeInfo removes info r
func (r *RecentList) removeInfo(rec *RecordHead) {
	r.mutex.RLock()
	for i, v := range r.infos {
		r.mutex.RUnlock()
		if v.equals(rec) {
			r.mutex.Lock()
			defer r.mutex.Unlock()
			r.infos, r.infos[len(r.infos)-1] = append(r.infos[:i], r.infos[i+1:]...), nil
			return
		}
		r.mutex.RLock()
	}
	r.mutex.RUnlock()
}

//getRecstrSlice returns slice of recstr string of infos.
func (r *RecentList) getRecstrSlice() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	result := make([]string, len(r.infos))
	for i, v := range r.infos {
		result[i] = v.recstr()
	}
	return result
}

//sync remove old records and save to the file.
func (r *RecentList) sync() {

	r.mutex.Lock()
	for i, rec := range r.infos {
		if defaultUpdateRange > 0 && rec.Stamp+int64(defaultUpdateRange) < time.Now().Unix() {
			r.infos, r.infos[len(r.infos)-1] = append(r.infos[:i], r.infos[i+1:]...), nil
		}
	}
	r.mutex.Unlock()
	r.fmutex.Lock()
	err := writeSlice(r.recent, r.getRecstrSlice())
	r.fmutex.Unlock()
	if err != nil {
		log.Println(err)
	}
}

//getAll retrieves recent records from nodes in searchlist and stores them.
//tags are shuffled and truncated to tagsize and stored to sugtags in cache.
//also source nodes are stored into lookuptable.
//also tags which recentlist doen't have in sugtagtable are truncated
func (r *RecentList) getAll() {
	const searchNodes = 5

	var begin int64
	if r.recentRange > 0 {
		begin = time.Now().Unix() - r.recentRange
	}
	nodes := r.nodeManager.random(nil, searchNodes)
	var res []string
	for _, n := range nodes {
		var err error
		res, err = n.talk("/recent/" + strconv.FormatInt(begin, 10) + "-")
		if err != nil {
			r.nodeManager.removeFromAllTable(n)
			log.Println(err)
			continue
		}
		for _, line := range res {
			rec := makeRecord(line)
			if rec == nil {
				continue
			}
			r.append(rec)
			tags := strings.Fields(strings.TrimSpace(rec.GetBodyValue("tag", "")))
			if len(tags) > r.tagSize {
				shuffle(sort.StringSlice(tags))
				tags = tags[:r.tagSize]
			}
			if len(tags) > 0 {
				r.suggestedTagTable.addString(rec.datfile, tags)
				r.suggestedTagTable.sync()
				r.nodeManager.appendToTable(rec.datfile, n)
			}
		}
	}
	r.sync()
	r.nodeManager.sync()
	r.suggestedTagTable.prune(r)
	r.suggestedTagTable.sync()
}

//makeRecentCachelist returns sorted cachelist copied from recentlist.
//which doens't contain duplicate caches.
func (r *RecentList) makeRecentCachelist() caches {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	var cl caches
	var check []string
	for _, rec := range r.infos {
		if !hasString(check, rec.datfile) {
			ca := NewCache(rec.datfile)
			cl = append(cl, ca)
			check = append(check, rec.datfile)
		}
	}
	sort.Sort(sort.Reverse(sortByRecentStamp{cl}))
	return cl
}
