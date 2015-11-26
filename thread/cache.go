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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/node"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

var (
	errSpam = errors.New("this is spam")
	errGet  = errors.New("cannot get data")
	//CacheCfg is config for Cache struct.it must be set before using it.
	CacheCfg *CacheConfig
)

//CacheInfo represents size/len/velocity of cache.
type CacheInfo struct {
	Size     int64 //size of total records
	Len      int   //# of records
	Velocity int   //# of new records in one day
	Stamp    int64 //stamp of newest record
	Oldest   int64 //oldeest stamp
}

//CacheConfig is config for Cache struct.
type CacheConfig struct {
	CacheDir          string
	RecordLimit       int
	SyncRange         int64
	GetRange          int64
	NodeManager       *node.Manager
	UserTag           *UserTag
	SuggestedTagTable *SuggestedTagTable
	RecentList        *RecentList
	Fmutex            *sync.RWMutex
}

//Cache represents cache of one file.
type Cache struct {
	*CacheConfig
	Datfile string
	tags    Tagslice //made by the user
	mutex   sync.RWMutex
}

//NewCache read tag files to set and returns cache obj.
//it uses sync.pool to ensure that only one cache obj exists for one datfile.
//and garbage collected when not used.
func NewCache(datfile string) *Cache {
	if CacheCfg == nil {
		log.Fatal("must set CacheCfg")
	}
	c := &Cache{
		Datfile:     datfile,
		CacheConfig: CacheCfg,
	}
	c.tags = loadTagslice(path.Join(c.Datpath(), "tag.txt"))
	return c
}

//AddTags add user tag list from vals.
func (c *Cache) AddTags(vals []string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tags = c.tags.addString(vals)
}

//SetTags sets user tag list from vals.
func (c *Cache) SetTags(vals []string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tags = newTagslice(vals)
}

//LenTags returns # of set user tag.
func (c *Cache) LenTags() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.tags.Len()
}

//TagString returns string of user tag.
func (c *Cache) TagString() string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.tags.string()
}

//GetTagstrSlice returns tagstr slice of user tag.
func (c *Cache) GetTagstrSlice() []string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.tags.GetTagstrSlice()
}

//GetTags returns copy of usertags.
func (c *Cache) GetTags() Tagslice {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	ts := make([]*Tag, c.tags.Len())
	copy(ts, c.tags)
	return Tagslice(ts)
}

//HasTagstr returns true if tag has tagstr.
func (c *Cache) HasTagstr(tagstr string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.tags.HasTagstr(tagstr)
}

//HasTag returns true if cache has tagstr=board tag in usertag or sugtag.
func (c *Cache) HasTag(board string) bool {
	if c.SuggestedTagTable.HasTagstr(c.Datfile, board) {
		return true
	}
	return c.HasTagstr(board)
}

//dathash returns datfile itself is type=asis.
func (c *Cache) dathash() string {
	return util.FileHash(c.Datfile)
}

//Datpath returns real file path of this cache.
func (c *Cache) Datpath() string {
	return path.Join(c.CacheDir, c.dathash())
}

//RecentStamp  returns time of getting by /recent.
func (c *Cache) RecentStamp() int64 {
	n := c.RecentList.Newest(c.Datfile)
	s := c.ReadInfo().Stamp
	if n == nil || n.Stamp < s {
		return s
	}
	return n.Stamp
}

//ReadInfo reads cache info from disk and returns #,velocity, and total size.
func (c *Cache) ReadInfo() *CacheInfo {
	c.Fmutex.RLock()
	defer c.Fmutex.RUnlock()
	ci := &CacheInfo{}
	d := path.Join(c.Datpath(), "record")
	if !util.IsDir(d) {
		return ci
	}
	err := util.EachFiles(d, func(dir os.FileInfo) error {
		stamp, err := strconv.ParseInt(strings.Split(dir.Name(), "_")[0], 10, 64)
		if err != nil {
			log.Println(err)
			return nil
		}
		if ci.Stamp < stamp {
			ci.Stamp = stamp
		}
		if ci.Oldest == 0 || ci.Oldest > stamp {
			ci.Oldest = stamp
		}
		if time.Unix(stamp, 0).After(time.Now().Add(-7 * 24 * time.Hour)) {
			ci.Velocity++
		}
		ci.Size += dir.Size()
		ci.Len++
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return ci
}

//LoadAllRecords loads and returns record maps from the disk including removed.
func (c *Cache) LoadAllRecords() RecordMap {
	recs := c.loadRecords("record")
	for k, v := range c.loadRecords("removed") {
		recs[k] = v
	}
	return recs
}

//LoadRecords loads and returns record maps from the disk .
func (c *Cache) LoadRecords() RecordMap {
	return c.loadRecords("record")
}

//loadRecords loads and returns record maps on path.
func (c *Cache) loadRecords(rpath string) RecordMap {
	r := path.Join(c.Datpath(), rpath)
	if !util.IsDir(r) {
		return nil
	}
	if !c.Exists() {
		return nil
	}
	recs := make(map[string]*Record)
	c.Fmutex.RLock()
	defer c.Fmutex.RUnlock()
	err := util.EachFiles(r, func(f os.FileInfo) error {
		recs[f.Name()] = NewRecord(c.Datfile, f.Name())
		return nil
	})
	if err != nil {
		log.Println(err, c.Datpath())
	}
	return RecordMap(recs)
}

//HasRecord return true if  cache has more than one records or removed records.
func (c *Cache) HasRecord() bool {
	c.Fmutex.RLock()
	defer c.Fmutex.RUnlock()
	f, err := ioutil.ReadDir(path.Join(c.Datpath(), "record"))
	if err != nil {
		return false
	}
	removed := path.Join(c.Datpath(), "removed")
	d, err := ioutil.ReadDir(removed)
	return len(f) > 0 || (err == nil && len(d) > 0)
}

//SyncTag saves usertags to files.
func (c *Cache) SyncTag() {
	c.Fmutex.Lock()
	defer c.Fmutex.Unlock()
	c.tags.sync(path.Join(c.Datpath(), "tag.txt"))
}

//SetupDirectories make necessary dirs.
func (c *Cache) SetupDirectories() {
	c.Fmutex.Lock()
	defer c.Fmutex.Unlock()
	for _, d := range []string{"", "/record", "/removed"} {
		di := path.Join(c.Datpath(), d)
		if !util.IsDir(di) {
			err := os.Mkdir(di, 0755)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

//checkData makes a record from res and checks its records meets condisions of args.
//adds the rec to cache if meets conditions.
//if spam or big data, remove the rec from disk.
//returns count of added records to the cache and spam/getting error.
func (c *Cache) checkData(res string, stamp int64, id string, begin, end int64) error {
	r := NewRecord(c.Datfile, "")
	if errr := r.parse(res); errr != nil {
		return errGet

	}
	if r.Exists() || r.Removed() {
		return nil
	}
	r.Sync()
	return r.checkData(begin, end)
}

//Remove Remove all files and dirs of cache.
func (c *Cache) Remove() {
	c.Fmutex.Lock()
	defer c.Fmutex.Unlock()
	err := os.RemoveAll(c.Datpath())
	if err != nil {
		log.Println(err)
	}
}

//Exists return true is datapath exists.
func (c *Cache) Exists() bool {
	c.Fmutex.RLock()
	defer c.Fmutex.RUnlock()
	return util.IsDir(c.Datpath())
}

//headWithRange checks node n has records with range and adds records which should be downloaded to downloadmanager.
func (c *Cache) headWithRange(n *node.Node, dm *DownloadManager) bool {
	begin := time.Now().Unix() - c.GetRange
	if rec := c.RecentList.Newest(c.Datfile); rec != nil {
		begin = rec.Stamp - c.GetRange
	}
	if c.GetRange == 0 || begin < 0 {
		begin = 0
	}
	res, err := n.Talk(fmt.Sprintf("/head/%s/%d-", c.Datfile, begin), false, nil)
	if err != nil {
		return false
	}
	if len(res) == 0 {
		ress, errr := n.Talk(fmt.Sprintf("/have/%s", c.Datfile), false, nil)
		if errr != nil || len(ress) == 0 || ress[0] != "YES" {
			c.NodeManager.RemoveFromTable(c.Datfile, n)
		} else {
			c.NodeManager.AppendToTable(c.Datfile, n)
			c.NodeManager.Sync()
		}
		return false
	}
	dm.Set(res, n)
	return true
}

//getWithRange gets records with range using node n and adds to cache after checking them.
//if no records exist in cache, uses head
//return true if gotten records>0
func (c *Cache) getWithRange(n *node.Node, dm *DownloadManager) bool {
	got := false
	for {
		from, to := dm.Get(n)
		if from <= 0 {
			return got
		}

		var okcount int
		_, err := n.Talk(fmt.Sprintf("/get/%s/%d-%d", c.Datfile, from, to), false, func(res string) error {
			err := c.checkData(res, -1, "", from, to)
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
func (c *Cache) GetCache(background bool) bool {
	const searchDepth = 5 // Search node size
	ns := c.NodeManager.NodesForGet(c.Datfile, searchDepth)
	found := false
	done := make(chan struct{}, searchDepth+1)
	var wg sync.WaitGroup
	var mutex sync.RWMutex
	dm := NewDownloadManger(c)
	for _, n := range ns {
		wg.Add(1)
		go func(n *node.Node) {
			defer wg.Done()
			if !c.headWithRange(n, dm) {
				return
			}
			if c.getWithRange(n, dm) {
				c.NodeManager.AppendToTable(c.Datfile, n)
				c.NodeManager.Sync()
				mutex.Lock()
				found = true
				mutex.Unlock()
				done <- struct{}{}
				return
			}
			c.NodeManager.RemoveFromTable(c.Datfile, n)
		}(n)
	}
	switch {
	case c.RecentList.Newest(c.Datfile).Stamp == c.ReadInfo().Stamp:
	case background:
		go func() {
			wg.Wait()
			done <- struct{}{}
		}()
	b:
		for {
			select {
			case <-done:
				break b
			case <-time.After(3 * time.Second):
				if c.HasRecord() {
					break b
				}
			}
		}
	default:
		wg.Wait()
	}
	mutex.RLock()
	defer mutex.RUnlock()
	return found
}

//Gettitle returns title part if *_*.
//returns ca.datfile if not.
func (c *Cache) Gettitle() string {
	if strings.HasPrefix(c.Datfile, "thread_") {
		return util.FileDecode(c.Datfile)
	}
	return c.Datfile
}

//GetContents returns recstrs of cache.
//len(recstrs) is <=2.
func (c *Cache) GetContents() []string {
	contents := make([]string, 0, 2)
	recs := c.LoadRecords()
	for _, rec := range recs {
		err := rec.Load()
		if err != nil {
			log.Println(err)
		}
		contents = append(contents, util.Escape(rec.Recstr()))
		if len(contents) > 2 {
			return contents
		}
	}
	return contents
}

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
func NewDownloadManger(ca *Cache) *DownloadManager {
	if d, exist := managers[ca.Datfile]; exist {
		log.Println(ca.Datfile, "is downloading")
		return d
	}
	recs := ca.LoadAllRecords()
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
	for _, r := range res {
		s := strings.Split(r, "<>")
		if len(s) != 2 {
			log.Println("format is illegal", s)
			continue
		}
		recstr := s[0] + "_" + s[1]
		stamp, err := strconv.ParseInt(s[0], 10, 64)
		if err != nil {
			log.Println(err)
			continue
		}
		dm.mutex.Lock()
		if rec, exist := dm.recs[recstr]; exist {
			if !rec.finished {
				rec.node = append(rec.node, n)
			}
		} else {
			dm.recs[recstr] = &targetRec{
				node:  []*node.Node{n},
				stamp: stamp,
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

//Finished set records n is downloading as finished.
func (dm *DownloadManager) Finished(n *node.Node, success bool) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	finished := true
	for _, rec := range dm.recs {
		if rec.downloading != nil && rec.downloading.Equals(n) {
			if success {
				rec.finished = true
			} else {
				rec.count++
				rec.downloading = nil
			}
		}
		if !rec.finished {
			finished = false
		}
	}
	if finished {
		log.Println(dm.datfile, ":finished downloading")
		delete(managers, dm.datfile)
	}
}
