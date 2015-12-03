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
	"crypto/md5"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//RecordHead represents one line in updatelist/recentlist
type RecordHead struct {
	Datfile string //cache file name
	Stamp   int64  //unixtime
	ID      string //md5(bodystr)
}

//newUpdateInfoFromLine parse one line in udpate/recent list and returns updateInfo obj.
func newRecordHeadFromLine(line string) (*RecordHead, error) {
	strs := strings.Split(strings.TrimRight(line, "\n\r"), "<>")
	if len(strs) < 3 || util.FileDecode(strs[2]) == "" || !strings.HasPrefix(strs[2], "thread") {
		err := errors.New("illegal format")
		log.Println(err)
		return nil, err
	}
	u := &RecordHead{
		ID:      strs[1],
		Datfile: strs[2],
	}
	var err error
	u.Stamp, err = strconv.ParseInt(strs[0], 10, 64)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return u, nil
}

//equals returns true if u=v
func (u *RecordHead) equals(rec *RecordHead) bool {
	return u.Datfile == rec.Datfile && u.ID == rec.ID && u.Stamp == rec.Stamp
}

//hash returns md5 of RecordHead.
func (u *RecordHead) hash() [16]byte {
	return md5.Sum([]byte(u.Recstr()))
}

//Recstr returns one line of update/recentlist file.
func (u *RecordHead) Recstr() string {
	return fmt.Sprintf("%d<>%s<>%s", u.Stamp, u.ID, u.Datfile)
}

//Idstr returns real file name of the record file.
func (u *RecordHead) Idstr() string {
	return fmt.Sprintf("%d_%s", u.Stamp, u.ID)
}

type recordHeads []*RecordHead

//Less returns true if stamp of infos[i] < [j]
func (r recordHeads) Less(i, j int) bool {
	return r[i].Stamp < r[j].Stamp
}

//Swap swaps infos order.
func (r recordHeads) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

//Len returns size of infos
func (r recordHeads) Len() int {
	return len(r)
}

//has returns true if recordHeads has rec.
func (r recordHeads) has(rec *RecordHead) bool {
	for _, v := range r {
		if v.equals(rec) {
			return true
		}
	}
	return false
}

//RecordMap is a map key=stamp_id, value=record.
type RecordMap map[string]*Record

//Get returns records which hav key=i.
//return def if not found.
func (r RecordMap) Get(i string, def *Record) *Record {
	if v, exist := r[i]; exist {
		return v
	}
	return def
}

//Keys returns key strings(ids) of records
func (r RecordMap) Keys() []string {
	ks := make([]string, len(r))
	i := 0
	for k := range r {
		ks[i] = k
		i++
	}
	sort.Strings(ks)
	return ks
}

//removeRecords remove old records while remaing #saveSize records.
//and also removes duplicates recs.
func (r RecordMap) removeRecords(limit int64, saveSize int) {
	ids := r.Keys()
	if saveSize < len(ids) {
		ids = ids[:len(ids)-saveSize]
		if limit > 0 {
			for _, re := range ids {
				if r[re].Stamp+limit < time.Now().Unix() {
					err := r[re].Remove()
					if err != nil {
						log.Println(err)
					}
					delete(r, re)
				}
			}
		}
	}
	once := make(map[string]struct{})
	for k, rec := range r {
		if util.IsFile(rec.path()) {
			if _, exist := once[rec.ID]; exist {
				err := rec.Remove()
				if err != nil {
					log.Println(err)
				}
				delete(r, k)
			} else {
				once[rec.ID] = struct{}{}
			}
		}
	}
}

func parseHeadResponse(res []string) RecordMap {
	m := make(RecordMap)
	for _, r := range res {
		if rec := makeRecord(r); rec != nil {
			m[rec.Recstr()] = rec
		}
	}
	return m	
}
