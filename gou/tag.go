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

var tagCache = make(map[string]*tagList)

type tag struct {
	tagstr string
	weight int
}

type tagList struct {
	datfile string
	path    string
	tags    tagSlice
}

func (t tagList) Len() int {
	return len(t.tags)
}
func (t tagList) Swap(i, j int) {
	t.tags[i], t.tags[j] = t.tags[j], t.tags[i]
}
func (t tagList) Less(i, j int) bool {
	return t.tags[i].weight < t.tags[j].weight
}

func (t *tagList) Get(i int) string {
	return t.tags[i].tagstr
}

func newTagList(datfile, path string, caching bool) *tagList {
	if t, exist := tagCache[path]; exist {
		return t
	}
	t := &tagList{datfile: datfile,
		path: path,
		tags: make([]*tag, 0),
	}
	err := eachLine(path, func(line string, i int) error {
		t.tags = append(t.tags, &tag{line, 0})
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	if caching {
		tagCache[path] = t
	}

	return t
}

type tagSlice []*tag

func (ts tagSlice)toStringSlice() []string {
	result := make([]string, len(ts))
	for i, v := range ts {
		result[i] = v.tagstr
	}
	return result
}

func (t *tagList) string() string {
	var result string
	for _, v := range t.tags {
		result += v.tagstr
	}
	return result

}

func (t *tagList) checkAppend(val string) {
	if strings.ContainsAny(val, "<>&") || hasString(t, val) {
		return
	}
	t.tags = append(t.tags, &tag{val, 1})
}

func (t *tagList) update(val []string) {
	t.tags = t.tags[:0]
	for _, v := range val {
		ta := &tag{
			tagstr: v,
		}
		t.tags = append(t.tags, ta)
	}
}

func (t *tagList) addString(vals []string) {
	for _, val := range vals {
		t.checkAppend(val)
	}
}

func (t *tagList) add(vals []*tag) {
	for _, val := range vals {
		if i := findString(t, val.tagstr); i >= 0 {
			t.tags[i].weight++
		} else {
			t.checkAppend(val.tagstr)
		}
	}
}

func (t *tagList) sync() {
	err := writeSlice(t.path, t)
	if err != nil {
		log.Println(err)
	}
}

type suggestedTagTable struct {
	datfile  string
	tieddict map[string]*suggestedTagList
}

func newSuggestedTagTable() *suggestedTagTable {
	s := &suggestedTagTable{
		datfile:  sugtag,
		tieddict: make(map[string]*suggestedTagList),
	}
	err := eachKeyValueLine(sugtag, func(k string, vs []string, i int) error {
		s.tieddict[k] = newSuggestedTagList(s, "", nil)
		for _, v := range vs {
			s.tieddict[k].tags = append(s.tieddict[k].tags, &tag{v, 0})
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return s
}

func (s *suggestedTagTable) Len() int {
	return len(s.tieddict)
}

func (s *suggestedTagTable) Get(i string, def *suggestedTagList) *suggestedTagList {
	if v, exist := s.tieddict[i]; exist {
		return v
	}
	return def
}

func (s *suggestedTagTable) keys() []string {
	ary := make([]string, len(s.tieddict))
	i := 0
	for k := range s.tieddict {
		ary[i] = k
		i++
	}
	return ary
}

func (s *suggestedTagTable) sync() {
	m := make(map[string][]string)
	for k, v := range s.tieddict {
		s := v.tags.toStringSlice()
		m[k] = s
	}
	err := writeMap(s.datfile, m)
	if err != nil {
		log.Println(err)
	}
}

func (s *suggestedTagTable) prune(recentlist *recentList) {
	tmp := s.keys()
	for _, r := range recentlist.tiedlist {
		if l := findString(stringSlice(tmp), r.datfile); l != -1 {
			tmp = append(tmp[:l], tmp[l:]...)
		}
		if v, exist := s.tieddict[r.datfile]; exist {
			v.prune(tagSize)
		}
	}
	for _, datfile := range tmp {
		delete(s.tieddict, datfile)
	}
}

type suggestedTagList struct {
	tagList
	datfile string
	table   *suggestedTagTable
}

func newSuggestedTagList(table *suggestedTagTable, datfile string, values []string) *suggestedTagList {
	s := &suggestedTagList{
		datfile: datfile,
		table:   table,
	}
	for _, v := range values {
		s.tags = append(s.tags, &tag{v, 0})
	}
	return s
}

func (s *suggestedTagList) prune(size int) {
	sort.Sort(sort.Reverse(s.tagList))
	s.tagList.tags = s.tagList.tags[:size]
}

type userTagList struct {
	*tagList
}

func newUserTagList() *userTagList {
	t := newTagList("", taglist, true)
	return &userTagList{t}
}

func (u *userTagList) sync() {
	sort.Sort(u.tagList)
	u.tagList.sync()
}
func (u *userTagList) updateAll() {
	cachelist := newCacheList()
	u.update([]string{})
	for _, c := range cachelist.caches {
		u.add(c.tags.tags)
	}
	u.sync()
}
