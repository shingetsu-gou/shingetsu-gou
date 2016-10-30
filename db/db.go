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

package db

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"path"

	"encoding/json"

	"github.com/boltdb/bolt"
	"github.com/shingetsu-gou/shingetsu-gou/cfg"
)

/*
bucket key value

keylibST Stamp Thread
keylibTS Thread Stamp
lookupT Thread json(map[addr]struct{})
lookupA Addr json(map[threads]struct{})
thread Thread ""
sugtag Thread json(map[tags]struct{})
usertag Thread json(map[tags]struct{})
usertagTag Tag json(map[threads]struct{})
recent thread:stamp:hash json(Datfile,Stamp.ID)
recentS stamp json(map[thread,stamp.hash]struct{})
record thread:stamp:hash json(Datfile,Stamp.ID,Body,Deleted)
recordS stamp json(map[thread,stamp.hash]struct{})


var tables = []string{
	`CREATE TABLE IF NOT EXISTS  "keylib" ("ID" integer not null primary key autoincrement, "Stamp" integer, "Thread" varchar(255) unique)`,
	`CREATE TABLE IF NOT EXISTS  "lookup" ("ID" integer not null primary key autoincrement, "Thread" varchar(255), "Addr" varchar(255), unique ("Addr", "Thread"))`,
	`CREATE TABLE IF NOT EXISTS  "recent" ("ID" integer not null primary key autoincrement, "Stamp" integer, "Hash" varchar(255), "Thread" varchar(255), unique ("Stamp", "Thread", "Hash"))`,
	`CREATE TABLE IF NOT EXISTS  "sugtag" ("ID" integer not null primary key autoincrement, "Thread" varchar(255), "Tag" varchar(255), unique ("Thread", "Tag"))`,
	`CREATE TABLE IF NOT EXISTS  "thread" ("ID" integer not null primary key autoincrement, "Thread" varchar(255) unique)`,
	`CREATE TABLE IF NOT EXISTS  "record" ("ID" integer not null primary key autoincrement, "Stamp" integer, "Hash" varchar(255), "Thread" varchar(255), "Body" varchar(255), "Deleted" integer, unique ("Stamp", "Hash", "Thread"))`,
	`CREATE TABLE IF NOT EXISTS  "usertag" ("ID" integer not null primary key autoincrement, "Thread" varchar(255), "Tag" varchar(255), unique ("Tag", "Thread"))`,
}
*/

//DB is bolt.DB for operating database.
var DB *bolt.DB

//Setup setups db.
func Setup() {
	dbpath := path.Join(cfg.RunDir, "gou_bolt.db")
	var err error
	DB, err = bolt.Open(dbpath, 0644, nil)
	if err != nil {
		log.Fatal(err)
	}
}

// Tob returns an 8-byte big endian representation of v.
func Tob(v interface{}) ([]byte, error) {
	switch t := v.(type) {
	case []byte:
		return t, nil
	case string:
		return []byte(t), nil
	case int:
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(t))
		return b, nil
	case int64:
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(t))
		return b, nil
	case uint64:
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, t)
		return b, nil
	default:
		return json.Marshal(v)
	}
}

//MustTob is Tob , except that this fatals when error.
func MustTob(v interface{}) []byte {
	b, err := Tob(v)
	if err != nil {
		log.Fatal(err)
	}
	return b
}

//ToKey makes key of db from v.
func ToKey(v ...interface{}) []byte {
	var r []byte
	for _, vv := range v {
		b := MustTob(v)
		r = append(r, b...)
		if _, ok := vv.(string); ok {
			r = append(r, '\x00')
		}
	}
	return r
}

//b2v converts from 'from' to 'to' according to 'to' type.
func b2v(from []byte, to interface{}) error {
	var err error
	switch t := to.(type) {
	case *string:
		(*t) = string(from)
	case *int64:
		(*t) = int64(binary.BigEndian.Uint64(from))
	case *int:
		(*t) = int(binary.BigEndian.Uint64(from))
	case *uint64:
		(*t) = binary.BigEndian.Uint64(from)
	case nil:
		//fallthrough
	default:
		err = json.Unmarshal(from, to)
	}
	return err
}

//Get gets one value from db and converts it to value type.
func Get(tx *bolt.Tx, bucket string, key []byte, value interface{}) ([]byte, error) {
	b := tx.Bucket([]byte(bucket))
	if b == nil {
		return nil, errors.New("bucket not found")
	}
	v := b.Get(key)
	if v == nil {
		return nil, errors.New("key not found")
	}
	return v, b2v(v, value)
}

//Put sets one key/value pair.
func Put(tx *bolt.Tx, bucket string, key []byte, value interface{}) error {
	val, err := Tob(value)
	if err != nil {
		return err
	}

	b, errr := tx.CreateBucketIfNotExists([]byte(bucket))
	if errr != nil {
		return fmt.Errorf("create bucket: %s", err)
	}
	return b.Put(key, val)
}

//HasKey returns true if db has key.
func HasKey(tx *bolt.Tx, bucket string, key []byte) (bool, error) {
	var v []byte
	b := tx.Bucket([]byte(bucket))
	if b == nil {
		return false, errors.New("bucket not found")
	}
	v = b.Get(key)
	return v != nil, nil
}

//Get1st gets 1st value whose key has prefix from db and converts it to value type.
func Get1st(tx *bolt.Tx, bucket string, prefix []byte, value interface{}) ([]byte, error) {
	b := tx.Bucket([]byte(bucket))
	if b == nil {
		return nil, errors.New("bucket not found")
	}
	c := b.Cursor()
	k, v := c.Seek(prefix)
	if bytes.HasPrefix(k, prefix) {
		return v, b2v(v, value)
	}
	return nil, errors.New("no data")
}

//Count counts #data whose key has prefix.
func Count(tx *bolt.Tx, bucket string, prefix []byte) (int, error) {
	var cnt int
	b := tx.Bucket([]byte(bucket))
	if b == nil {
		return 0, errors.New("bucket not found")
	}
	c := b.Cursor()
	for k, _ := c.Seek(prefix); bytes.HasPrefix(k, prefix); k, _ = c.Next() {
		cnt++
	}
	return cnt, nil
}

//GetStrings returns string values whose key has prefix.
func GetStrings(tx *bolt.Tx, bucket string, prefix []byte) ([]string, error) {
	var cnt []string
	b := tx.Bucket([]byte(bucket))
	if b == nil {
		return nil, errors.New("bucket not found")
	}
	c := b.Cursor()
	for k, v := c.Seek(prefix); bytes.HasPrefix(k, prefix); k, v = c.Next() {
		var str string
		if err := b2v(v, &str); err != nil {
			return nil, err
		}
		cnt = append(cnt, str)
	}
	return cnt, nil
}

//KeyStrings returns string keys.
func KeyStrings(tx *bolt.Tx, bucket string) ([]string, error) {
	var cnt []string
	b := tx.Bucket([]byte(bucket))
	if b == nil {
		return nil, errors.New("bucket not found")
	}
	err := b.ForEach(func(k, v []byte) error {
		var str string
		if err := b2v(k, &str); err != nil {
			return err
		}
		cnt = append(cnt, str)
		return nil
	})
	return cnt, err
}

//Del deletes one key-value pair.
func Del(tx *bolt.Tx, bucket string, key []byte) error {
	b := tx.Bucket([]byte(bucket))
	if b == nil {
		return errors.New("bucket not found")
	}
	return b.Delete(key)
}

//GetMap gets map[string]struct{} value.
func GetMap(tx *bolt.Tx, bucket string, key []byte) (map[string]struct{}, error) {
	var rs map[string]struct{}
	_, err := Get(tx, bucket, key, &rs)
	return rs, err
}

//PutMap adds val to map[string]struct{} type value.
func PutMap(tx *bolt.Tx, bucket string, key []byte, val string) error {
	rs, err := GetMap(tx, bucket, key)
	if err != nil {
		rs = make(map[string]struct{})
	}
	rs[val] = struct{}{}
	return Put(tx, bucket, key, rs)
}

//DelMap deletes val from map[string]struct{} type value.
func DelMap(tx *bolt.Tx, bucket string, key []byte, val string) error {
	rs, err := GetMap(tx, bucket, key)
	if err != nil {
		return err
	}
	delete(rs, val)
	if len(rs) == 0 {
		return Del(tx, bucket, key)
	}
	return Put(tx, bucket, key, rs)
}

//MapKeys returns []string from keys of map[string]struct{} type value
func MapKeys(tx *bolt.Tx, bucket string, key []byte) ([]string, error) {
	m, err := GetMap(tx, bucket, key)
	if err != nil {
		return nil, err
	}
	r := make([]string, len(m))
	i := 0
	for k := range m {
		r[i] = k
		i++
	}
	return r, nil
}

//HasVal returns true if map[string]struct{} type values has val.
func HasVal(tx *bolt.Tx, bucket string, key []byte, val string) bool {
	m, err := GetMap(tx, bucket, key)
	if err != nil {
		return false
	}
	_, exist := m[val]
	return exist
}

//GetPrefixs get string prefixs of keys.
func GetPrefixs(tx *bolt.Tx, bucket string) ([]string, error) {
	var cnt []string
	var last string
	var blast []byte
	b := tx.Bucket([]byte(bucket))
	if b == nil {
		return nil, errors.New("bucket not found")
	}
	err := b.ForEach(func(k, v []byte) error {
		if bytes.HasPrefix(k, blast) {
			return nil
		}
		loc := bytes.IndexByte(k, 0x00)
		if loc == -1 {
			return errors.New("not have string prefix")
		}
		last = string(k[:loc-1])
		cnt = append(cnt, last)
		blast := make([]byte, len(last)+1)
		copy(blast, k[:loc])
		return nil
	})
	return cnt, err
}
