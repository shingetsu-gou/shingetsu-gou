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
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
)

//updateInfo represents one line in updatelist/recentlist
type updateInfo struct {
	stamp   int64
	id      string
	datfile string
}

//newUpdateInfoFromLine parse one line in udpate/recent list and returns updateInfo obj.
func newUpdateInfoFromLine(line string) (*updateInfo, error) {
	strs := strings.Split(strings.TrimRight(line, "\n\r"), "<>")
	if len(strs) < 3 {
		err := errors.New("illegal format")
		log.Println(err)
		return nil, err
	}
	u := &updateInfo{
		id:      strs[2],
		datfile: strs[3],
	}
	var err error
	u.stamp, err = strconv.ParseInt(strs[0], 10, 64)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return u, nil
}

//recstr returns one line of update/recentlist file.
func (u *updateInfo) recstr() string {
	return fmt.Sprintf("%d<>%s<>%s", u.stamp, u.id, u.datfile)
}

//UpdateList represents records list updated by remote nodes.
type UpdateList struct {
	updateFile  string
	updateRange int64
	infos       []*updateInfo
}

//newUpdateList makes UpdateList obj.
func newUpdateList() *UpdateList {
	u := &UpdateList{
		updateFile:  update,
		updateRange: int64(defaultUpdateRange),
	}
	u.loadFile()
	return u
}

//loadFile reads from file and add records.
func (u *UpdateList) loadFile() {
	err := eachLine(u.updateFile, func(line string, i int) error {
		vr, err := newUpdateInfoFromLine(line)
		if err == nil {
			u.infos = append(u.infos, vr)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

//append add a infos generated from the record.
func (u *UpdateList) append(r *record) {
	ui := &updateInfo{
		stamp:   r.stamp,
		datfile: r.datfile,
		id:      r.id,
	}
	u.infos = append(u.infos, ui)
}

//Less returns true if stamp of infos[i] < [j]
func (u *UpdateList) Less(i, j int) bool {
	return u.infos[i].stamp < u.infos[j].stamp
}

//Swap swaps infos order.
func (u *UpdateList) Swap(i, j int) {
	u.infos[i], u.infos[j] = u.infos[j], u.infos[i]
}

//Len returns size of infos
func (u *UpdateList) Len() int {
	return len(u.infos)
}

//find finds records and returns index. returns -1 if not found.
func (u *UpdateList) find(r *record) int {
	for i, v := range u.infos {
		if v.datfile == r.datfile && v.id == r.id && v.stamp == r.stamp {
			return i
		}
	}
	return -1
}

//hasRecord returns true if has record r.
func (u *UpdateList) hasInfo(r *record) bool {
	return u.find(r) != -1
}

//remove removes info which is same as record r
func (u *UpdateList) remove(rec *record) {
	if l := u.find(rec); l != -1 {
		u.infos = append(u.infos[:l], u.infos[l:]...)
	}
}

//removeInfo removes info r
func (u *UpdateList) removeInfo(r *updateInfo) {
	for i, v := range u.infos {
		if v.datfile == r.datfile && v.id == r.id && v.stamp == r.stamp {
			u.infos = append(u.infos[:i], u.infos[i:]...)
			return
		}
	}
}

//getRecstrSlice returns slice of recstr string of infos.
func (u *UpdateList) getRecstrSlice() []string {
	result := make([]string, len(u.infos))
	for i, v := range u.infos {
		result[i] = v.recstr()
	}
	return result
}

//sync remove old records and save to the file.
func (u *UpdateList) sync() {
	for i, r := range u.infos {
		if u.updateRange > 0 && r.stamp+u.updateRange < time.Now().Unix() {
			u.infos = append(u.infos[:i], u.infos[i:]...)
		}
		err := writeSlice(u.updateFile, u.getRecstrSlice())
		if err != nil {
			log.Println(err)
		}
	}
}

//Recentlist represents records list udpated by remote host and
//gotten by /gateway.cgi/recent
type RecentList struct {
	*UpdateList
}

//newRecentList load a file and create a RecentList obj.
func newRecentList() *RecentList {
	r := &UpdateList{
		updateFile:  recent,
		updateRange: int64(recentRange),
	}
	r.loadFile()
	return &RecentList{r}
}

//getAll retrieves recent records from nodes insearchlist and stores them.
//tags are shuffled and truncated to tagsize and stored to sugtags in cache.
//also source nodes are stored into lookuptable.
//also tags which recentlist doen't have in sugtagtable are truncated
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
			rec := &record{}
			err := rec.parse(line)
			if err == nil {
				r.append(rec)
				ca := newCache(rec.datfile)
				tags := strings.Fields(strings.TrimSpace(rec.getBodyValue("tag", "")))
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

//uniq removes infos which has same datfile.
//new ones are alive.
func (r *RecentList) uniq() {
	date := make(map[string]*updateInfo)
	for i, rec := range r.infos {
		if _, exist := date[rec.datfile]; !exist {
			date[rec.datfile] = rec
		} else {
			if date[rec.datfile].stamp < rec.stamp {
				r.removeInfo(date[rec.datfile])
				date[rec.datfile] = rec
			} else {
				r.infos = append(r.infos[:i], r.infos[i:]...)
			}
		}
	}
}

//sync singlize records and save new ones.
func (r *RecentList) sync() {
	r.uniq()
	r.UpdateList.sync()
}
