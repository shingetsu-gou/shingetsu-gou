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
	"os"
	"strconv"
	"time"
)

func cron() {
	for {
		time.Sleep(client_cycle)
		c:=client{}
		go c.run()
	}
}

type status struct {
	dict map[string]int64
}

func newStatus() *status {
	s := &status{}
	if !isFile(client_log) {
		return s
	}
	k := []string{"ping", "init", "sync"}
	err := eachLine(client_log, func(line string, i int) error {
		var err error
		s.dict[k[i]], err = strconv.ParseInt(line, 10, 64)
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
	return s
}

func (s *status) sync() {
	f, err := os.Create(client_log)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	for _, v := range []string{"ping", "init", "sync"} {
		f.WriteString(strconv.FormatInt(s.dict[v], 10) + "\n")
	}
}

func (s *status) check(key string) {
	s.dict[key] = time.Now().Unix()
}

type client struct {
	timelimit int64
}

func (c *client) run() {
	s := newStatus()
	c.timelimit = time.Now().Add(client_timeout).Unix()
	if time.Now().Unix()-s.dict["ping"] >= int64(ping_cycle) {
		c.doPing()
		s = newStatus()
		c.doUpdate()
	}
	nl := newNodeList()
	if nl.Len() == 0 {
		c.doInit()
		nl = newNodeList()
		if nl != nil {
			c.doSync()
		}
		s = newStatus()
	}

	if time.Now().Unix()-s.dict["init"] > int64(init_cycle)*int64(nl.Len()) {
		c.doInit()
		s = newStatus()
	} else {
		if nl.Len() < default_nodes {
			c.doRejoin()
			s = newStatus()
		}
	}
	if time.Now().Unix()-s.dict["sync"] >= int64(sync_cycle) {
		c.doSync()
	}
}

func (c *client) check(key string) {
	s := newStatus()
	s.check(key)
	s.sync()
}

func (c *client) doPing() {
	c.check("ping")
	nl := newNodeList()
	nl.pingAll()
	nl.sync()
	log.Println("nodelist.pingall finished")
}

func (c *client) doUpdate() {
	q := newUpdateQue()
	go q.run()
	log.Println("updatequeue started")
}

func (c *client) doInit() {
	c.check("init")
	nl := newNodeList()
	nl.initialize()
	nl.sync()
	sl := newSearchList()
	sl.extend(nl.tiedlist)
	sl.sync()
	log.Println("nodelist.init finished")
}

func (c *client) doRejoin() {
	nl := newNodeList()
	sl := newSearchList()
	nl.rejoin(sl)
	log.Println("nodelist.rejoin finished")
}

func (c *client) doSync() {
	c.check("sync")
	nl := newNodeList()
	for _, n := range nl.tiedlist {
		nl.join(n)
	}
	nl.sync()
	log.Println("nodelist.josin finished")

	cl := newCacheList()
	cl.rehash()
	log.Println("cachelist.rehash finished")

	cl.cleanRecords()
	log.Println("cachelist.cleanRecords finished")

	cl.removeRemoved()
	log.Println("cachelist.removeRemoved finished")

	ut := newUserTagList()
	ut.updateAll()
	log.Println("userTagList.updateAll finished")

	rl := newRecentList()
	rl.getAll()
	log.Println("recentList.getall finished")

	cl.getall(c.timelimit)
	log.Println("cacheList.getall finished")
}
