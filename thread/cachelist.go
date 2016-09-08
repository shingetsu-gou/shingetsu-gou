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
	"log"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/db"
)

//AllCaches returns all  thread names
func AllCaches() Caches {
	var r []string
	r, err := db.Strings("select Thread from thread group by Thread ")
	if err != nil {
		log.Print(err)
		return nil
	}
	ca := make(Caches, len(r))
	for i, t := range r {
		ca[i] = NewCache(t)
	}
	return ca
}

//Len returns # of Caches
func Len() int {
	r, err := db.Int64("select count(*) from record group by Thread")
	if err != nil {
		log.Print(err)
		return 0
	}

	return int(r)
}

//Search reloads records in Caches in cachelist
//and returns slice of cache which matches query.
func Search(q string) Caches {
	r, err := db.Strings("select Thread from record where Body like ? group by Thread", q)
	if err != nil {
		log.Print(err)
		return nil
	}
	result := make([]*Cache, len(r))

	for i, rr := range r {
		result[i] = NewCache(rr)
	}
	return result
}

//CleanRecords remove old or duplicates records for each Caches.
func CleanRecords() {
	l := int64(Len())
	if l > cfg.SaveRecord {
		_, err := db.DB.Exec("delete from record  order by Stamp limit ? ", l-cfg.SaveRecord)
		if err != nil {
			log.Println(err)
		}
	}
	_, err := db.DB.Exec("update record set Deleted=1 where  Stamp =?", time.Now().Unix()-cfg.SaveRecord)
	if err != nil {
		log.Println(err)
	}
}

//RemoveRemoved removes files in removed dir if old.
func RemoveRemoved() {
	if cfg.SaveRemoved > 0 {
		return
	}
	_, err := db.DB.Exec("delete from record where Deleted=1  and Stamp <? ", time.Now().Unix()-cfg.SaveRemoved)
	if err != nil {
		log.Println(err)
	}
}
