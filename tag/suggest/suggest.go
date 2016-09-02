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

package suggest

import (
	"log"
	"sync"

	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/record"
	"github.com/shingetsu-gou/shingetsu-gou/tag"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//SuggestedTagTable represents tags associated with datfile retrieved from network.
var list = make(map[string]tag.Slice)
var mutex sync.RWMutex

//init makes SuggestedTagTable obj and read info from the file.
func init() {
	if !util.IsFile(cfg.Sugtag()) {
		return
	}
	err := util.EachKeyValueLine(cfg.Sugtag(), func(k string, vs []string, i int) error {
		list[k] = tag.NewSlice(vs)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

//Save saves Sugtaglists.
func Save() {
	m := make(map[string][]string)
	mutex.RLock()
	for k, v := range list {
		s := v.GetTagstrSlice()
		m[k] = s
	}
	mutex.RUnlock()
	cfg.Fmutex.Lock()
	err := util.WriteMap(cfg.Sugtag(), m)
	cfg.Fmutex.Unlock()
	if err != nil {
		log.Println(err)
	}
}

//Get returns copy of Slice associated with datfile or returns def if not exists.
func Get(datfile string, def tag.Slice) tag.Slice {
	mutex.RLock()
	defer mutex.RUnlock()
	if v, exist := list[datfile]; exist {
		tags := make([]*tag.Tag, v.Len())
		copy(tags, v)
		return tag.Slice(tags)
	}
	return def
}

//keys return datfile names of Sugtaglist.
func keys() []string {
	mutex.RLock()
	defer mutex.RUnlock()
	ary := make([]string, len(list))
	i := 0
	for k := range list {
		ary[i] = k
		i++
	}
	return ary
}

//AddString adds tags to datfile from tagstrings.
func AddString(datfile string, vals []string) {
	mutex.Lock()
	defer mutex.Unlock()
	list[datfile] = list[datfile].AddString(vals)
}

//HasTagstr return true if one of tags has tagstr
func HasTagstr(datfile string, tagstr string) bool {
	mutex.RLock()
	defer mutex.RUnlock()
	return list[datfile].HasTagstr(tagstr)
}

//String return tagstr string of datfile.
func String(datfile string) string {
	mutex.RLock()
	defer mutex.RUnlock()
	return list[datfile].String()
}

//Prune removes Sugtaglists which are not listed in recs,
//or truncates its size to tagsize if listed.
func Prune(recs []*record.Head) {
	tmp := keys()
	mutex.Lock()
	defer mutex.Unlock()
	for _, r := range recs {
		if l := util.FindString(tmp, r.Datfile); l != -1 {
			tmp = append(tmp[:l], tmp[l+1:]...)
		}
		if v, exist := list[r.Datfile]; exist {
			v.Prune(cfg.TagSize)
		}
	}
	for _, datfile := range tmp {
		delete(list, datfile)
	}
}
