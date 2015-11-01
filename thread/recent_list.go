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
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/node"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

const defaultUpdateRange = 24 * time.Hour // Seconds

//IsInUpdateRange returns true if stamp is in updateRange.
func IsInUpdateRange(nstamp int64) bool {
	now := time.Now()
	if now.Add(-defaultUpdateRange).Unix() < nstamp && nstamp < now.Add(defaultUpdateRange).Unix() {
		return true
	}
	return false
}

//RecentListConfig is confi for Recentlist struct.
type RecentListConfig struct {
	RecentRange       int64
	TagSize           int
	Recent            string
	Fmutex            *sync.RWMutex
	NodeManager       *node.Manager
	SuggestedTagTable *SuggestedTagTable
}

//RecentList represents records list udpated by remote host and
//gotten by /gateway.cgi/Recent
type RecentList struct {
	*RecentListConfig
	infos   recordHeads
	isDirty bool
	mutex   sync.RWMutex
}

//NewRecentList load the saved file and create a RecentList obj.
func NewRecentList(cfg *RecentListConfig) *RecentList {
	r := &RecentList{
		RecentListConfig: cfg,
	}
	r.loadFile()
	return r
}

//loadFile reads recentlist from the file and add as records.
func (r *RecentList) loadFile() {
	r.Fmutex.RLock()
	defer r.Fmutex.RUnlock()
	err := util.EachLine(r.Recent, func(line string, i int) error {
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

//Newest returns newest record of datfile in the list.
//if not found returns nil.
func (r *RecentList) Newest(Datfile string) *RecordHead {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	for _, v := range r.infos {
		if v.Datfile == Datfile {
			return v
		}
	}
	return nil
}

//Append add a infos generated from the record.
func (r *RecentList) Append(rec *Record) {
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
func (r *RecentList) find(rec *Record) int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	for i, v := range r.infos {
		if v.equals(&rec.RecordHead) {
			return i
		}
	}
	return -1
}

//hasInfo returns true if has record r.
func (r *RecentList) hasInfo(rec *Record) bool {
	return r.find(rec) != -1
}

//remove removes info which is same as record rec
func (r *RecentList) remove(rec *Record) {
	if l := r.find(rec); l != -1 {
		r.mutex.Lock()
		defer r.mutex.Unlock()
		r.infos = append(r.infos[:l], r.infos[l+1:]...)
	}
}

//removeInfo removes info rec
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
		result[i] = v.Recstr()
	}
	return result
}

//Sync remove old records and save to the file.
func (r *RecentList) Sync() {
	r.mutex.Lock()
	for i, rec := range r.infos {
		if defaultUpdateRange > 0 && rec.Stamp+int64(defaultUpdateRange) < time.Now().Unix() {
			r.infos, r.infos[len(r.infos)-1] = append(r.infos[:i], r.infos[i+1:]...), nil
		}
	}
	r.mutex.Unlock()
	r.Fmutex.Lock()
	err := util.WriteSlice(r.Recent, r.getRecstrSlice())
	r.Fmutex.Unlock()
	if err != nil {
		log.Println(err)
	}
}

//Getall retrieves Recent records from nodes in searchlist and stores them.
//tags are shuffled and truncated to tagsize and stored to sugtags in cache.
//also source nodes are stored into lookuptable.
//also tags which Recentlist doen't have in sugtagtable are truncated
func (r *RecentList) Getall() {
	const searchNodes = 5

	var begin int64
	if r.RecentRange > 0 {
		begin = time.Now().Unix() - r.RecentRange
	}
	nodes := r.NodeManager.Random(nil, searchNodes)
	var res []string
	for _, n := range nodes {
		var err error
		res, err = n.Talk("/Recent/" + strconv.FormatInt(begin, 10) + "-")
		if err != nil {
			r.NodeManager.RemoveFromAllTable(n)
			log.Println(err)
			continue
		}
		for _, line := range res {
			rec := makeRecord(line)
			if rec == nil {
				continue
			}
			r.Append(rec)
			tags := strings.Fields(strings.TrimSpace(rec.GetBodyValue("tag", "")))
			if len(tags) > r.TagSize {
				util.Shuffle(sort.StringSlice(tags))
				tags = tags[:r.TagSize]
			}
			if len(tags) > 0 {
				r.SuggestedTagTable.addString(rec.Datfile, tags)
				r.SuggestedTagTable.sync()
				r.NodeManager.AppendToTable(rec.Datfile, n)
			}
		}
	}
	r.Sync()
	r.NodeManager.Sync()
	r.SuggestedTagTable.prune(r)
	r.SuggestedTagTable.sync()
}

//MakeRecentCachelist returns sorted cachelist copied from Recentlist.
//which doens't contain duplicate Caches.
func (r *RecentList) MakeRecentCachelist() Caches {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	var cl Caches
	var check []string
	for _, rec := range r.infos {
		if !util.HasString(check, rec.Datfile) {
			ca := NewCache(rec.Datfile)
			cl = append(cl, ca)
			check = append(check, rec.Datfile)
		}
	}
	sort.Sort(sort.Reverse(SortByRecentStamp{cl}))
	return cl
}

//GetRecords copies and returns recorcds in recentlist.
func (r *RecentList) GetRecords() []*RecordHead {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	inf := make([]*RecordHead, len(r.infos))
	copy(inf, r.infos)
	return inf
}
