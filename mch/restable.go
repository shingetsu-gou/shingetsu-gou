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

package mch

import (
	"regexp"
	"strconv"

	"github.com/shingetsu-gou/shingetsu-gou/thread"
)

//ResTable maps id[:8] and its number.
type ResTable struct {
	ID2num map[string]int
	Num2id []string
}

//NewResTable creates ane returns a resTable instance.
func NewResTable(ca *thread.Cache) *ResTable {
	r := &ResTable{
		make(map[string]int),
		make([]string, ca.ReadInfo().Len+1),
	}
	recs := ca.LoadRecords(thread.Alive)
	for i, k := range recs.Keys() {
		rec := recs.Get(k, nil)
		r.Num2id[i+1] = rec.ID[:8]
		r.ID2num[rec.ID[:8]] = i + 1
	}
	return r
}

//MakeRSSAnchor replaces id to the record number in body.
func (table *ResTable) MakeRSSAnchor(body string) string {
	reg := regexp.MustCompile("&gt;&gt;([0-9a-f]{8})")
	return reg.ReplaceAllStringFunc(body, func(str string) string {
		id := reg.FindStringSubmatch(str)[1]
		no := table.ID2num[id]
		return "&gt;&gt;" + strconv.Itoa(no)
	})
}
