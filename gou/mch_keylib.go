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
	"strconv"
	"strings"
	"sync"
	"time"
)


type datakeyTable struct {
	file            string
	datakey2filekey map[int64]string
	filekey2datkey  map[string]int64
	mutex           sync.Mutex
}

func newDatakeyTable(file string) *datakeyTable {
	d := &datakeyTable{}
	d.file = file
	d.datakey2filekey = make(map[int64]string)
	d.filekey2datkey = make(map[string]int64)
	return d
}

func (d *datakeyTable) loadInternal() {
	err := eachLine(d.file, func(line string, i int) error {
		dat := strings.Split(strings.TrimSpace(line), "<>")
		stamp, err := strconv.ParseInt(dat[0], 10, 64)
		if err != nil {
			log.Println(err)
			return nil
		}
		d.setEntry(stamp, dat[1])
		return nil
	})
	if err != nil {
		log.Println(err)
	}
}

func (d *datakeyTable) load() {
	d.loadInternal()
	for _, c := range newCacheList().caches {
		c.load()
		d.setFromCache(c)
	}
	for _, rec := range recentList.records {
		c := newCache(rec.datfile)
		c.load()
		c.recentStamp = rec.stamp
		d.setFromCache(c)
	}
	d.save()
}

func (d *datakeyTable) save() {
	str := make([]string, len(d.datakey2filekey))
	i := 0
	for stamp, filekey := range d.datakey2filekey {
		str[i] = fmt.Sprintf("%s<>%s\n", strconv.FormatInt(stamp, 10), filekey)
	}
	err := writeSlice(d.file, str)
	if err != nil {
		log.Println(err)
	}
}

func (d *datakeyTable) setEntry(stamp int64, filekey string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.datakey2filekey[stamp] = filekey
	d.filekey2datkey[filekey] = stamp
}

func (d *datakeyTable) setFromCache(ca *cache) {
	if _, exist := d.filekey2datkey[ca.datfile]; exist {
		return
	}
	var firstStamp int64
	if len(ca.keys()) == 0 {
		firstStamp = ca.recentStamp
	} else {
		if rec := ca.get(ca.keys()[0], nil); rec != nil {
			firstStamp = rec.stamp
		}
	}
	if firstStamp == 0 {
		firstStamp = time.Now().Add(-24 * time.Hour).Unix()
	}
	for {
		if _, exist := d.datakey2filekey[firstStamp]; exist {
			break
		}
		firstStamp++
	}

}

func (d *datakeyTable) getDatkey(filekey string) (int64, error) {
	if v, exist := d.filekey2datkey[filekey]; exist {
		return v, nil
	}
	c := newCache(filekey)
	c.load()
	d.setFromCache(c)
	d.save()
	if v, exist := d.filekey2datkey[filekey]; exist {
		return v, nil
	}
	return -1, errors.New(filekey + " not found")
}

func (d *datakeyTable) getFilekey(datkey string) (string, error) {
	nDatkey, err := strconv.ParseInt(datkey, 10, 64)
	if err != nil {
		log.Println(err)
		return "", err
	}
	if v, exist := d.datakey2filekey[nDatkey]; exist {
		return v, nil
	}
	return "", fmt.Errorf("%s not found", datkey)
}
