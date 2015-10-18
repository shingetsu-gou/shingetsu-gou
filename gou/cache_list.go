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
	"os"
	"path"
	"regexp"
	"time"
)

//cacheList is slice of *cache
type cacheList struct {
	Caches []*cache
}

//newCacheList loads all caches in disk and returns cachelist obj.
func newCacheList() *cacheList {
	c := &cacheList{}
	c.load()
	return c
}

//append adds cache cc to list.
func (c *cacheList) append(cc *cache) {
	c.Caches = append(c.Caches, cc)
}

//Len returns # of caches
func (c *cacheList) Len() int {
	return len(c.Caches)
}

//Swap swaps cache order.
func (c *cacheList) Swap(i, j int) {
	c.Caches[i], c.Caches[j] = c.Caches[j], c.Caches[i]
}

//locad loads all caches in disk
func (c *cacheList) load() {
	if c.Caches != nil {
		c.Caches = c.Caches[:0]
	}
	err := eachFiles(cacheDir, func(f os.FileInfo) error {
		cc := newCache(f.Name())
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
func (c *cacheList) getall(timelimit time.Time) {
	shuffle(c)
	for _, ca := range c.Caches {
		now := time.Now()
		if now.After(timelimit) {
			log.Println("client timeout")
			return
		}
		ca.updateFromRecords()
		ca.checkBody()
		ca.checkAttach()
		ca.syncStatus()
	}
}

//caches is a slice of *cache
type caches []*cache

//has return true is caches has cache cc
func (c caches) has(cc *cache) bool {
	for _, c := range c {
		if c.Datfile == cc.Datfile {
			return true
		}
	}
	return false
}

//Len returns size of cache slice.
func (c caches) Len() int {
	return len(c)
}

//Swap swaps order of cache slice.
func (c caches) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

//sortByRecentStamp is for sorting by recentStamp.
type sortByRecentStamp struct {
	caches
}

//Less returns true if cache[i].recentStamp < cache[j].recentStamp.
func (c sortByRecentStamp) Less(i, j int) bool {
	return c.caches[i].RecentStamp < c.caches[j].RecentStamp
}

//sortByStamp is for sorting by stamp.
type sortByStamp struct {
	caches
}

//Less returns true if cache[i].stamp < cache[j].stamp.
func (c sortByStamp) Less(i, j int) bool {
	return c.caches[i].stamp < c.caches[j].stamp
}

//sortByValidStamp is for sorting by validStamp.
type sortByValidStamp struct {
	caches
}

//Less returns true if cache[i].validStamp < cache[j].validStamp.
func (c sortByValidStamp) Less(i, j int) bool {
	return c.caches[i].ValidStamp < c.caches[j].ValidStamp
}

//sortByVelocity is for sorting by velocity.
type sortByVelocity struct {
	caches
}

//Less returns true if cache[i].velocity < cache[j].velocity.
//if velocity[i]==velocity[j],  returns true if cache[i].size< cache[j].size.
func (c sortByVelocity) Less(i, j int) bool {
	if c.caches[i].velocity != c.caches[j].velocity {
		return c.caches[i].velocity < c.caches[j].velocity
	}
	return c.caches[i].Len() < c.caches[j].Len()
}

//search reloads records in caches in cachelist
//and returns slice of cache which matches query.
func (c *cacheList) search(query *regexp.Regexp) caches {
	var result []*cache
	for _, ca := range c.Caches {
		for _, rec := range ca.recs {
			err := rec.loadBody()
			if err != nil {
				log.Println(err)
			}
			if query.MatchString(rec.recstr()) {
				result = append(result, ca)
				break
			}
		}
	}
	return result
}

//cleanRecords remove old or duplicates records for each caches.
func (c *cacheList) cleanRecords() {
	for _, ca := range c.Caches {
		ca.removeRecords(ca.saveRecord())
	}
}

//removeRemoved removes removed files if old.
func (c *cacheList) removeRemoved() {
	for _, ca := range c.Caches {
		r := path.Join(ca.Datfile, "removed")
		if !IsDir(r) {
			continue
		}
		err := eachFiles(r, func(f os.FileInfo) error {
			rec := newRecord(ca.Datfile, f.Name())
			if ca.saveRemoved() > 0 && rec.Stamp+ca.saveRemoved() < time.Now().Unix() &&
				rec.Stamp < ca.stamp {
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
