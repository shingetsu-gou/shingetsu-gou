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
	"os"
	"path"
	"regexp"
	"sync"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//Caches is a slice of *cache
type Caches []*Cache

//has return true is Caches has cache cc
func (c Caches) Has(cc *Cache) bool {
	for _, c := range c {
		if c.Datfile == cc.Datfile {
			return true
		}
	}
	return false
}

//Len returns size of cache slice.
func (c Caches) Len() int {
	return len(c)
}

//Swap swaps order of cache slice.
func (c Caches) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

//sortByRecentStamp is for sorting by recentStamp.
type SortByRecentStamp struct {
	Caches
}

//Less returns true if cache[i].recentStamp < cache[j].recentStamp.
func (c SortByRecentStamp) Less(i, j int) bool {
	return c.Caches[i].RecentStamp() < c.Caches[j].RecentStamp()
}

//sortByStamp is for sorting by stamp.
type SortByStamp struct {
	Caches
	stamp []int64
}

func NewSortByStamp(cs Caches) SortByStamp {
	s := SortByStamp{
		Caches: cs,
		stamp:  make([]int64, cs.Len()),
	}
	for i, v := range cs {
		s.stamp[i] = v.ReadInfo().Stamp
	}
	return s
}

//Less returns true if cache[i].stamp < cache[j].stamp.
func (c SortByStamp) Less(i, j int) bool {
	return c.stamp[i] < c.stamp[j]
}

//sortByVelocity is for sorting by velocity.
type SortByVelocity struct {
	Caches
	velocity []int
	size     []int64
}

func NewSortByVelocity(cs Caches) SortByVelocity {
	s := SortByVelocity{
		Caches:   cs,
		velocity: make([]int, cs.Len()),
		size:     make([]int64, cs.Len()),
	}
	for i, v := range cs {
		f := v.ReadInfo()
		s.velocity[i] = f.Velocity
		s.size[i] = f.Size
	}
	return s
}

//Less returns true if cache[i].velocity < cache[j].velocity.
//if velocity[i]==velocity[j],  returns true if cache[i].size< cache[j].size.
func (c SortByVelocity) Less(i, j int) bool {
	if c.velocity[i] != c.velocity[j] {
		return c.velocity[i] < c.velocity[j]
	}
	return c.size[i] < c.size[j]
}

var CacheListCfg *CacheListConfig

type CacheListConfig struct {
	SaveSize    int
	SaveRemoved int64
	CacheDir    string
	SaveRecord  int64
	Fmutex      *sync.RWMutex
}

//cacheList is slice of *cache
type cacheList struct {
	Caches Caches
	*CacheListConfig
}

//newCacheList loads all Caches in disk and returns cachelist obj.
func NewCacheList() *cacheList {
	c := &cacheList{
		CacheListConfig: CacheListCfg,
	}
	c.load()
	return c
}

//append adds cache cc to list.
func (c *cacheList) Append(cc *Cache) {
	c.Caches = append(c.Caches, cc)
}

//Len returns # of Caches
func (c *cacheList) Len() int {
	return len(c.Caches)
}

//Swap swaps cache order.
func (c *cacheList) Swap(i, j int) {
	c.Caches[i], c.Caches[j] = c.Caches[j], c.Caches[i]
}

//locad loads all Caches in disk
func (c *cacheList) load() {
	if c.Caches != nil {
		c.Caches = c.Caches[:0]
	}
	err := util.EachFiles(c.CacheDir, func(f os.FileInfo) error {
		cc := NewCache(f.Name())
		c.Caches = append(c.Caches, cc)
		return nil
	})
	//only implements "asis"
	if err != nil {
		log.Println(err)
	}
}

//getall reload all records in cache in cachelist from network,
//and reset params.
func (c *cacheList) Getall() {
	const clientTimeout = 30 * time.Minute // Seconds; client_timeout < sync_cycle

	timelimit := time.Now().Add(clientTimeout)
	util.Shuffle(c)
	for _, ca := range c.Caches {
		now := time.Now()
		if now.After(timelimit) {
			log.Println("client timeout")
			return
		}
		ca.GetCache()
		ca.checkAttach()
	}
}

//search reloads records in Caches in cachelist
//and returns slice of cache which matches query.
func (c *cacheList) Search(query *regexp.Regexp) Caches {
	var result []*Cache
	for _, ca := range c.Caches {
		recs := ca.LoadRecords()
		for _, rec := range recs {
			err := rec.Load()
			if err != nil {
				log.Println(err)
			}
			if query.MatchString(rec.Recstr()) {
				result = append(result, ca)
				break
			}
		}
	}
	return result
}

//cleanRecords remove old or duplicates records for each Caches.
func (c *cacheList) CleanRecords() {
	c.Fmutex.Lock()
	defer c.Fmutex.Unlock()
	for _, ca := range c.Caches {
		recs := ca.LoadRecords()
		recs.removeRecords(c.SaveRecord, c.SaveSize)
	}
}

//removeRemoved removes removed files if old.
func (c *cacheList) RemoveRemoved() {
	for _, ca := range c.Caches {
		r := path.Join(ca.Datfile, "removed")
		if !util.IsDir(r) {
			continue
		}
		err := util.EachFiles(r, func(f os.FileInfo) error {
			rec := NewRecord(ca.Datfile, f.Name())
			if c.SaveRemoved > 0 && rec.Stamp+c.SaveRemoved < time.Now().Unix() &&
				rec.Stamp < ca.ReadInfo().Stamp {
				err := os.Remove(path.Join(ca.datpath(), "removed", f.Name()))
				if err != nil {
					log.Println(err)
				}
			}
			return nil
		})
		if err != nil {
			log.Println(err)
		}
	}
}
