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

package recentlist

import (
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/node"
	"github.com/shingetsu-gou/shingetsu-gou/node/manager"
	"github.com/shingetsu-gou/shingetsu-gou/record"
	"github.com/shingetsu-gou/shingetsu-gou/tag/suggest"
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

//RecentList represents records list udpated by remote host and
//gotten by /gateway.cgi/Recent
var infos = make(map[string]record.Heads)
var isDirty bool
var mutex sync.RWMutex

//NewRecentList load the saved file and create a RecentList obj.
func init() {
	loadFile()
}

//loadFile reads recentlist from the file and add as records.
func loadFile() {
	cfg.Fmutex.RLock()
	err := util.EachLine(cfg.Recent(), func(line string, i int) error {
		vr, err := record.NewHeadFromLine(line)
		if err == nil {
			cfg.Fmutex.RUnlock()
			mutex.Lock()
			infos[vr.Datfile] = append(infos[vr.Datfile], vr)
			isDirty = true
			mutex.Unlock()
			cfg.Fmutex.RLock()
		}
		return nil
	})
	cfg.Fmutex.RUnlock()
	if err != nil {
		log.Println(err)
	}
}

func Datfiles() []string {
	cfg.Fmutex.RLock()
	defer cfg.Fmutex.RUnlock()
	datfile := make([]string, len(infos))
	i := 0
	for df := range infos {
		datfile[i] = df
		i++
	}
	return datfile
}

//Newest returns newest record of datfile in the list.
//if not found returns nil.
func Newest(Datfile string) *record.Head {
	mutex.RLock()
	defer mutex.RUnlock()
	var rh *record.Head
	for _, v := range infos[Datfile] {
		if v.Datfile == Datfile && (rh == nil || rh.Stamp < v.Stamp) {
			rh = v
		}
	}
	return rh
}

//Append add a infos generated from the record.
func Append(rec *record.Record) {
	if loc := find(rec); loc >= 0 {
		return
	}
	mutex.Lock()
	defer mutex.Unlock()
	infos[rec.Datfile] = append(infos[rec.Datfile], rec.Head)

	isDirty = true
}

//find finds records and returns index. returns -1 if not found.
func find(rec *record.Record) int {
	mutex.RLock()
	defer mutex.RUnlock()
	for i, v := range infos[rec.Datfile] {
		if v.Equals(rec.Head) {
			return i
		}
	}
	return -1
}

//hasInfo returns true if has record r.
func hasInfo(rec *record.Record) bool {
	return find(rec) != -1
}

//remove removes info which is same as record rec
func remove(rec *record.Record) {
	if l := find(rec); l != -1 {
		mutex.Lock()
		defer mutex.Unlock()
		infos[rec.Datfile], infos[rec.Datfile][len(infos[rec.Datfile])-1] = append(infos[rec.Datfile][:l], infos[rec.Datfile][l+1:]...), nil
	}
}

//removeInfo removes info rec
func removeInfo(rec *record.Head) {
	mutex.RLock()
	for i, v := range infos[rec.Datfile] {
		mutex.RUnlock()
		if v.Equals(rec) {
			mutex.Lock()
			defer mutex.Unlock()
			infos[rec.Datfile], infos[rec.Datfile][len(infos[rec.Datfile])-1] = append(infos[rec.Datfile][:i], infos[rec.Datfile][i+1:]...), nil
			return
		}
		mutex.RLock()
	}
	mutex.RUnlock()
}

//getRecstrSlice returns slice of recstr string of infos.
func getRecstrSlice() []string {
	mutex.RLock()
	defer mutex.RUnlock()
	var result []string
	for _, vs := range infos {
		for _, v := range vs {
			result = append(result, v.Recstr())
		}
	}
	return result
}

//Sync remove old records and save to the file.
func Sync() {
	mutex.Lock()
	for _, recs := range infos {
		for i, rec := range recs {
			if defaultUpdateRange > 0 && rec.Stamp+int64(defaultUpdateRange) < time.Now().Unix() {
				recs, recs[len(recs)-1] = append(recs[:i], recs[i+1:]...), nil
			}
		}
	}
	mutex.Unlock()
	recstrSlice := getRecstrSlice()
	cfg.Fmutex.Lock()
	err := util.WriteSlice(cfg.Recent(), recstrSlice)
	cfg.Fmutex.Unlock()
	if err != nil {
		log.Println(err)
	}
}

//Getall retrieves Recent records from nodes in searchlist and stores them.
//tags are shuffled and truncated to tagsize and stored to sugtags in cache.
//also source nodes are stored into lookuptable.
//also tags which Recentlist doen't have in sugtagtable are truncated
func Getall(all bool) {
	const searchNodes = 5

	var begin int64
	if cfg.RecentRange > 0 && !all {
		begin = time.Now().Unix() - cfg.RecentRange
	}
	nodes := manager.Random(nil, searchNodes)
	var wg sync.WaitGroup
	for _, n := range nodes {
		wg.Add(1)
		go func(n *node.Node) {
			defer wg.Done()
			var res []string
			var err error
			res, err = n.Talk("/recent/"+strconv.FormatInt(begin, 10)+"-", nil)
			if err != nil {
				manager.RemoveFromAllTable(n)
				log.Println(err)
				return
			}
			for _, line := range res {
				rec := record.Make(line)
				if rec == nil {
					continue
				}
				Append(rec)
				tags := strings.Fields(strings.TrimSpace(rec.GetBodyValue("tag", "")))
				if len(tags) > cfg.TagSize {
					util.Shuffle(sort.StringSlice(tags))
					tags = tags[:cfg.TagSize]
				}
				if len(tags) > 0 {
					suggest.AddString(rec.Datfile, tags)
					manager.AppendToTable(rec.Datfile, n)
				}
			}
		}(n)
	}
	wg.Wait()
	Sync()
	manager.Sync()
	suggest.Prune(GetRecords())
	suggest.Save()
}

//GetRecords copies and returns recorcds in recentlist.
func GetRecords() []*record.Head {
	mutex.RLock()
	defer mutex.RUnlock()
	var inf []*record.Head
	for _, recs := range infos {
		for _, rec := range recs {
			inf = append(inf, rec)
		}
	}
	return inf
}
