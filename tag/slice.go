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

package tag

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//Slice is a slice of *Tag.
type Slice []*Tag

//Len returns size of tags.
func (t Slice) Len() int {
	return len(t)
}

//Swap swaps tag order.
func (t Slice) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

//Less is true if weight[i]< weigt[j]
func (t Slice) Less(i, j int) bool {
	return t[i].weight < t[j].weight
}

//NewSlice create TagList obj and adds tags tagstr=value.
func NewSlice(values []string) Slice {
	s := make([]*Tag, len(values))
	for i, v := range values {
		s[i] = &Tag{v, 0}
	}
	return s
}

//Load load a file and returns Slice.
func Load(path string) Slice {
	var t Slice
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

//GetTagstrSlice returns tagstr slice of tags.
func (t Slice) GetTagstrSlice() []string {
	result := make([]string, t.Len())
	for i, v := range t {
		result[i] = v.Tagstr
	}
	return result
}

//String concatenates and returns tagstr of tags.
func (t Slice) String() string {
	return strings.Join(t.GetTagstrSlice(), " ")
}

//Prune truncates non-weighted tagList to size=size.
func (t Slice) Prune(size int) Slice {
	sort.Sort(sort.Reverse(t))
	if t.Len() > size {
		t = t[:size]
	}
	return t
}

//checkAppend append tagstr=val tag if tagList doesn't have its tag.
func (t Slice) checkAppend(val string) Slice {
	if strings.ContainsAny(val, "<>&") || util.HasString(t.GetTagstrSlice(), val) {
		return t
	}
	return append(t, &Tag{val, 1})
}

//HasTagstr return true if one of tags has tagstr
func (t Slice) HasTagstr(tagstr string) bool {
	for _, v := range t {
		if v.Tagstr == tagstr {
			return true
		}
	}
	return false
}

//AddString add tagstr=vals tag
func (t Slice) AddString(vals []string) Slice {
	for _, val := range vals {
		t = t.checkAppend(val)
	}
	return t
}

//Sync saves tagstr of tags to path.
func (t Slice) Sync(path string) {
	err := util.WriteSlice(path, t.GetTagstrSlice())
	if err != nil {
		log.Println(err)
	}
}

//GetUserTag reads tags from the disk  if dirty and returns Slice.
func GetUserTag() Slice {
	cfg.Fmutex.RLock()
	defer cfg.Fmutex.RUnlock()
	var tags Slice
	err := util.EachFiles(cfg.CacheDir, func(i os.FileInfo) error {
		fname := path.Join(cfg.CacheDir, i.Name(), "tag.txt")
		if i.IsDir() && util.IsFile(fname) {
			t, err := ioutil.ReadFile(fname)
			if err != nil {
				return err
			}
			tags = tags.AddString(strings.Split(string(t), "\r\n"))
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return tags
}
