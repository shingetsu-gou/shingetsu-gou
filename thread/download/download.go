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

package download

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/node"
	"github.com/shingetsu-gou/shingetsu-gou/node/manager"
	"github.com/shingetsu-gou/shingetsu-gou/recentlist"
	"github.com/shingetsu-gou/shingetsu-gou/record"
	"github.com/shingetsu-gou/shingetsu-gou/thread"
)

//targetRec represents target records for downloading.
type targetRec struct {
	node        node.Slice
	downloading *node.Node
	finished    bool
	count       int
	stamp       int64
}

//TargetRecSlice represents slice of targetRec
type TargetRecSlice []*targetRec

//Len returns length of TargetRecSlice
func (t TargetRecSlice) Len() int {
	return len(t)
}

//Swap swaps the location of TargetRecSlice
func (t TargetRecSlice) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

//Len returns true if stamp of targetRec[i] is less.
func (t TargetRecSlice) Less(i, j int) bool {
	return t[i].stamp < t[j].stamp
}

var managers = make(map[string]*DownloadManager)

//DownloadManager manages download range of records.
type DownloadManager struct {
	datfile string
	recs    map[string]*targetRec
	mutex   sync.RWMutex
}

//NewDownloadManger sets recs as finished recs and returns DownloadManager obj.
func NewDownloadManger(ca *thread.Cache) *DownloadManager {
	if d, exist := managers[ca.Datfile]; exist {
		log.Println(ca.Datfile, "is downloading")
		return d
	}
	recs := ca.LoadRecords(thread.All)
	dm := &DownloadManager{
		datfile: ca.Datfile,
		recs:    make(map[string]*targetRec),
	}
	for k := range recs {
		dm.recs[k] = &targetRec{
			finished: true,
		}
	}
	return dm
}

//Set sets res as targets n is holding.
func (dm *DownloadManager) Set(res []string, n *node.Node) {
	recs := record.ParseHeadResponse(res, dm.datfile)
	for _, r := range recs {
		dm.mutex.Lock()
		if rec, exist := dm.recs[r.Idstr()]; exist {
			if !rec.finished {
				rec.node = append(rec.node, n)
			}
		} else {
			dm.recs[r.Recstr()] = &targetRec{
				node:  []*node.Node{n},
				stamp: r.Stamp,
			}
		}
		dm.mutex.Unlock()
	}
}

//Get returns begin and end stamp to be gotten for node n.
func (dm *DownloadManager) Get(n *node.Node) (int64, int64) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	var s TargetRecSlice
	for _, rec := range dm.recs {
		if rec.node.Has(n) && !rec.finished && rec.downloading == nil && rec.count < 5 {
			s = append(s, rec)
		}
	}
	if len(s) == 0 {
		dm.checkFinished()
		return -1, -1
	}
	managers[dm.datfile] = dm
	sort.Sort(sort.Reverse(s))
	begin := len(s) - 1
	if len(s) > 5 {
		begin = len(s) / 2
	}
	for i := 0; i <= begin; i++ {
		s[i].downloading = n
	}
	return s[begin].stamp, s[0].stamp
}

func (dm *DownloadManager) checkFinished() {
	if _, exist := managers[dm.datfile]; !exist {
		return
	}
	finished := true
	for _, rec := range dm.recs {
		if rec.count >= 5 {
			rec.finished = true
		}
		if !rec.finished {
			finished = false
		}
	}
	if finished {
		log.Println(dm.datfile, ":finished downloading")
		managers[dm.datfile] = nil
		delete(managers, dm.datfile)
	}
}

//Finished set records n is downloading as finished.
func (dm *DownloadManager) Finished(n *node.Node, success bool) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	for _, rec := range dm.recs {
		if rec.downloading != nil && rec.downloading.Equals(n) {
			if success {
				rec.finished = true
			} else {
				rec.count++
			}
			rec.downloading = nil
		}
	}
	dm.checkFinished()
}

//headWithRange checks node n has records with range and adds records which should be downloaded to downloadmanager.
func headWithRange(n *node.Node, c *thread.Cache, dm *DownloadManager) bool {
	begin := time.Now().Unix() - cfg.GetRange
	if rec := recentlist.Newest(c.Datfile); rec != nil {
		begin = rec.Stamp - cfg.GetRange
	}
	if cfg.GetRange == 0 || begin < 0 {
		begin = 0
	}
	res, err := n.Talk(fmt.Sprintf("/head/%s/%d-", c.Datfile, begin), nil)
	if err != nil {
		return false
	}
	if len(res) == 0 {
		ress, errr := n.Talk(fmt.Sprintf("/have/%s", c.Datfile), nil)
		if errr != nil || len(ress) == 0 || ress[0] != "YES" {
			manager.RemoveFromTable(c.Datfile, n)
		} else {
			manager.AppendToTable(c.Datfile, n)
		}
		return false
	}
	manager.AppendToTable(c.Datfile, n)
	dm.Set(res, n)
	return true
}

//getWithRange gets records with range using node n and adds to cache after checking them.
//if no records exist in cache, uses head
//return true if gotten records>0
func getWithRange(n *node.Node, c *thread.Cache, dm *DownloadManager) bool {
	got := false
	for {
		from, to := dm.Get(n)
		if from <= 0 {
			return got
		}

		var okcount int
		_, err := n.Talk(fmt.Sprintf("/get/%s/%d-%d", c.Datfile, from, to), func(res string) error {
			err := c.CheckData(res, -1, "", from, to)
			if err == nil {
				okcount++
			}
			return nil
		})
		if err != nil {
			dm.Finished(n, false)
			return false
		}
		dm.Finished(n, true)
		log.Println(c.Datfile, okcount, "records were saved from", n.Nodestr)
		got = okcount > 0
	}
}

//GetCache checks  nodes in lookuptable have the cache.
//if found gets records.
func GetCache(background bool, c *thread.Cache) bool {
	const searchDepth = 5 // Search node size
	ns := manager.NodesForGet(c.Datfile, searchDepth)
	found := false
	done := make(chan struct{}, searchDepth+1)
	var wg sync.WaitGroup
	var mutex sync.RWMutex
	dm := NewDownloadManger(c)
	for _, n := range ns {
		wg.Add(1)
		go func(n *node.Node) {
			defer wg.Done()
			if !headWithRange(n, c, dm) {
				return
			}
			if getWithRange(n, c, dm) {
				mutex.Lock()
				found = true
				mutex.Unlock()
				done <- struct{}{}
				return
			}
		}(n)
	}
	waitFor(background, c, done, &wg)
	mutex.RLock()
	defer mutex.RUnlock()
	return found
}

func waitFor(background bool, c *thread.Cache, done chan struct{}, wg *sync.WaitGroup) {
	newest := recentlist.Newest(c.Datfile)
	if background {
		go func() {
			wg.Wait()
			done <- struct{}{}
		}()
		if newest != nil && (newest.Stamp == c.ReadInfo().Stamp) {
			return
		}
		for {
			select {
			case <-done:
				return
			case <-time.After(3 * time.Second):
				if c.HasRecord() {
					return
				}
			}
		}
	}
	wg.Wait()
}

//Getall reload all records in cache in cachelist from network.
func Getall() {
	for _, ca := range thread.AllCaches() {
		GetCache(false, ca)
	}
}
