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
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type cache struct {
	node        *rawNodeList
	datfile     string
	datpath     string
	size        int
	count       int
	velocity    int
	loaded      bool
	typee       string
	tags        *tagList
	dathash     string
	validStamp  int64
	recentStamp int64
	stamp       int64
	removed     map[string]string
	sugtags     *suggestedTagList
	saveRecord  int
	saveSize    int
	getRange    int
	syncRange   int
	saveRemoved int
	recs        map[string]*record
}

func newCache(datfile string) *cache {
	c := &cache{
		datfile: datfile,
		dathash: fileHash(datfile),
		removed: make(map[string]string),
		recs:    make(map[string]*record),
	}
	c.datpath = cacheDir + c.dathash
	c.stamp = c.loadStatus("stamp")
	c.recentStamp = c.stamp
	c.validStamp = c.loadStatus("validstamp")
	c.size = int(c.loadStatus("size"))
	c.count = int(c.loadStatus("count"))
	c.velocity = int(c.loadStatus("velocity"))
	c.node = newRawNodeList(path.Join(c.datpath, "node.txt"))
	c.tags = newTagList(path.Join(c.datfile, "tag.txt"))
	if v, exist := suggestedTagTable.sugtaglist[c.datfile]; exist {
		c.sugtags = v
	} else {
		c.sugtags = newSuggestedTagList(nil)
	}

	for _, t := range types {
		if strings.HasPrefix(c.datfile, t) {
			c.typee = t
			break
		}
	}

	c.saveRecord = saveRecord[c.typee]
	c.saveSize = savesize[c.typee]
	c.getRange = getRange[c.typee]
	c.syncRange = syncRange[c.typee]
	c.saveRemoved = saveRemoved[c.typee]

	if c.syncRange == 0 {
		c.saveRecord = 0
	} else if c.saveRemoved != 0 {
		if c.saveRemoved <= c.syncRange {
			c.saveRemoved = c.syncRange + 1
		}
	}
	return c
}
func (c *cache) Len() int {
	return len(c.recs)
}

func (c *cache) get(i string, def *record) *record {
	if v, exist := c.recs[i]; exist {
		return v
	}
	return def
}

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

func (c *cache) load() {
	if !c.loaded && isDir(c.datpath) {
		c.loaded = true
	}
	err := eachFiles(c.datpath, func(dir os.FileInfo) error {
		c.recs[c.datfile] = newRecord(c.datfile, dir.Name())
		return nil
	})
	if err != nil {
		log.Println(err, c.datpath)
	}
}

func (c *cache) hasRecord() bool {
	removed := path.Join(c.datpath, "removed")
	d, err := ioutil.ReadDir(removed)
	return len(c.recs) > 0 || (err != nil && len(d) > 0)
}

func (c *cache) loadStatus(key string) int64 {
	p := path.Join(c.datpath, key+".stat")
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

func (c *cache) saveStatus(key string, val interface{}) {
	p := path.Join(c.datpath, key+".stat")
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

func (c *cache) syncStatus() {
	c.saveStatus("stamp", c.stamp)
	c.saveStatus("validstamp", c.validStamp)
	c.saveStatus("size", c.size)
	c.saveStatus("count", c.count)
	c.saveStatus("velocity", c.velocity)
	if !isFile(c.datpath + "/dat.stat") {
		c.saveStatus("dat", c.datfile)
	}
}

func (c *cache) setupDirectories() {
	for _, d := range []string{"", "/attach", "/body", "/record", "/removed"} {
		di := c.datpath + d
		if !isDir(di) {
			err := os.Mkdir(di, 0666)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}


func (c *cache) checkData(res []string, stamp int64, id string, begin, end int64) (int, error) {
	var err error
	count := 0
	var i string
	flagGot := false
	for count, i = range res {
		r := newRecord(c.datfile, "")
		if r.parse(i) == nil &&
			(stamp != 0 || r.content["stamp"] == strconv.FormatInt(stamp, 10)) &&
			(id != "" || r.content["id"] == id) &&
			begin <= r.stamp && r.stamp <= end && r.md5check() {
			flagGot = true
			if len(i) > recordLimit*1024 || spamCheck(i) {
				log.Println("warning:", c.datfile, "/", r.idstr, ": too large or spamn record")
				c.addData(r, false)
				errr := r.remove()
				if errr != nil {
					log.Println(errr)
				}
				err = errSpam
			} else {
				c.addData(r, true)
			}
		} else {
			var strStamp string
			if stamp >= 0 {
				strStamp = "/" + strconv.FormatInt(stamp, 10)
			} else {
				if v, exist := r.content["stamp"]; exist {
					strStamp = "/" + v
				}
			}
			log.Println("warning:", c.datfile, strStamp, ":broken record")
		}
	}
	if err == nil && !flagGot {
		err = errGet
	}
	return count + 1, err
}

func (c *cache) getData(stamp int64, id string, n *node) error {
	res, err := n.talk("/get/" + c.datfile + "/" + strconv.FormatInt(stamp, 10) + "/" + id)
	if err != nil {
		log.Println(err)
		return errGet
	}
	count, err := c.checkData(res, stamp, id, -1, -1)
	if count > 0 {
		c.syncStatus()
	} else {
		log.Println(c.datfile, stamp, "records not found")
	}
	return err
}
func (c *cache) addData(rec *record, really bool) {
	c.setupDirectories()
	rec.sync(false)
	if really {
		c.recs[rec.idstr] = rec
		c.size += len(rec.idstr) + 1
		c.count++
		c.velocity++
		if c.validStamp < rec.stamp {
			c.validStamp = rec.stamp
		}
	}
	if c.stamp < rec.stamp {
		c.stamp = rec.stamp
	}
}

func (c *cache) getWithRange(n *node) bool {
	var err error
	oldcount := len(c.recs)
	now := time.Now().Unix()
	var begin, begin2 int64
	if c.stamp > 0 {
		begin = c.stamp
	}
	if c.syncRange > 0 {
		begin2 = now - int64(c.syncRange)
	}
	if begin2 < 0 {
		begin2 = 0
	} else {
		if begin2 < begin {
			begin = begin2
		}
	}
	var res []string
	if begin == 0 && len(c.recs) == 0 {
		if c.getRange > 0 {
			if begin = now - int64(c.getRange); begin < 0 {
				begin = 0
			}
		} else {
			begin = 0
		}
		res, err = n.talk("/get/" + c.datfile + "/" + strconv.FormatInt(begin, 10) + "-")
	} else {
		var head []string
		head, err = n.talk("/head/" + c.datfile + "/" + strconv.FormatInt(begin, 10) + "-")
		res = getRecords(c.datfile, n, head)
	}
	if err != nil {
		return false
	}
	count, err := c.checkData(res, -1, "", begin, now)
	if err == nil || count > 0 {
		c.syncStatus()
		if oldcount == 0 {
			c.loaded = true
		}
	}
	return count > 0
}
func (c *cache) checkBody() {
	dir := path.Join(cacheDir, c.dathash, "body")
	err := eachFiles(dir, func(d os.FileInfo) error {
		rec := newRecord(c.datfile, d.Name())
		if !isFile(rec.path) {
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

func (c *cache) checkAttach() {
	dir := path.Join(cacheDir, c.dathash, "attach")
	err := eachFiles(dir, func(d os.FileInfo) error {
		idstr := d.Name()
		if i := strings.IndexRune(idstr, '.'); i > 0 {
			idstr = idstr[:i]
		}
		if strings.HasPrefix(idstr, "s") {
			idstr = idstr[1:]
		}
		rec := newRecord(c.datfile, idstr)
		if !isFile(rec.path) {
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

func (c *cache) remove() {
	err := os.RemoveAll(c.datpath)
	if err != nil {
		log.Println(err)
	}
}

func (c *cache) removeRecords(now int64, limit int64) {
	ids := c.keys()
	if c.saveSize < len(ids) {
		ids = ids[:len(ids)-1-c.saveSize]
		if limit > 0 {
			for _, r := range ids {
				rec := c.recs[r]
				if rec.stamp+limit < time.Now().Unix() {
					err := rec.remove()
					if err != nil {
						log.Println(err)
					}
					delete(c.recs, r)
					c.count--
				}
			}
		}
	}
	once := make(map[string]struct{})
	for r, rec := range c.recs {
		if !isFile(rec.path) {
			if _, exist := once[rec.id]; exist {
				err := rec.remove()
				if err != nil {
					log.Println(err)
				}
				delete(c.recs, r)
				c.count--
			} else {
				once[rec.id] = struct{}{}
			}
		}
	}
	c.syncStatus()
}
func (c *cache) exists() bool {
	return isDir(c.datpath)
}

func (c *cache) search(myself *node) bool {
	c.setupDirectories()
	if myself != nil {
		myself = nodeList.myself()
	}
	n := searchList.search(nil, myself, lookupTable.get(c.datfile, nil))
	if n != nil {
		if !nodeList.hasNode(n) {
			nodeList.append(n)
			nodeList.sync()
		}
		c.getWithRange(n)
		if !c.node.hasNode(n) {
			for c.node.Len() >= shareNodes {
				n = c.node.random()
				c.node.removeNode(n)
			}
			c.node.append(n)
			c.node.sync()
		}
		return true
	}
	c.syncStatus()
	return false
}

type cacheList struct {
	caches []*cache
}

func newCacheList() *cacheList {
	c := &cacheList{}
	c.load()
	return c
}

func (c *cacheList) append(cc *cache) {
	c.caches = append(c.caches, cc)
}

func (c *cacheList) Len() int {
	return len(c.caches)
}

func (c *cacheList) Swap(i, j int) {
	c.caches[i], c.caches[j] = c.caches[j], c.caches[i]
}

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

func (c *cacheList) getall(timelimit time.Time) {
	now := time.Now()
	shuffle(c)
	my := nodeList.myself()
	for _, ca := range c.caches {
		if now.After(timelimit) {
			log.Println("client timeout")
			return
		}
		if !ca.exists() {
			return
		}
		ca.search(my)
		ca.size = 0
		ca.count = 0
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
				ca.size += rec.Len()
				ca.count++
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

type caches []*cache

func (c caches) has(cc *cache) bool {
	for _, c := range c {
		if c.datfile == cc.datfile {
			return true
		}
	}
	return false
}

func (c caches) Len() int {
	return len(c)
}
func (c caches) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

type sortByRecentStamp struct {
	caches
}

func (c sortByRecentStamp) Less(i, j int) bool {
	return c.caches[i].recentStamp < c.caches[j].recentStamp
}

type sortByValidStamp struct {
	caches
}

func (c sortByValidStamp) Less(i, j int) bool {
	return c.caches[i].validStamp < c.caches[j].validStamp
}

type sortByVelocity struct {
	caches
}

func (c sortByVelocity) Less(i, j int) bool {
	if c.caches[i].velocity != c.caches[j].velocity {
		return c.caches[i].velocity < c.caches[j].velocity
	}
	return c.caches[i].count < c.caches[j].count
}

func (c *cacheList) search(query *regexp.Regexp) caches {
	var result []*cache
	for _, ca := range c.caches {
		for _, rec := range ca.recs {
			err := rec.loadBody()
			if err != nil {
				log.Println(err)
			}
			if query.MatchString(rec.recstr) {
				result = append(result, ca)
				break
			}
		}
	}
	return result
}

func (c *cacheList) cleanRecords() {
	for _, ca := range c.caches {
		ca.removeRecords(time.Now().Unix(), int64(ca.saveRecord))
	}
}

func (c *cacheList) removeRemoved() {
	for _, ca := range c.caches {
		err := eachFiles(path.Join(ca.datfile, "removed"), func(f os.FileInfo) error {
			rec := newRecord(ca.datfile, f.Name())
			if ca.saveRemoved > 0 && rec.stamp+int64(ca.saveRemoved) < time.Now().Unix() &&
				rec.stamp < ca.stamp {
				err := os.Remove(path.Join(ca.datpath, "removed", f.Name()))
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
