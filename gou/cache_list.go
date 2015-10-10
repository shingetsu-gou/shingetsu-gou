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
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

//cacheList is slice of *cache
type cacheList struct {
	caches []*cache
}

//newCacheList loads all caches in disk and returns cachelist obj.
func newCacheList() *cacheList {
	c := &cacheList{}
	c.load()
	return c
}

//append adds cache cc to list.
func (c *cacheList) append(cc *cache) {
	c.caches = append(c.caches, cc)
}

//Len returns # of caches
func (c *cacheList) Len() int {
	return len(c.caches)
}

//Swap swaps cache order.
func (c *cacheList) Swap(i, j int) {
	c.caches[i], c.caches[j] = c.caches[j], c.caches[i]
}

//locad loads all caches in disk
func (c *cacheList) load() {
	if c.caches != nil {
		c.caches = c.caches[:0]
	}
	err := eachFiles(cacheDir, func(f os.FileInfo) error {
		cc := newCache(f.Name())
		c.caches = append(c.caches, cc)
		return nil
	})
	//only implements "asis"
	if err != nil {
		log.Println(err)
	}
}

//rehash reads thread name from dat.stat (if not exists, creates it from dir name),
//and changes dir name to hashed name.
func (c *cacheList) rehash() {
	toreload := false
	err := eachFiles(cacheDir, func(f os.FileInfo) error {
		datStatFile := path.Join(cacheDir, f.Name(), "dat.stat")
		var datStat string
		if isFile(datStatFile) {
			datStatt, err := ioutil.ReadFile(datStatFile)
			if err != nil {
				log.Println("rehash err", err)
				return nil
			}
			datStat = string(datStatt)
			datStat = strings.Trim(strings.Split(string(datStat), "\n")[0], "\r\n")
		} else {
			datStat = f.Name()
			err := ioutil.WriteFile(datStatFile, []byte(datStat+"\n"), 0755)
			if err != nil {
				log.Println("rehash err", err)
				return nil
			}
		}
		hash := fileHash(datStat)
		if hash == f.Name() {
			return nil
		}
		log.Println("rehash", f.Name(), "to", hash)
		err := moveFile(path.Join(cacheDir, f.Name()), path.Join(cacheDir, hash))
		if err != nil {
			return err
		}
		toreload = true
		return nil
	})
	if err != nil {
		log.Println("rehash err", err)
	}
	if toreload {
		c.load()
	}
}

//getall reload all records in cache in cachelist from network,
//and reset params.
func (c *cacheList) getall(timelimit time.Time) {
	shuffle(c)
	my := nodeList.myself()
	for _, ca := range c.caches {
		now := time.Now()
		if now.After(timelimit) {
			log.Println("client timeout")
			return
		}
		if !ca.exists() {
			return
		}
		ca.search(my)
		ca.size = 0
		ca.velocity = 0
		ca.validStamp = 0
		for _, rec := range ca.recs {
			if !rec.exists() {
				continue
			}
			if rec.load() != nil {
				if ca.stamp < rec.stamp {
					ca.stamp = rec.stamp
				}
				if ca.validStamp < rec.stamp {
					ca.validStamp = rec.stamp
				}
				ca.size += rec.len()
				if now.Add(-7 * 24 * time.Hour).Before(time.Unix(rec.stamp, 0)) {
					ca.velocity++
				}
				rec.sync(false)
			}
		}
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
		if c.datfile == cc.datfile {
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
	return c.caches[i].recentStamp < c.caches[j].recentStamp
}

//sortByValidStamp is for sorting by validStamp.
type sortByValidStamp struct {
	caches
}

//Less returns true if cache[i].validStamp < cache[j].validStamp.
func (c sortByValidStamp) Less(i, j int) bool {
	return c.caches[i].validStamp < c.caches[j].validStamp
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
	return c.caches[i].len() < c.caches[j].len()
}

//search reloads records in caches in cachelist
//and returns slice of cache which matches query.
func (c *cacheList) search(query *regexp.Regexp) caches {
	var result []*cache
	for _, ca := range c.caches {
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
	for _, ca := range c.caches {
		ca.removeRecords(ca.saveRecord())
	}
}

//removeRemoved removes removed files if old.
func (c *cacheList) removeRemoved() {
	for _, ca := range c.caches {
		err := eachFiles(path.Join(ca.datfile, "removed"), func(f os.FileInfo) error {
			rec := newRecord(ca.datfile, f.Name())
			if ca.saveRemoved() > 0 && rec.stamp+ca.saveRemoved() < time.Now().Unix() &&
				rec.stamp < ca.stamp {
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
