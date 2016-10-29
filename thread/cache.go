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
	"strings"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/db"
	"github.com/shingetsu-gou/shingetsu-gou/recentlist"
	"github.com/shingetsu-gou/shingetsu-gou/record"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//Cache represents cache of one file.
type Cache struct {
	Datfile string
}

//NewCache read tag files to set and returns cache obj.
//it uses sync.pool to ensure that only one cache obj exists for one datfile.
//and garbage collected when not used.
func NewCache(datfile string) *Cache {
	c := &Cache{
		Datfile: datfile,
	}
	return c
}

//Stamp returns latest stampl of records in the cache.
func (c *Cache) Stamp() int64 {
	r, err := record.GetFromDBs(c.Datfile)
	if err != nil {
		log.Print(err)
		return 0
	}
	if len(r) == 0 {
		return 0
	}
	var stamp int64
	for i := len(r) - 1; i >= 0; i-- {
		if !r[i].Deleted {
			stamp = r[i].Stamp
			break
		}
	}
	return stamp
}

//Len returns # of records in the cache.
func (c *Cache) Len() int {
	r, err := record.GetFromDBs(c.Datfile)
	if err != nil {
		log.Print(err)
		return 0
	}
	return len(r)
}

//Velocity returns number of records in one days in the cache.
func (c *Cache) Velocity() int {
	r, err := record.GetFromDBs(c.Datfile)
	if err != nil {
		log.Print(err)
		return 0
	}
	cnt := 0
	t := int64(time.Now().Add(-7 * 24 * time.Hour).Second())
	for _, rr := range r {
		if rr.Stamp > t {
			cnt++
		}
	}
	return int(cnt)
}

//Size returns sum of body char length of records in the cache.
func (c *Cache) Size() int64 {
	if c.Len() == 0 {
		return 0
	}
	r, err := record.GetFromDBs(c.Datfile)
	if err != nil {
		log.Print(err)
		return 0
	}
	var cnt int64
	for _, rr := range r {
		cnt += int64(len(rr.Body))
	}
	return cnt
}

//LoadRecords loads and returns record maps from the disk..
func (c *Cache) LoadRecords(kind int) record.Map {
	m, err := record.FromRecordDB(c.Datfile, kind)
	if err != nil {
		log.Print(err)
		return nil
	}
	return m
}

//Subscribe add the thread to thread db.
func (c *Cache) Subscribe() {
	err := db.Put("thread", []byte(c.Datfile), []byte(""))
	if err != nil {
		log.Print(err)
	}
}

//CheckData makes a record from res and checks its records meets condisions of args.
//adds the rec to cache if meets conditions.
//if spam or big data, remove the rec from disk.
//returns count of added records to the cache and spam/getting error.
func (c *Cache) CheckData(res string, stamp int64, id string, begin, end int64) error {
	r := record.New(c.Datfile, "", 0)
	if errr := r.Parse(res); errr != nil {
		return cfg.ErrGet

	}
	if r.Exists() || r.Removed() {
		return nil
	}
	r.Sync()
	return r.CheckData(begin, end)
}

//Remove Remove all files and dirs of cache.
func (c *Cache) Remove() {
	r, err := record.GetFromDBs(c.Datfile)
	if err != nil {
		log.Print(err)
	}
	for _, rr := range r {
		rr.Del()
	}
	err = db.Del("thread", []byte(c.Datfile))
	if err != nil {
		log.Println(err)
	}
}

//HasRecord return true if  cache has more than one records or removed records.
func (c *Cache) HasRecord() bool {
	r, err := record.GetFromDBs(c.Datfile)
	if err != nil {
		log.Print(err)
		return false
	}
	for _, rr := range r {
		if !rr.Deleted {
			return true
		}
	}
	return false
}

//Exists return true is datapath exists.
func (c *Cache) Exists() bool {
	cnt, err := db.Count("thread", []byte(c.Datfile))
	if err != nil {
		log.Print(err)
		return false
	}
	return cnt > 0
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
	m, err := record.FromRecordDB(c.Datfile, record.Alive)
	if err != nil {
		log.Print(err)
		return nil
	}
	contents := make([]string, 0, 2)
	for _, tt := range m {
		contents = append(contents, util.Escape(tt.Recstr()))
		if len(contents) > 2 {
			return contents
		}
	}
	return contents
}

//CreateAllCachedirs creates all dirs in recentlist to be retrived when called recentlist.getall.
//(heavymoon)
func CreateAllCachedirs() {
	for _, rh := range recentlist.GetRecords() {
		ca := NewCache(rh.Datfile)
		if !ca.Exists() {
			ca.Subscribe()
		}
	}
}

//RecentStamp  returns time of getting by /recent.
func (c *Cache) RecentStamp() int64 {
	n, err := recentlist.Newest(c.Datfile)
	s := c.Stamp()
	if err != nil || n.Stamp < s {
		return s
	}
	return n.Stamp
}
