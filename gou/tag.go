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
	"sort"
	"strings"
	"sync"
)

//tag represents one tag.
type tag struct {
	Tagstr string
	weight int
}

type tagslice []*tag

//Len returns size of tags.
func (t tagslice) Len() int {
	return len(t)
}

//Swap swaps tag order.
func (t tagslice) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

//Less is true if weight[i]< weigt[j]
func (t tagslice) Less(i, j int) bool {
	return t[i].weight < t[j].weight
}

//newTagslice create suggestedTagList obj and adds tags tagstr=value.
func newTagslice(values []string) tagslice {
	s := make([]*tag, len(values))
	for i, v := range values {
		s[i] = &tag{v, 0}
	}
	return s
}

//loadTagSlice load a file and returns tagslice.
func loadTagslice(path string) tagslice {
	var t tagslice
	if !IsFile(path) {
		return t
	}
	err := eachLine(path, func(line string, i int) error {
		t = append(t, &tag{line, 0})
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	return t
}

//getTagstrSlice returns tagstr slice of tags.
func (t tagslice) getTagstrSlice() []string {
	result := make([]string, t.Len())
	for i, v := range t {
		result[i] = v.Tagstr
	}
	return result
}

//string concatenates and returns tagstr of tags.
func (t tagslice) string() string {
	return strings.Join(t.getTagstrSlice(), " ")
}

//prune truncates non-weighted tagList to size=size.
func (t tagslice) prune(size int) tagslice {
	sort.Sort(sort.Reverse(t))
	if t.Len() > size {
		t = t[:size]
	}
	return t
}

//checkAppend append tagstr=val tag if tagList doesn't have its tag.
func (t tagslice) checkAppend(val string) {
	if strings.ContainsAny(val, "<>&") || hasString(t.getTagstrSlice(), val) {
		return
	}
	t = append(t, &tag{val, 1})
}

//hasTagstr return true if one of tags has tagstr
func (t tagslice) hasTagstr(tagstr string) bool {
	for _, v := range t {
		if v.Tagstr == tagstr {
			return true
		}
	}
	return false
}

//addString add tagstr=vals tag
func (t tagslice) addString(vals []string) {
	for _, val := range vals {
		t.checkAppend(val)
	}
}

//add adds vals tags.
func (t tagslice) add(vals []*tag) {
	for _, val := range vals {
		if i := findString(t.getTagstrSlice(), val.Tagstr); i >= 0 {
			t[i].weight++
		} else {
			t.checkAppend(val.Tagstr)
		}
	}
}

//sync saves tagstr of tags to path.
func (t tagslice) sync(path string) {
	err := writeSlice(path, t.getTagstrSlice())
	if err != nil {
		log.Println(err)
	}
}

type SuggestedTagTableConfig struct {
	tagSize int
	sugtag  string
	fmutex  *sync.RWMutex
}

//SuggestedTagTable represents tags associated with datfile retrieved from network.
type SuggestedTagTable struct {
	*SuggestedTagTableConfig
	sugtaglist map[string]tagslice
	mutex      sync.RWMutex
}

//newSuggestedTagTable make SuggestedTagTable obj and read info from the file.
func newSuggestedTagTable(cfg *SuggestedTagTableConfig) *SuggestedTagTable {
	s := &SuggestedTagTable{
		SuggestedTagTableConfig: cfg,
		sugtaglist:              make(map[string]tagslice),
	}
	if !IsFile(cfg.sugtag) {
		return s
	}
	err := eachKeyValueLine(cfg.sugtag, func(k string, vs []string, i int) error {
		s.sugtaglist[k] = newTagslice(vs)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return s
}

//sync saves sugtaglists.
func (s *SuggestedTagTable) sync() {
	log.Println("syncing..")
	m := make(map[string][]string)
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for k, v := range s.sugtaglist {
		s := v.getTagstrSlice()
		m[k] = s
	}
	s.fmutex.Lock()
	err := writeMap(s.sugtag, m)
	s.fmutex.Unlock()
	if err != nil {
		log.Println(err)
	}
}

//get returns copy of suggestedTagList associated with datfile or returns def if not exists.
func (s *SuggestedTagTable) get(datfile string, def tagslice) tagslice {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if v, exist := s.sugtaglist[datfile]; exist {
		tags := make([]*tag, v.Len())
		copy(tags, v)
		return tagslice(tags)
	}
	return def
}

//keys return datfile names of sugtaglist.
func (s *SuggestedTagTable) keys() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	ary := make([]string, len(s.sugtaglist))
	i := 0
	for k := range s.sugtaglist {
		ary[i] = k
		i++
	}
	return ary
}

//addString adds tags to datfile from tagstrings.
func (s *SuggestedTagTable) addString(datfile string, vals []string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.sugtaglist[datfile].addString(vals)
}

//hasTagstr return true if one of tags has tagstr
func (s *SuggestedTagTable) hasTagstr(datfile string, tagstr string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.sugtaglist[datfile].hasTagstr(tagstr)
}

//string return tagstr string.
func (s *SuggestedTagTable) string(datfile string) string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.sugtaglist[datfile].string()
}

//prune removes sugtaglists which are not listed in recentlist,
//or truncates its size to tagsize if listed.
func (s *SuggestedTagTable) prune(recentlist *RecentList) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	tmp := s.keys()
	for _, r := range recentlist.infos {
		if l := findString(tmp, r.datfile); l != -1 {
			tmp = append(tmp[:l], tmp[l+1:]...)
		}
		if v, exist := s.sugtaglist[r.datfile]; exist {
			v.prune(s.tagSize)
		}
	}
	for _, datfile := range tmp {
		delete(s.sugtaglist, datfile)
	}
}

type UserTagConfig struct {
	cacheDir string
	fmutex   *sync.RWMutex
}

//UserTagList represents tags saved by the user.
type UserTag struct {
	*UserTagConfig
	mutex   sync.Mutex
	isClean bool
	tags    tagslice
}

func newUserTag(cfg *UserTagConfig) *UserTag {
	return &UserTag{
		UserTagConfig: cfg,
	}
}

//setDirty sets dirty flag.
func (u *UserTag) setDirty() {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	u.isClean = false
}

//get reads tags from the disk and retrusn tagslice.
func (u *UserTag) get() tagslice {
	u.fmutex.RLock()
	defer u.fmutex.RUnlock()
	u.mutex.Lock()
	defer u.mutex.Unlock()
	if u.isClean {
		return u.tags
	}
	var tags tagslice
	err := eachFiles(u.cacheDir, func(i os.FileInfo) error {
		fname := path.Join(u.cacheDir, i.Name(), "tag.txt")
		if i.IsDir() && IsFile(fname) {
			t, err := ioutil.ReadFile(fname)
			if err != nil {
				return err
			}
			tags.addString(strings.Split(string(t), "\r\n"))
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	u.tags = tags
	u.isClean = true
	return tags
}
