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
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//tag represents one tag.
type Tag struct {
	Tagstr string
	weight int
}

type Tagslice []*Tag

//Len returns size of tags.
func (t Tagslice) Len() int {
	return len(t)
}

//Swap swaps tag order.
func (t Tagslice) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

//Less is true if weight[i]< weigt[j]
func (t Tagslice) Less(i, j int) bool {
	return t[i].weight < t[j].weight
}

//newTagslice create suggestedTagList obj and adds tags tagstr=value.
func newTagslice(values []string) Tagslice {
	s := make([]*Tag, len(values))
	for i, v := range values {
		s[i] = &Tag{v, 0}
	}
	return s
}

//loadTagSlice load a file and returns Tagslice.
func loadTagslice(path string) Tagslice {
	var t Tagslice
	if !util.IsFile(path) {
		return t
	}
	err := util.EachLine(path, func(line string, i int) error {
		t = append(t, &Tag{line, 0})
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	return t
}

//getTagstrSlice returns tagstr slice of tags.
func (t Tagslice) GetTagstrSlice() []string {
	result := make([]string, t.Len())
	for i, v := range t {
		result[i] = v.Tagstr
	}
	return result
}

//string concatenates and returns tagstr of tags.
func (t Tagslice) string() string {
	return strings.Join(t.GetTagstrSlice(), " ")
}

//prune truncates non-weighted tagList to size=size.
func (t Tagslice) prune(size int) Tagslice {
	sort.Sort(sort.Reverse(t))
	if t.Len() > size {
		t = t[:size]
	}
	return t
}

//checkAppend append tagstr=val tag if tagList doesn't have its tag.
func (t Tagslice) checkAppend(val string) {
	if strings.ContainsAny(val, "<>&") || util.HasString(t.GetTagstrSlice(), val) {
		return
	}
	t = append(t, &Tag{val, 1})
}

//hasTagstr return true if one of tags has tagstr
func (t Tagslice) HasTagstr(tagstr string) bool {
	for _, v := range t {
		if v.Tagstr == tagstr {
			return true
		}
	}
	return false
}

//addString add tagstr=vals tag
func (t Tagslice) addString(vals []string) {
	for _, val := range vals {
		t.checkAppend(val)
	}
}

//add adds vals tags.
func (t Tagslice) add(vals []*Tag) {
	for _, val := range vals {
		if i := util.FindString(t.GetTagstrSlice(), val.Tagstr); i >= 0 {
			t[i].weight++
		} else {
			t.checkAppend(val.Tagstr)
		}
	}
}

//sync saves tagstr of tags to path.
func (t Tagslice) sync(path string) {
	err := util.WriteSlice(path, t.GetTagstrSlice())
	if err != nil {
		log.Println(err)
	}
}

type SuggestedTagTableConfig struct {
	TagSize int
	Sugtag  string
	Fmutex  *sync.RWMutex
}

//SuggestedTagTable represents tags associated with datfile retrieved from network.
type SuggestedTagTable struct {
	*SuggestedTagTableConfig
	Sugtaglist map[string]Tagslice
	mutex      sync.RWMutex
}

//newSuggestedTagTable make SuggestedTagTable obj and read info from the file.
func NewSuggestedTagTable(cfg *SuggestedTagTableConfig) *SuggestedTagTable {
	s := &SuggestedTagTable{
		SuggestedTagTableConfig: cfg,
		Sugtaglist:              make(map[string]Tagslice),
	}
	if !util.IsFile(cfg.Sugtag) {
		return s
	}
	err := util.EachKeyValueLine(cfg.Sugtag, func(k string, vs []string, i int) error {
		s.Sugtaglist[k] = newTagslice(vs)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return s
}

//sync saves Sugtaglists.
func (s *SuggestedTagTable) sync() {
	log.Println("syncing..")
	m := make(map[string][]string)
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for k, v := range s.Sugtaglist {
		s := v.GetTagstrSlice()
		m[k] = s
	}
	s.Fmutex.Lock()
	err := util.WriteMap(s.Sugtag, m)
	s.Fmutex.Unlock()
	if err != nil {
		log.Println(err)
	}
}

//get returns copy of suggestedTagList associated with datfile or returns def if not exists.
func (s *SuggestedTagTable) Get(datfile string, def Tagslice) Tagslice {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if v, exist := s.Sugtaglist[datfile]; exist {
		tags := make([]*Tag, v.Len())
		copy(tags, v)
		return Tagslice(tags)
	}
	return def
}

//keys return datfile names of Sugtaglist.
func (s *SuggestedTagTable) keys() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	ary := make([]string, len(s.Sugtaglist))
	i := 0
	for k := range s.Sugtaglist {
		ary[i] = k
		i++
	}
	return ary
}

//addString adds tags to datfile from tagstrings.
func (s *SuggestedTagTable) addString(datfile string, vals []string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Sugtaglist[datfile].addString(vals)
}

//hasTagstr return true if one of tags has tagstr
func (s *SuggestedTagTable) HasTagstr(datfile string, tagstr string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.Sugtaglist[datfile].HasTagstr(tagstr)
}

//string return tagstr string.
func (s *SuggestedTagTable) String(datfile string) string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.Sugtaglist[datfile].string()
}

//prune removes Sugtaglists which are not listed in recentlist,
//or truncates its size to tagsize if listed.
func (s *SuggestedTagTable) prune(recentlist *RecentList) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	tmp := s.keys()
	for _, r := range recentlist.infos {
		if l := util.FindString(tmp, r.Datfile); l != -1 {
			tmp = append(tmp[:l], tmp[l+1:]...)
		}
		if v, exist := s.Sugtaglist[r.Datfile]; exist {
			v.prune(s.TagSize)
		}
	}
	for _, datfile := range tmp {
		delete(s.Sugtaglist, datfile)
	}
}

type UserTagConfig struct {
	CacheDir string
	Fmutex   *sync.RWMutex
}

//UserTagList represents tags saved by the user.
type UserTag struct {
	*UserTagConfig
	mutex   sync.Mutex
	isClean bool
	tags    Tagslice
}

func NewUserTag(cfg *UserTagConfig) *UserTag {
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

//get reads tags from the disk and retrusn Tagslice.
func (u *UserTag) Get() Tagslice {
	u.Fmutex.RLock()
	defer u.Fmutex.RUnlock()
	u.mutex.Lock()
	defer u.mutex.Unlock()
	if u.isClean {
		return u.tags
	}
	var tags Tagslice
	err := util.EachFiles(u.CacheDir, func(i os.FileInfo) error {
		fname := path.Join(u.CacheDir, i.Name(), "tag.txt")
		if i.IsDir() && util.IsFile(fname) {
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
