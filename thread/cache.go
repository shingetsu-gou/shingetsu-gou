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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/node"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

var (
	cacheMap = make(map[string]sync.Pool)
	errSpam  = errors.New("this is spam")
	errGet   = errors.New("cannot get data")
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
	p, exist := cacheMap[datfile]
	if !exist {
		p.New = func() interface{} {
			c := &Cache{
				Datfile:     datfile,
				CacheConfig: CacheCfg,
			}
			c.tags = loadTagslice(path.Join(c.Datpath(), "tag.txt"))
			return c
		}
	}
	c := p.Get().(*Cache)
	p.Put(c)
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

//LoadRecords loads and returns record maps from the disk .
func (c *Cache) LoadRecords() RecordMap {
	r := path.Join(c.Datpath(), "record")
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

//checkData makes records from res and checks its records meets condisions of args.
//adds the rec to cache if meets conditions.
//if spam or big data, remove the rec from disk.
//returns count of added records to the cache and spam/getting error.
func (c *Cache) checkData(res []string, stamp int64, id string, begin, end int64) (int, error) {
	var err error
	count := 0
	for _, i := range res {
		r := NewRecord(c.Datfile, "")
		if r.Exists() {
			continue
		}
		if errr := r.parse(i); errr != nil {
			err = errGet
			continue
		}
		r.Sync()
		err = r.checkData(begin, end)
		if err == nil {
			count++
		}
	}
	return count, err
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

//getWithRange gets records with range using node n and adds to cache after checking them.
//if no records exist in cache, uses head
//return true if gotten records>0
func (c *Cache) getWithRange(n *node.Node) bool {
	now := time.Now().Unix()
	begin := now - c.GetRange
	if c.GetRange == 0 {
		begin = 0
	}
	res, err := n.Talk(fmt.Sprintf("/get/%s/%d-", c.Datfile, begin))
	if err != nil {
		return false
	}
	count, err := c.checkData(res, -1, "", begin, now)
	if err == nil || count > 0 {
		log.Println(c.Datfile, count, "records were saved")
	}
	return count > 0
}

//GetCache checks  nodes in lookuptable have the cache.
//if found gets records.
func (c *Cache) GetCache() bool {
	return c.NodeManager.EachNodes(c.Datfile, nil, c.getWithRange)
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
