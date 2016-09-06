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

package recentlist

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/db"
	"github.com/shingetsu-gou/shingetsu-gou/node"
	"github.com/shingetsu-gou/shingetsu-gou/node/manager"
	"github.com/shingetsu-gou/shingetsu-gou/record"
	"github.com/shingetsu-gou/shingetsu-gou/tag/suggest"
)

const defaultUpdateRange = 24 * time.Hour // Seconds

//IsInUpdateRange returns true if stamp is in updateRange.
func IsInUpdateRange(nstamp int64) bool {
	now := time.Now()
	if now.Add(-defaultUpdateRange).Unix() < nstamp && nstamp < now.Add(defaultUpdateRange).Unix() {
		return true
	}
	return false
}

//RecentList represents records list udpated by remote host and
//gotten by /gateway.cgi/Recent

//Datfiles returns all datfile names in recentlist.
func Datfiles() []string {
	datfile, err := db.Strings("select Thread from recent group by Thread")
	if err != nil {
		log.Print(err)
		return nil
	}
	return datfile
}

//Newest returns newest record of datfile in the list.
//if not found returns nil.
func Newest(datfile string) *record.Head {
	rows, err := db.DB.Query("select * from recent  where Thread=? order by Stamp DESC", datfile)
	defer func() {
		errr := rows.Close()
		if errr != nil {
			log.Println(err)
		}
	}()
	if err != nil {
		log.Print(err)
		return nil
	}
	if !rows.Next() {
		return nil
	}
	r := record.Head{}
	var id int
	err = rows.Scan(&id, &r.Stamp, &r.ID, &r.Datfile)
	if err != nil {
		log.Print(err)
		return nil
	}
	return &r
}

//Append add a infos generated from the record.
func Append(rec *record.Head) {
	if find(rec) {
		return
	}
	_, err := db.DB.Exec("insert into recent(Stamp,Hash,Thread) values(?,?,?)", rec.Stamp, rec.ID, rec.Datfile)
	if err != nil {
		log.Print(err)
	}
}

//find finds records and returns index. returns -1 if not found.
func find(rec *record.Head) bool {
	r, err := db.Int64("select count(*) from recent where Stamp=? and Hash=? and Thread=?", rec.Stamp, rec.ID, rec.Datfile)
	if err != nil {
		log.Println(err)
		return false
	}
	return r > 0
}

//RemoveOlds remove old records..
func RemoveOlds() {
	if defaultUpdateRange <= 0 {
		return
	}
	_, err := db.DB.Exec("delete from recent where Stamp<? ", time.Now().Unix()-int64(defaultUpdateRange))
	if err != nil {
		log.Println(err)
	}
}

//Getall retrieves Recent records from nodes in searchlist and stores them.
//tags are shuffled and truncated to tagsize and stored to sugtags in cache.
//also source nodes are stored into lookuptable.
//also tags which Recentlist doen't have in sugtagtable are truncated
func Getall(all bool) {
	const searchNodes = 5

	var begin int64
	if cfg.RecentRange > 0 && !all {
		begin = time.Now().Unix() - cfg.RecentRange
	}
	nodes := manager.Random(nil, searchNodes)
	var wg sync.WaitGroup
	for _, n := range nodes {
		wg.Add(1)
		go get(begin, &wg, n)
	}
	wg.Wait()
	suggest.Prune(GetRecords())
}

func get(begin int64, wg *sync.WaitGroup, n *node.Node) {
	defer wg.Done()
	var res []string
	var err error
	res, err = n.Talk("/recent/"+strconv.FormatInt(begin, 10)+"-", nil)
	if err != nil {
		manager.RemoveFromAllTable(n)
		log.Println(err)
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		log.Print(err)
		return
	}
	for _, line := range res {
		rec := record.Make(line)
		if rec == nil {
			continue
		}
		Append(rec.Head)
		tags := strings.Fields(strings.TrimSpace(rec.GetBodyValue("tag", "")))
		if len(tags) > 0 {
			suggest.AddString(rec.Datfile, tags)
			manager.AppendToTable(rec.Datfile, n)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Println(err)
	}
	log.Println("added", len(res), "recent records from", n.Nodestr)
}

//GetRecords copies and returns recorcds in recentlist.
func GetRecords() []*record.Head {
	inf, err := record.FromRecentDB("select * from recent")
	if err != nil {
		log.Print(err)
		return nil
	}
	return inf
}
