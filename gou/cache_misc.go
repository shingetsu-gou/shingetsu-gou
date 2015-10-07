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

type UpdateList struct {
	updateFile  string
	updateRange int64
	lookup      map[string]*record
	records     []*record
}

func newUpdateList() *UpdateList {
	u := &UpdateList{
		updateFile:  update,
		updateRange: int64(defaultUpdateRange),
		lookup:      make(map[string]*record),
	}
	u.loadFile()
	return u
}
func (u *UpdateList) loadFile() {
	err := eachLine(u.updateFile, func(line string, i int) error {
		vr := u.makeRecord(line)
		u.append(vr)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

func (u *UpdateList) append(r *record) {
	u.records = append(u.records, r)
}
func (u *UpdateList) Less(i, j int) bool {
	return u.records[i].stamp < u.records[j].stamp
}

func (u *UpdateList) Swap(i, j int) {
	u.records[i], u.records[j] = u.records[j], u.records[i]
}

func (u *UpdateList) Len() int {
	return len(u.records)
}

func (u *UpdateList) find(r *record) int {
	for i, v := range u.records {
		if v.recstr == r.recstr {
			return i
		}
	}
	return -1
}

func (u *UpdateList) hasRecord(r *record) bool {
	return u.find(r) != -1
}

func (u *UpdateList) remove(rec *record) {
	if l := u.find(rec); l != -1 {
		u.records = append(u.records[:l], u.records[l:]...)
	}
}

func (u *UpdateList) addLookup(rec *record) {
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
func (u *UpdateList) makeRecord(line string) *record {
	buf := strings.Split(strings.TrimRight(line, "\n\r"), "<>")
	if len(buf) > 2 && buf[0] != "" && buf[1] != "" && buf[2] != "" {
		idstr := buf[0] + "_" + buf[1]
		vr := newRecord(buf[2], idstr)
		err := vr.parse(line)
		if err != nil {
			log.Println(err)
		}
		return vr
	}
	return nil
}

func (u *UpdateList) getRecstrSlice() []string {
	result := make([]string, len(u.records))
	for i, v := range u.records {
		result[i] = v.recstr
	}
	return result
}

func (u *UpdateList) sync() {
	for _, r := range u.records {
		if u.updateRange > 0 && r.stamp+u.updateRange < time.Now().Unix() {
			u.remove(r)
		}
		err := writeSlice(u.updateFile, u.getRecstrSlice())
		if err != nil {
			log.Println(err)
		}
	}
}

type RecentList struct {
	*UpdateList
}

func newRecentList() *RecentList {
	r := &UpdateList{
		updateFile:  recent,
		updateRange: int64(recentRange),
		lookup:      make(map[string]*record),
	}
	r.loadFile()
	return &RecentList{r}
}

func (r *RecentList) getAll() {
	lookupTable.clear()
	var begin int64
	if recentRange > 0 {
		begin = time.Now().Unix() - int64(recentRange)
	}
	var res []string
	for count, n := range searchList.nodes {
		var err error
		res, err = n.talk("/recent/" + strconv.FormatInt(begin, 10) + "-")
		if err != nil {
			log.Println(err)
			continue
		}
		for _, line := range res {
			rec := r.makeRecord(line)
			if rec != nil {
				r.records = append(r.records, rec)
				ca := newCache(rec.datfile)
				tags := strings.Fields(strings.TrimSpace(rec.Get("tag", "")))
				shuffle(sort.StringSlice(tags))
				tags = tags[tagSize:]
				if len(tags) > 0 {
					ca.sugtags.addString(tags)
					ca.sugtags.sync()
					lookupTable.add(rec.datfile, n)
				}
			}
		}
		if count >= searchDepth {
			break
		}
	}
	r.sync()
	lookupTable.sync(false)
	suggestedTagTable.prune(r)
	suggestedTagTable.sync()
}

func (r *RecentList) uniq() {
	date := make(map[string]*record)
	for _, rec := range r.records {
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

func (r *RecentList) sync() {
	r.uniq()
	r.UpdateList.sync()
}
