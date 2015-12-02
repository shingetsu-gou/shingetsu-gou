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
	HeavyMoon         bool
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
	infos   map[string]recordHeads
	isDirty bool
	mutex   sync.RWMutex
}

//NewRecentList load the saved file and create a RecentList obj.
func NewRecentList(cfg *RecentListConfig) *RecentList {
	r := &RecentList{
		RecentListConfig: cfg,
		infos:            make(map[string]recordHeads),
	}
	r.loadFile()
	return r
}

//loadFile reads recentlist from the file and add as records.
func (r *RecentList) loadFile() {
	r.Fmutex.RLock()
	err := util.EachLine(r.Recent, func(line string, i int) error {
		vr, err := newRecordHeadFromLine(line)
		if err == nil {
			r.Fmutex.RUnlock()
			r.mutex.Lock()
			r.infos[vr.Datfile] = append(r.infos[vr.Datfile], vr)
			r.isDirty = true
			r.mutex.Unlock()
			r.Fmutex.RLock()
		}
		return nil
	})
	r.Fmutex.RUnlock()
	if err != nil {
		log.Println(err)
	}
}

//Newest returns newest record of datfile in the list.
//if not found returns nil.
func (r *RecentList) Newest(Datfile string) *RecordHead {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	var rh *RecordHead
	for _, v := range r.infos[Datfile] {
		if v.Datfile == Datfile && (rh == nil || rh.Stamp < v.Stamp) {
			rh = v
		}
	}
	return rh
}

//Append add a infos generated from the record.
func (r *RecentList) Append(rec *Record) {
	if loc := r.find(rec); loc >= 0 {
		return
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.infos[rec.Datfile] = append(r.infos[rec.Datfile], &rec.RecordHead)
	if r.HeavyMoon {
		if ca := NewCache(rec.Datfile); !ca.Exists() {
			ca.SetupDirectories()
		}
	}
	r.isDirty = true
}

//find finds records and returns index. returns -1 if not found.
func (r *RecentList) find(rec *Record) int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	for i, v := range r.infos[rec.Datfile] {
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
		i := r.infos[rec.Datfile]
		i, i[len(i)-1] = append(i[:l], i[l+1:]...), nil
	}
}

//removeInfo removes info rec
func (r *RecentList) removeInfo(rec *RecordHead) {
	r.mutex.RLock()
	for i, v := range r.infos[rec.Datfile] {
		r.mutex.RUnlock()
		if v.equals(rec) {
			r.mutex.Lock()
			defer r.mutex.Unlock()
			in := r.infos[rec.Datfile]
			in, in[len(r.infos)-1] = append(in[:i], in[i+1:]...), nil
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
	var result []string
	for _, vs := range r.infos {
		for _, v := range vs {
			result = append(result, v.Recstr())
		}
	}
	return result
}

//Sync remove old records and save to the file.
func (r *RecentList) Sync() {
	r.mutex.Lock()
	for _, recs := range r.infos {
		for i, rec := range recs {
			if defaultUpdateRange > 0 && rec.Stamp+int64(defaultUpdateRange) < time.Now().Unix() {
				recs, recs[len(recs)-1] = append(recs[:i], recs[i+1:]...), nil
			}
		}
	}
	r.mutex.Unlock()
	recstrSlice := r.getRecstrSlice()
	r.Fmutex.Lock()
	err := util.WriteSlice(r.Recent, recstrSlice)
	r.Fmutex.Unlock()
	if err != nil {
		log.Println(err)
	}
}

//Getall retrieves Recent records from nodes in searchlist and stores them.
//tags are shuffled and truncated to tagsize and stored to sugtags in cache.
//also source nodes are stored into lookuptable.
//also tags which Recentlist doen't have in sugtagtable are truncated
func (r *RecentList) Getall(all bool) {
	const searchNodes = 5

	var begin int64
	if r.RecentRange > 0 && !all {
		begin = time.Now().Unix() - r.RecentRange
	}
	nodes := r.NodeManager.Random(nil, searchNodes)
	var wg sync.WaitGroup
	for _, n := range nodes {
		wg.Add(1)
		go func(n *node.Node) {
			defer wg.Done()
			var res []string
			var err error
			res, err = n.Talk("/recent/"+strconv.FormatInt(begin, 10)+"-", false, nil)
			if err != nil {
				r.NodeManager.RemoveFromAllTable(n)
				log.Println(err)
				return
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
					r.NodeManager.AppendToTable(rec.Datfile, n)
				}
			}
		}(n)
	}
	wg.Wait()
	r.Sync()
	r.NodeManager.Sync()
	r.SuggestedTagTable.prune(r)
	r.SuggestedTagTable.sync()
}

//MakeRecentCachelist returns sorted cachelist copied from Recentlist.
//which doens't contain duplicate Caches.
func (r *RecentList) MakeRecentCachelist() Caches {
	r.mutex.RLock()
	var cl Caches
	for datfile := range r.infos {
		ca := NewCache(datfile)
		cl = append(cl, ca)
	}
	r.mutex.RUnlock()
	sort.Sort(sort.Reverse(NewSortByStamp(cl, true)))
	return cl
}

//GetRecords copies and returns recorcds in recentlist.
func (r *RecentList) GetRecords() []*RecordHead {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	var inf []*RecordHead
	for _, recs := range r.infos {
		for _, rec := range recs {
			inf = append(inf, rec)
		}
	}
	return inf
}

//CreateAllCachedirs creates all dirs in recentlist to be retrived when called recentlist.getall.
//(heavymoon)
func (r *RecentList) CreateAllCachedirs() {
	for _, rh := range r.GetRecords() {
		ca := NewCache(rh.Datfile)
		if !ca.Exists() {
			ca.SetupDirectories()
		}
	}
}
