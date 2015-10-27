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
)

//cache represents cache of one file.
type cache struct {
	Datfile     string
	velocity    int      // records count per unit time
	Typee       string   //"thread"
	tags        tagslice //made by the user
	ValidStamp  int64    //last record stamp excpet spam
	RecentStamp int64    //when got by "/recent"
	stamp       int64    //last record stamp including spam
	recs        map[string]*record
	loaded      bool // loaded records
	isDirty     bool
	mutex       sync.RWMutex
}

//saveRecord returns max # of records to be saved.
func (c *cache) saveRecord() int64 {
	if c.syncRange() == 0 {
		return 0
	}
	return saveRecord[c.Typee]
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

//saveSize returns # of records to be holded.
func (c *cache) saveSize() int {
	return savesize[c.Typee]
}

//getRange returns default time range when getting records.
func (c *cache) getRange() int64 {
	return getRange[c.Typee]
}

//syncRange returns default time range when syncing(using head) records.
func (c *cache) syncRange() int64 {
	return syncRange[c.Typee]
}

//saveRemoved returns default time range removed mark is alive in disk.
func (c *cache) saveRemoved() int64 {
	if saveRemoved[c.Typee] != 0 && saveRemoved[c.Typee] <= c.syncRange() {
		return c.syncRange() + 1
	}
	return saveRemoved[c.Typee]
}

//dathash returns datfile itself is type=asis.
func (c *cache) dathash() string {
	return fileHash(c.Datfile)
}

//datpath returns real file path of this cache.
func (c *cache) datpath() string {
	return path.Join(cacheDir, c.dathash())
}

//newCache read files to set params and returns cache obj.
func newCache(datfile string) *cache {
	if c, exist := cacheMap[datfile]; exist {
		return c
	}
	c := &cache{
		Datfile: datfile,
		recs:    make(map[string]*record),
	}
	c.stamp = c.loadStatus("stamp")
	c.RecentStamp = c.stamp
	c.ValidStamp = c.loadStatus("validstamp")
	c.velocity = int(c.loadStatus("velocity"))
	c.tags = loadTagslice(path.Join(c.datpath(), "tag.txt"))
	for _, t := range types {
		if strings.HasPrefix(c.Datfile, t) {
			c.Typee = t
			break
		}
	}
	cacheMap[datfile] = c
	return c
}

//close removes myself from caches.
func (c *cache) close() {
	delete(cacheMap, c.Datfile)
}

//len returns size of records on disk.
func (c *cache) Len() int {
	l, _ := c.lenSize()
	return l
}

//Size returns sum of records size on disk.
func (c *cache) Size() int {
	_, l := c.lenSize()
	return l
}

//lenSize reads records from disk and returns # and total size.
func (c *cache) lenSize() (int, int) {
	var size int64
	var len int
	d := path.Join(c.datpath(), "record")
	if !IsDir(d) {
		return 0, 0
	}
	err := eachFiles(d, func(dir os.FileInfo) error {
		size += dir.Size()
		len++
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return len, int(size)
}

//get returns records which hav key=i.
//return def if not found.
func (c *cache) get(i string, def *record) *record {
	if v, exist := c.recs[i]; exist {
		return v
	}
	return def
}

//keys returns key strings(ids) of records
func (c *cache) keys() []string {
	c.load()
	r := make([]string, len(c.recs))
	i := 0
	for k := range c.recs {
		r[i] = k
		i++
	}
	sort.Strings(r)
	return r
}

//load loads records from files on the disk if not loaded.
func (c *cache) load() {
	r := path.Join(c.datpath(), "record")
	if !IsDir(r) {
		return
	}
	if c.loaded && !c.Exists() {
		return
	}
	c.loaded = true
	err := eachFiles(r, func(f os.FileInfo) error {
		c.recs[f.Name()] = newRecord(c.Datfile, f.Name())
		return nil
	})
	if err != nil {
		log.Println(err, c.datpath())
	}
}

//hasRecord return true if  cache has more than one records or removed records.
func (c *cache) hasRecord() bool {
	removed := path.Join(c.datpath(), "removed")
	d, err := ioutil.ReadDir(removed)
	return len(c.recs) > 0 || (err == nil && len(d) > 0)
}

//loadStatus load int value from the file on disk.
func (c *cache) loadStatus(key string) int64 {
	if !IsDir(c.datpath()) {
		return 0
	}
	p := path.Join(c.datpath(), key+".stat")
	f, err := ioutil.ReadFile(p)
	if err != nil {
		log.Println(err)
		return 0
	}
	r, err := strconv.ParseInt(strings.Trim(string(f), "\n\r"), 10, 64)
	if err != nil {
		log.Println(err)
		return 0
	}
	return r
}

//saveStatus convert vals to strings and files.
func (c *cache) saveStatus(key string, val interface{}) {
	p := path.Join(c.datpath(), key+".stat")
	var err error
	switch v := val.(type) {
	case int:
		err = ioutil.WriteFile(p, []byte(strconv.Itoa(v)+"\n"), 0755)
	case int64:
		err = ioutil.WriteFile(p, []byte(strconv.FormatInt(v, 10)+"\n"), 0755)
	case string:
		err = ioutil.WriteFile(p, []byte(v+"\n"), 0755)
	default:
		err = errors.New("unknown format")
	}
	if err != nil {
		log.Println(err)
	}
}

//syncStatus saves params to files.
func (c *cache) syncStatus() {
	fmutex.Lock()
	defer fmutex.RUnlock()
	c.mutex.RLock()
	defer c.mutex.RLock()
	c.saveStatus("stamp", c.stamp)
	c.saveStatus("validstamp", c.ValidStamp)
	c.saveStatus("velocity", c.velocity)
	if !IsFile(c.datpath() + "/dat.stat") {
		c.saveStatus("dat", c.Datfile)
	}
	c.tags.sync(path.Join(c.datpath(), "tag.txt"))
}

//setupDirectories make necessary dirs.
func (c *cache) setupDirectories() {
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
				c.addData(r)
				errr := r.remove()
				if errr != nil {
					log.Println(errr)
				}
			} else {
				c.addData(r)
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
	if count > 0 {
		c.syncStatus()
	} else {
		log.Println(c.Datfile, stamp, "records not found")
	}
	return err
}

//addData adds rec to cache.
func (c *cache) addData(rec *record) {
	c.recs[rec.Idstr()] = rec
	c.velocity++
	c.updateStamp(rec)
	if c.ValidStamp < rec.Stamp {
		c.ValidStamp = rec.Stamp
	}
	rec.sync()
}

//updateStamp updates cache's stamp to rec if rec is newer.
func (c *cache) updateStamp(rec *record) {
	rec.sync()
	if c.stamp < rec.Stamp {
		c.stamp = rec.Stamp
	}
}

//getWithRange gets records with range using node n and adds to cache after checking them.
//if no records exist in cache, uses head
//return true if gotten records>0
func (c *cache) getWithRange(n *node) bool {
	var err error
	oldcount := len(c.recs)
	now := time.Now().Unix()

	begin := c.stamp
	begin2 := now - c.syncRange()

	if begin2 < begin {
		begin = begin2
	}
	var res []string
	if begin == 0 && len(c.recs) == 0 {
		begin = now - c.getRange()
		res, err = n.talk(fmt.Sprintf("/get/%s/%d-", c.Datfile, begin))
	} else {
		var head []string
		head, err = n.talk(fmt.Sprintf("/head/%s/%d-", c.Datfile, begin))
		res = getRecords(c.Datfile, n, head)
	}
	if err != nil {
		return false
	}
	count, err := c.checkData(res, -1, "", begin, now)
	if err == nil || count > 0 {
		log.Println(c.ValidStamp, "saved")
		c.syncStatus()
		if oldcount == 0 {
			c.loaded = true
		}
	}
	return count > 0
}

//checkBody checks body files in the disk.
//if no record files in record dir, rm the body file.
func (c *cache) checkBody() {
	dir := path.Join(cacheDir, c.dathash(), "body")
	err := eachFiles(dir, func(d os.FileInfo) error {
		rec := newRecord(c.Datfile, d.Name())
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

//checkAttach checks files attach dir and if corresponding records
//don't exist in record dir, removes the attached file.
func (c *cache) checkAttach() {
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
	err := os.RemoveAll(c.datpath())
	if err != nil {
		log.Println(err)
	}
}

//removeRecords remove old records while remaing #saveSize records.
//and also removes duplicates recs.
func (c *cache) removeRecords(limit int64) {
	ids := c.keys()
	if c.saveSize() < len(ids) {
		ids = ids[:len(ids)-c.saveSize()]
		if limit > 0 {
			for _, r := range ids {
				rec := c.recs[r]
				if rec.Stamp+limit < time.Now().Unix() {
					err := rec.remove()
					if err != nil {
						log.Println(err)
					}
					delete(c.recs, r)
				}
			}
		}
	}
	once := make(map[string]struct{})
	for r, rec := range c.recs {
		if !IsFile(rec.path()) {
			if _, exist := once[rec.ID]; exist {
				err := rec.remove()
				if err != nil {
					log.Println(err)
				}
				delete(c.recs, r)
			} else {
				once[rec.ID] = struct{}{}
			}
		}
	}
	c.syncStatus()
}

//exists return true is datapath exists.
func (c *cache) Exists() bool {
	return IsDir(c.datpath())
}

//search checks  nodes in lookuptable have the cache.
//if found adds to nodelist ,get records , and adds to nodes in cache.
func (c *cache) search(myself *node) bool {
	if myself != nil {
		myself = nodeManager.myself()
	}
	n := nodeManager.search(c, myself, nodeManager.get(c.Datfile, nil))
	if n != nil {
		c.getWithRange(n)
		return true
	}
	return false
}

//updateFromRecords reload all records in cache from network,
//and reset params.
func (c *cache) updateFromRecords() {
	my := nodeManager.myself()
	if !c.Exists() {
		return
	}
	c.search(my)
	c.velocity = 0
	c.ValidStamp = 0
	for _, rec := range c.recs {
		if !rec.Exists() {
			continue
		}
		if err := rec.load(); err != nil {
			log.Println(err)
			err := rec.remove()
			if err != nil {
				log.Println(err)
			}
			continue
		}
		if c.stamp < rec.Stamp {
			c.stamp = rec.Stamp
		}
		if c.ValidStamp < rec.Stamp {
			c.ValidStamp = rec.Stamp
		}
		if time.Now().Add(-7 * 24 * time.Hour).Before(time.Unix(rec.Stamp, 0)) {
			c.velocity++
		}
		rec.sync()
	}
}
