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
	"sort"
	"strconv"
	"strings"
	"time"
)

type updateList struct {
	updateFile  string
	updateRange int64
	lookup      map[string]*record
	tiedlist    []*record
}

func newUpdateList(updateFile string, updateRange int64) *updateList {
	if updateFile == "" {
		updateFile = update
	}
	if updateRange == 0 {
		updateRange = int64(update_range)
	}
	u := &updateList{
		updateFile:  updateFile,
		updateRange: updateRange,
		lookup:      make(map[string]*record),
		tiedlist:    make([]*record, 0),
	}
	//cache:true
	err := eachLine(updateFile, func(line string, i int) error {
		vr := u.makeRecord(line)
		u.append(vr)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return u
}
func (u *updateList) append(r *record) {
	u.tiedlist = append(u.tiedlist, r)
}

func (u *updateList) Len() int {
	return len(u.tiedlist)
}

func (u *updateList) find(r *record) int {
	return findString(u, r.recstr)
}

func (u *updateList) has(r *record) bool {
	return u.find(r) != -1
}

func (u *updateList) Get(i int) string {
	return u.tiedlist[i].recstr
}

func (u *updateList) remove(rec *record) {
	if l := u.find(rec); l != -1 {
		u.tiedlist = append(u.tiedlist[:l], u.tiedlist[l:]...)
	}
}

func (u *updateList) addLookup(rec *record) {
	exist := false
	for k, v := range u.lookup {
		if k == rec.datfile && v.stamp < rec.stamp {
			u.lookup[rec.datfile] = rec
		}
	}
	if !exist {
		u.lookup[rec.datfile] = rec
	}
}
func (u *updateList) makeRecord(line string) *record {
	buf := strings.Split(strings.TrimRight(line, "\n\r"), "<>")
	if len(buf) > 2 && buf[0] != "" && buf[1] != "" && buf[2] != "" {
		idstr := buf[0] + "_" + buf[1]
		vr := newRecord(buf[2], idstr)
		vr.parse(line)
		return vr
	}
	return nil
}

func (u *updateList) sync() {
	for _, r := range u.tiedlist {
		if u.updateRange > 0 && r.stamp+u.updateRange < time.Now().Unix() {
			u.remove(r)
		}
		writeSlice(u.updateFile, u)
	}
}

type recentList struct {
	*updateList
}

func newRecentList() *recentList {
	r := newUpdateList(recent, int64(recent_range))
	return &recentList{r}
}

func (r *recentList) getAll() {
	sl := newSearchList()
	lt := newLookupTable()
	lt.clear()
	st := newSuggestedTagTable()
	var begin int64
	if recent_range > 0 {
		begin = time.Now().Unix() - int64(recent_range)
	}
	var res []string
	for count, n := range sl.tiedlist {
		var err error
		res, err = n.talk("/recent/" + strconv.FormatInt(begin, 10) + "-")
		if err != nil {
			log.Println(err)
			continue
		}
		for _, line := range res {
			rec := r.makeRecord(line)
			if rec != nil {
				r.tiedlist = append(r.tiedlist, rec)
				ca := newCache(rec.datfile, st, nil)
				tags := strings.Split(strings.TrimSpace(rec.Get("tag", "")), " \t\r\n")
				shuffle(sort.StringSlice(tags))
				tags = tags[tag_size:]
				if len(tags) > 0 {
					ca.sugtags.addString(tags)
					ca.sugtags.sync()
					lt.add(rec.datfile, n)
				}
			}
		}
		if count >= search_depth {
			break
		}
	}
	r.sync()
	lt.sync(false)
	st.prune(r)
	st.sync()
}

func (r *recentList) uniq() {
	date := make(map[string]*record)
	for _, rec := range r.tiedlist {
		if _, exist := date[rec.datfile]; !exist {
			date[rec.datfile] = rec
		} else {
			if date[rec.datfile].stamp < rec.stamp {
				r.remove(date[rec.datfile])
				date[rec.datfile] = rec
			} else {
				r.remove(rec)
			}
		}
	}
}

func (r *recentList) sync() {
	r.uniq()
	r.updateList.sync()
}
