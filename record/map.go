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

package record

import (
	"fmt"
	"sort"

	"github.com/boltdb/bolt"
	"github.com/shingetsu-gou/shingetsu-gou/db"
)

//Map is a map key=stamp_id, value=record.
type Map map[string]*Record

const (
	//Alive counts records that are not removed.
	Alive = 1
	//Removed counts records that are  removed.
	Removed = 2
	//All counts all records
	All = 3
)

//FromRecordDB makes record map from record db.
func FromRecordDB(datfile string, kind int) (Map, error) {
	var r []*DB
	err := db.DB.View(func(tx *bolt.Tx) error {
		var err error
		r, err = GetFromDBs(tx, datfile)
		return err
	})
	if err != nil {
		return nil, err
	}
	m := make(Map)
	for _, rr := range r {
		rec := &Record{
			Head: rr.Head,
		}
		idd := fmt.Sprintf("%d_%s", rr.Stamp, rr.ID)
		switch kind {
		case Alive:
			if !rr.Deleted {
				m[idd] = rec
			}
		case Removed:
			if rr.Deleted {
				m[idd] = rec
			}
		case All:
			m[idd] = rec
		}
	}
	return m, nil
}

//Get returns records which hav key=i.
//return def if not found.
func (r Map) Get(i string, def *Record) *Record {
	if v, exist := r[i]; exist {
		return v
	}
	return def
}

//Keys returns key strings(ids) of records
func (r Map) Keys() []string {
	ks := make([]string, len(r))
	i := 0
	for k := range r {
		ks[i] = k
		i++
	}
	sort.Strings(ks)
	return ks
}
