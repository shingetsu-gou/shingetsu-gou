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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

//cacheInfo represents size/len/velocity of cache.
type cacheInfo struct {
	size     int64 //size of total records
	len      int   //# of records
	velocity int   //# of new records in one day
	stamp    int64 //stamp of newest record
}

//cache represents cache of one file.
type cache struct {
	Datfile string
	tags    tagslice //made by the user
	mutex   sync.RWMutex
}

//addTags add user tag list from vals.
func (c *cache) addTags(vals []string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tags.addString(vals)
	utag.setDirty()
}

//setTags set user tag list from vals.
func (c *cache) setTags(vals []string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tags = newTagslice(vals)
	utag.setDirty()
}

//hasTagstr returns true if tag has tagstr.
func (c *cache) hasTagstr(tagstr string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.tags.hasTagstr(tagstr)
}

//hasTab returns true if cache has tagstr=board tag in usertag or sugtag.
func (c *cache) hasTag(board string) bool {
	if suggestedTagTable.hasTagstr(c.Datfile, board) {
		return true
	}
	return c.hasTagstr(board)
}

//saveRemoved returns default time range removed mark is alive in disk.
func (c *cache) saveRemoved() int64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if saveRemoved != 0 && saveRemoved <= syncRange {
		return syncRange + 1
	}
	return saveRemoved
}

//dathash returns datfile itself is type=asis.
func (c *cache) dathash() string {
	return fileHash(c.Datfile)
}

//datpath returns real file path of this cache.
func (c *cache) datpath() string {
	return path.Join(cacheDir, c.dathash())
}

//recentStamp  returns time of getting by /recent.
func (c *cache) recentStamp() int64 {
	n := recentList.newest(c.Datfile)
	if n == nil {
		return c.readInfo().stamp
	}
	return n.Stamp
}

//newCache read files to set params and returns cache obj.
func newCache(datfile string) *cache {
	c := &cache{
		Datfile: datfile,
	}
	c.tags = loadTagslice(path.Join(c.datpath(), "tag.txt"))
	return c
}

//readInfo reads cache info from disk and returns #,velocity, and total size.
func (c *cache) readInfo() *cacheInfo {
	fmutex.RLock()
	defer fmutex.RUnlock()
	d := path.Join(c.datpath(), "record")
	if !IsDir(d) {
		return nil
	}
	ci := &cacheInfo{}
	err := eachFiles(d, func(dir os.FileInfo) error {
		stamp, err := strconv.ParseInt(strings.Split(dir.Name(), "_")[0], 10, 64)
		if err != nil {
			log.Println(err)
			return nil
		}
		if ci.stamp < stamp {
			ci.stamp = stamp
		}
		if time.Unix(stamp, 0).After(time.Now().Add(-7 * 24 * time.Hour)) {
			ci.velocity++
		}
		ci.size += dir.Size()
		ci.len++
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return ci
}

//load loads and returns records from files on the disk .
func (c *cache) loadRecords() recordMap {
	fmutex.RLock()
	defer fmutex.RUnlock()
	r := path.Join(c.datpath(), "record")
	if !IsDir(r) {
		return nil
	}
	if !c.Exists() {
		return nil
	}
	recs := make(map[string]*record)
	err := eachFiles(r, func(f os.FileInfo) error {
		recs[f.Name()] = newRecord(c.Datfile, f.Name())
		return nil
	})
	if err != nil {
		log.Println(err, c.datpath())
	}
	return recordMap(recs)
}

//hasRecord return true if  cache has more than one records or removed records.
func (c *cache) hasRecord() bool {
	fmutex.RLock()
	defer fmutex.RUnlock()
	f, err := ioutil.ReadDir(path.Join(c.datpath(), "record"))
	if err != nil {
		return false
	}
	removed := path.Join(c.datpath(), "removed")
	d, err := ioutil.ReadDir(removed)
	return len(f) > 0 || (err == nil && len(d) > 0)
}

//syncStatus saves params to files.
func (c *cache) syncTag() {
	fmutex.Lock()
	defer fmutex.Unlock()
	c.tags.sync(path.Join(c.datpath(), "tag.txt"))
}

//setupDirectories make necessary dirs.
func (c *cache) setupDirectories() {
	fmutex.Lock()
	defer fmutex.Unlock()
	for _, d := range []string{"", "/attach", "/body", "/record", "/removed"} {
		di := path.Join(c.datpath(), d)
		if !IsDir(di) {
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
func (c *cache) checkData(res []string, stamp int64, id string, begin, end int64) (int, error) {
	var err error
	count := 0
	for _, i := range res {
		r := newRecord(c.Datfile, "")
		if er := r.parse(i); er == nil && r.meets(i, stamp, id, begin, end) {
			count++
			if len(i) > recordLimit*1024 || cachedRule.check(i) {
				err = errSpam
				log.Printf("warning:%s/%s:too large or spam record", c.Datfile, r.Idstr())
				r.sync()
				errr := r.remove()
				if errr != nil {
					log.Println(errr)
				}
			} else {
				r.sync()
			}
		} else {
			log.Println("warning::broken record", c.Datfile, i)
		}
	}
	if count == 0 {
		return 0, errGet
	}
	return count, err
}

//getData gets records from node n and checks its is same as stamp and id in args.
//save recs if success. returns errSpam or errGet.
func (c *cache) getData(stamp int64, id string, n *node) error {
	res, err := n.talk(fmt.Sprintf("/get/%s/%d/%s", c.Datfile, stamp, id))
	if err != nil {
		log.Println(err)
		return errGet
	}
	count, err := c.checkData(res, stamp, id, -1, -1)
	if count == 0 {
		log.Println(c.Datfile, stamp, "records not found")
	}
	return err
}

//getWithRange gets records with range using node n and adds to cache after checking them.
//if no records exist in cache, uses head
//return true if gotten records>0
func (c *cache) getWithRange(n *node) bool {
	now := time.Now().Unix()

	begin := c.readInfo().stamp
	begin2 := now - syncRange
	if begin2 < begin {
		begin = begin2
	}

	if !c.hasRecord() {
		begin = now - getRange
	}

	res, err := n.talk(fmt.Sprintf("/get/%s/%d-", c.Datfile, begin))
	if err != nil {
		return false
	}
	count, err := c.checkData(res, -1, "", begin, now)
	if err == nil || count > 0 {
		log.Println(c.Datfile, count, "records were saved")
	}
	return count > 0
}

//checkAttach checks files attach dir and if corresponding records
//don't exist in record dir, removes the attached file.
func (c *cache) checkAttach() {
	fmutex.Lock()
	defer fmutex.Unlock()
	dir := path.Join(cacheDir, c.dathash(), "attach")
	err := eachFiles(dir, func(d os.FileInfo) error {
		idstr := d.Name()
		if i := strings.IndexRune(idstr, '.'); i > 0 {
			idstr = idstr[:i]
		}
		if strings.HasPrefix(idstr, "s") {
			idstr = idstr[1:]
		}
		rec := newRecord(c.Datfile, idstr)
		if !IsFile(rec.path()) {
			err := os.Remove(path.Join(dir, d.Name()))
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

//remove removes all files and dirs of cache.
func (c *cache) remove() {
	fmutex.Lock()
	defer fmutex.Unlock()
	err := os.RemoveAll(c.datpath())
	if err != nil {
		log.Println(err)
	}
}

//exists return true is datapath exists.
func (c *cache) Exists() bool {
	fmutex.RLock()
	defer fmutex.RUnlock()
	return IsDir(c.datpath())
}

//search checks  nodes in lookuptable have the cache.
//if found adds to nodelist ,get records , and adds to nodes in cache.
func (c *cache) search(myself *node) bool {
	if myself == nil {
		myself = nodeManager.myself()
	}
	n := nodeManager.search(c, myself, nodeManager.get(c.Datfile, nil))
	if n != nil {
		c.getWithRange(n)
		return true
	}
	return false
}

