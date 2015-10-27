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
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

//RecordHead represents one line in updatelist/recentlist
type RecordHead struct {
	datfile string //cache file name
	Stamp   int64  //unixtime
	ID      string //md5(bodystr)
}

//newUpdateInfoFromLine parse one line in udpate/recent list and returns updateInfo obj.
func newRecordHeadFromLine(line string) (*RecordHead, error) {
	strs := strings.Split(strings.TrimRight(line, "\n\r"), "<>")
	if len(strs) < 3 {
		err := errors.New("illegal format")
		log.Println(err)
		return nil, err
	}
	u := &RecordHead{
		ID:      strs[1],
		datfile: strs[2],
	}
	var err error
	u.Stamp, err = strconv.ParseInt(strs[0], 10, 64)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return u, nil
}

//equals returns true if u=v
func (u *RecordHead) equals(rec *RecordHead) bool {
	return u.datfile == rec.datfile && u.ID == rec.ID && u.Stamp == rec.Stamp
}

//hash returns md5 of RecordHead.
func (u *RecordHead) hash() [16]byte {
	m := md5.New()
	m.Write([]byte(u.datfile))
	binary.Write(m, binary.LittleEndian, u.Stamp)
	m.Write([]byte(u.ID))
	var r [16]byte
	m.Sum(r[:])
	return r
}

//recstr returns one line of update/recentlist file.
func (u *RecordHead) recstr() string {
	return fmt.Sprintf("%d<>%s<>%s", u.Stamp, u.ID, u.datfile)
}

//Idstr returns real file name of the record file.
func (u *RecordHead) Idstr() string {
	return fmt.Sprintf("%d_%s", u.Stamp, u.ID)
}

type recordHeads []*RecordHead

//Less returns true if stamp of infos[i] < [j]
func (r recordHeads) Less(i, j int) bool {
	return r[i].Stamp < r[j].Stamp
}

//Swap swaps infos order.
func (r recordHeads) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

//Len returns size of infos
func (r recordHeads) Len() int {
	return len(r)
}

//has returns true if recordHeads has rec.
func (r recordHeads) has(rec *RecordHead) bool {
	for _, v := range r {
		if v.equals(rec) {
			return true
		}
	}
	return false
}

//RecentList represents records list udpated by remote host and
//gotten by /gateway.cgi/recent
type RecentList struct {
	infos   recordHeads
	isDirty bool
	mutex   sync.RWMutex
}

//newRecentList load a file and create a RecentList obj.
func newRecentList() *RecentList {
	r := &RecentList{}
	r.loadFile()
	return r
}

//loadFile reads from file and add records.
func (r *RecentList) loadFile() {
	fmutex.RLock()
	defer fmutex.RUnlock()
	err := eachLine(recent, func(line string, i int) error {
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
	fmutex.Lock()
	err := writeSlice(recent, r.getRecstrSlice())
	fmutex.Unlock()
	if err != nil {
		log.Println(err)
	}
}

//getAll retrieves recent records from nodes in searchlist and stores them.
//tags are shuffled and truncated to tagsize and stored to sugtags in cache.
//also source nodes are stored into lookuptable.
//also tags which recentlist doen't have in sugtagtable are truncated
func (r *RecentList) getAll() {
	var begin int64
	if recentRange > 0 {
		begin = time.Now().Unix() - recentRange
	}
	nodes := nodeManager.random(nil, shareNodes)
	var res []string
	for count, n := range nodes {
		var err error
		res, err = n.talk("/recent/" + strconv.FormatInt(begin, 10) + "-")
		if err != nil {
			nodeManager.removeFromAllTable(n)
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
			if len(tags) > tagSize {
				shuffle(sort.StringSlice(tags))
				tags = tags[:tagSize]
			}
			if len(tags) > 0 {
				suggestedTagTable.addString(rec.datfile, tags)
				suggestedTagTable.sync()
				nodeManager.appendToTable(rec.datfile, n)
			}
		}
		if count >= searchDepth {
			break
		}
	}
	r.sync()
	nodeManager.sync()
	suggestedTagTable.prune(r)
	suggestedTagTable.sync()
}

//makeRecentCachelist returns sorted cachelist copied from recentlist.
//which doens't contain duplicate caches.
func (r *RecentList) makeRecentCachelist() *cacheList {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	var cl caches
	var check []string
	for _, rec := range r.infos {
		if !hasString(check, rec.datfile) {
			ca := newCache(rec.datfile)
			cl = append(cl, ca)
			check = append(check, rec.datfile)
		}
	}
	sort.Sort(sort.Reverse(sortByRecentStamp{cl}))
	return &cacheList{cl}
}
