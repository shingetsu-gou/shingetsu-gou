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

package user

import (
	"log"

	"github.com/shingetsu-gou/shingetsu-gou/db"
	"github.com/shingetsu-gou/shingetsu-gou/tag"
)

//String  returns string form of usertags.
func String(thread string) string {
	tags := GetByThread(thread)
	return tags.String()
}

//Len  returns # of usertags.
func Len(thread string) int {
	r, err := db.GetMap("usertag", []byte(thread))
	if err != nil {
		log.Print(err)
		return 0
	}
	return len(r)
}

//Has returns true if thread has the tag.
func Has(thread string, tag ...string) bool {
	for _, t := range tag {
		if r := db.HasVal("usertag", []byte(thread), t); r {
			return true
		}
	}
	return false
}

//Get tags from the disk and returns Slice.
func Get() tag.Slice {
	r, err := db.KeyStrings("usertagTag")
	if err != nil {
		log.Print(err)
		return nil
	}
	return tag.NewSlice(r)
}

//GetStrings gets thread tags from the disk
func GetStrings(thread string) []string {
	r, err := db.MapKeys("usergag", []byte(thread))
	if err != nil {
		log.Print(err)
		return nil
	}
	return r
}

//GetByThread gets thread tags from the disk
func GetByThread(thread string) tag.Slice {
	r := GetStrings(thread)
	return tag.NewSlice(r)
}

//Add saves tag strings.
func Add(thread string, tag []string) {
	for _, t := range tag {
		if err := db.PutMap("usertag", []byte(thread), t); err != nil {
			log.Print(err)
		}
		if err := db.PutMap("usertagTag", []byte(t), thread); err != nil {
			log.Print(err)
		}
	}
}

//AddTags saves tag slice..
func AddTags(thread string, tag tag.Slice) {
	Add(thread, tag.GetTagstrSlice())
}

//Set remove all tags and saves tag strings.
func Set(thread string, tag []string) {
	ts, err := db.GetMap("usertag", []byte(thread))
	if err != nil {
		log.Print(err)
		return
	}
	for t := range ts {
		err = db.DelMap("usertagTag", []byte(t), thread)
		if err != nil {
			log.Print(err)
		}
	}
	err = db.Del("usertag", []byte(thread))
	if err != nil {
		log.Print(err)
		return
	}
	Add(thread, tag)
}

//SetTags remove all tags and saves tag slice.
func SetTags(thread string, tag tag.Slice) {
	Set(thread, tag.GetTagstrSlice())
}
