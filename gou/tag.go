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
	"strings"
)

//tag represents one tag.
type tag struct {
	tagstr string
	weight int
}

//tagList represents list of tags and base of other tag list.
type tagList struct {
	path string
	tags []*tag
}

//Len returns size of tags.
func (t tagList) Len() int {
	return len(t.tags)
}

//Swap swaps tag order.
func (t tagList) Swap(i, j int) {
	t.tags[i], t.tags[j] = t.tags[j], t.tags[i]
}

//Less is true if weight[i]< weigt[j]
func (t tagList) Less(i, j int) bool {
	return t.tags[i].weight < t.tags[j].weight
}

//newTagList read the tag info from datfile and return a tagList instance.
func newTagList(path string) *tagList {
	t := &tagList{
		path: path,
	}
	err := eachLine(path, func(line string, i int) error {
		t.tags = append(t.tags, &tag{line, 0})
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	return t
}

//getTagstrSlice returns tagstr slice of tags.
func (t *tagList) getTagstrSlice() []string {
	result := make([]string, len(t.tags))
	for i, v := range t.tags {
		result[i] = v.tagstr
	}
	return result
}

//string concatenates and returns tagstr of tags.
func (t *tagList) string() string {
	return strings.Join(t.getTagstrSlice(), "")
}

//checkAppend append tagstr=val tag if tagList doesn't have its tag.
func (t *tagList) checkAppend(val string) {
	if strings.ContainsAny(val, "<>&") || hasString(t.getTagstrSlice(), val) {
		return
	}
	t.tags = append(t.tags, &tag{val, 1})
}

//update removes tags and add tagstr=val tags.
func (t *tagList) update(val []string) {
	t.tags = t.tags[:0]
	for _, v := range val {
		ta := &tag{
			tagstr: v,
		}
		t.tags = append(t.tags, ta)
	}
}

//hasTagstr return true if one of tags has tagstr
func (t *tagList) hasTagstr(tagstr string) bool{
	for _, v := range t.tags {
		if v.tagstr==tagstr{
			return true
		}
	}
	return false
}

//addString add tagstr=vals tag
func (t *tagList) addString(vals []string) {
	for _, val := range vals {
		t.checkAppend(val)
	}
}

//add adds vals tags.
func (t *tagList) add(vals []*tag) {
	for _, val := range vals {
		if i := findString(t.getTagstrSlice(), val.tagstr); i >= 0 {
			t.tags[i].weight++
		} else {
			t.checkAppend(val.tagstr)
		}
	}
}

//sync saves tagstr of tags.
func (t *tagList) sync() {
	err := writeSlice(t.path, t.getTagstrSlice())
	if err != nil {
		log.Println(err)
	}
}

//SuggestedTagTable represents tags associated with datfile retrieved from network.
type SuggestedTagTable struct {
	sugtaglist map[string]*suggestedTagList
}

//newSuggestedTagTable make SuggestedTagTable obj and read info from the file.
func newSuggestedTagTable() *SuggestedTagTable {
	s := &SuggestedTagTable{
		sugtaglist: make(map[string]*suggestedTagList),
	}
	err := eachKeyValueLine(sugtag, func(k string, vs []string, i int) error {
		s.sugtaglist[k] = newSuggestedTagList(vs)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return s
}

//get returns suggestedTagList associated with datfile or returns def if not exists.
func (s *SuggestedTagTable) get(datfile string, def *suggestedTagList) *suggestedTagList {
	if v, exist := s.sugtaglist[datfile]; exist {
		return v
	}
	return def
}

//keys return datfile names of sugtaglist.
func (s *SuggestedTagTable) keys() []string {
	ary := make([]string, len(s.sugtaglist))
	i := 0
	for k := range s.sugtaglist {
		ary[i] = k
		i++
	}
	return ary
}

//sync saves sugtaglists.
func (s *SuggestedTagTable) sync() {
	m := make(map[string][]string)
	for k, v := range s.sugtaglist {
		s := v.getTagstrSlice()
		m[k] = s
	}
	err := writeMap(sugtag, m)
	if err != nil {
		log.Println(err)
	}
}

//prune removes sugtaglists which are not listed in recentlist,
//or truncates its size to tagsize if listed.
func (s *SuggestedTagTable) prune(recentlist *RecentList) {
	tmp := s.keys()
	for _, r := range recentlist.infos {
		if l := findString(tmp, r.datfile); l != -1 {
			tmp = append(tmp[:l], tmp[l:]...)
		}
		if v, exist := s.sugtaglist[r.datfile]; exist {
			v.prune(tagSize)
		}
	}
	for _, datfile := range tmp {
		delete(s.sugtaglist, datfile)
	}
}

//suggestedTabList represents tags retrieved from network.
type suggestedTagList struct {
	tagList
}

//newSuggestedTagList create suggestedTagList obj and adds tags tagstr=value.
func newSuggestedTagList(values []string) *suggestedTagList {
	s := &suggestedTagList{}
	for _, v := range values {
		s.tags = append(s.tags, &tag{v, 0})
	}
	return s
}

//prune truncates non-weighted tagList to size=size.
func (s *suggestedTagList) prune(size int) {
	sort.Sort(sort.Reverse(s.tagList))
	s.tagList.tags = s.tagList.tags[:size]
}

//UserTagList represents tags saved by the user.
type UserTagList struct {
	*tagList
}

//newUserTagList return userTagList obj.
func newUserTagList() *UserTagList {
	t := newTagList(taglist)
	return &UserTagList{t}
}

//sync saves taglist.
func (u *UserTagList) sync() {
	sort.Sort(u.tagList)
	u.tagList.sync()
}

//updateall removes all tags and reload from cachlist.
func (u *UserTagList) updateAll() {
	cachelist := newCacheList()
	if u.tags != nil {
		u.tags = u.tags[:0]
	}
	for _, c := range cachelist.caches {
		u.add(c.tags.tags)
	}
	u.sync()
}
