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

//Stamp returns lastest stampl of records in the cache.
func (c *Cache) Stamp() int64 {
	r, err := db.Int64s("select  Stamp from record  where Thread=? order by Stamp ", c.Datfile)
	if err != nil {
		log.Print(err)
		return 0
	}
	if len(r) == 0 {
		return 0
	}
	return r[len(r)-1]
}

//Len returns # of records in the cache.
func (c *Cache) Len() int {
	cnt, err := db.Int64("select count(*)  from record where Thread=?", c.Datfile)
	if err != nil {
		log.Print(err)
		return 0
	}
	return int(cnt)
}

//Velocity returns number of records in one days in the cache.
func (c *Cache) Velocity() int {
	cnt, err := db.Int64("select count(*)  from record where Stamp>? and Thread=?",
		time.Now().Add(-7*24*time.Hour).Second(), c.Datfile)
	if err != nil {
		log.Print(err)
		return 0
	}
	return int(cnt)
}

//Size returns sum of body char length of records in the cache.
func (c *Cache) Size() int64 {
	if c.Len() == 0 {
		return 0
	}
	//sqlite3-specific cmd
	cntt, err := db.Int64("select sum(length(Body))  from record where Thread=?", c.Datfile)
	if err != nil {
		log.Print(err)
		return 0
	}
	return cntt
}

const (
	//Alive counts records that are not removed.
	Alive = 1
	//Removed counts records that are  removed.
	Removed = 2
	//All counts all records
	All = 3
)

//LoadRecords loads and returns record maps from the disk..
func (c *Cache) LoadRecords(kind int) record.Map {
	var cond string
	switch kind {
	case Alive:
		cond = "and Deleted=0"
	case Removed:
		cond = "and Deleted=1"
	case All:
	}
	r, err := record.FromRecordDB("select  * from record where Thread=? "+cond, c.Datfile)
	if err != nil {
		log.Print(err)
		return nil
	}
	return r
}

//Subscribe add the thread to thread db.
func (c *Cache) Subscribe() {
	_, err := db.DB.Exec("insert into thread(Thread) values(?)", c.Datfile)
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
	_, err := db.DB.Exec("delete from record   Thread= ? order where", c.Datfile)
	if err != nil {
		log.Println(err)
	}
	_, err = db.DB.Exec("delete from thread   Thread= ? order where ", c.Datfile)
	if err != nil {
		log.Println(err)
	}
}

//HasRecord return true if  cache has more than one records or removed records.
func (c *Cache) HasRecord() bool {
	cnt, err := db.Int64("select  count(*) from record where (Thread=? and Deleted=0)", c.Datfile)
	if err != nil {
		log.Print(err)
		return false
	}
	return cnt > 0
}

//Exists return true is datapath exists.
func (c *Cache) Exists() bool {
	cnt, err := db.Int64("select  count(*) from thread where Thread=?", c.Datfile)
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
	m, err := record.FromRecordDB("select * from record where Thread=? and Deleted=0", c.Datfile)
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
	n := recentlist.Newest(c.Datfile)
	s := c.Stamp()
	if n == nil || n.Stamp < s {
		return s
	}
	return n.Stamp
}
