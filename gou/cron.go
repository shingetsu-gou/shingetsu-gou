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

//cron runs a cron job periditically.
func cron() {
	c := newClient()
	for {
		<-connections
		c.run()
		connections <- struct{}{}
		time.Sleep(clientCycle)
	}
}

//client is for updating everything and saves cron status, i.e. save pinged/inited/synced times.
type client struct {
	utime     map[string]time.Time
	timelimit time.Time
}

//newClient read updated time from the file and creates client instance.
func newClient() *client {
	c := &client{utime: make(map[string]time.Time)}
	if !IsFile(clientLog) {
		return c
	}
	k := []string{"ping", "init", "sync"}
	err := eachLine(clientLog, func(line string, i int) error {
		var err error
		t, err := strconv.ParseInt(line, 10, 64)
		c.utime[k[i]] = time.Unix(t, 0)
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
	return c
}

//sync saves updated times.
func (c *client) sync() {
	f, err := os.Create(clientLog)
	if err != nil {
		log.Fatal(err)
	}
	defer fclose(f)
	for _, v := range []string{"ping", "init", "sync"} {
		_, err := f.WriteString(strconv.FormatInt(c.utime[v].Unix(), 10) + "\n")
		if err != nil {
			log.Println(err)
		}
	}
}

//check updates updated time.
func (c *client) check(key string) {
	c.utime[key] = time.Now()
	c.sync()
}

//run runs cron, and update everything if it is after specified cycle.
func (c *client) run() {
	log.Println("starting cron...")
	t := time.Now()
	c.timelimit = t.Add(clientTimeout)
	if t.Sub(c.utime["ping"]) >= pingCycle {
		c.check("ping")
		nodeList.pingAll()
		nodeList.sync()
		log.Println("nodelist.pingall finished")

		queue.run()
		log.Println("updatequeue finished")
	}
	if nodeList.Len() == 0 {
		c.doInit()
		if nodeList.Len() != 0 {
			c.doSync()
		}
	}

	if t.Sub(c.utime["init"]) >= initCycle*time.Duration(nodeList.Len()) {
		c.doInit()
	} else {
		if nodeList.Len() < defaultNodes {
			nodeList.rejoin(searchList)
			log.Println("nodelist.rejoin finished")
		}
	}
	if t.Sub(c.utime["sync"]) >= syncCycle {
		c.doSync()
	}
	log.Println("cron finished")
}

//doInit tries to find nodes from initNode and also add them to the search list.
func (c *client) doInit() {
	c.check("init")
	nodeList.initialize()
	nodeList.sync()
	searchList.extend(nodeList.nodes)
	searchList.sync()
	log.Println("nodelist.init finished")
}

//doSync checks nodes in the nodelist are alive, reloads cachelist, removes old removed files,
//reloads all tags from cachelist,reload srecent list from nodes in search list,
//and reloads cache info from files in the disk.
func (c *client) doSync() {
	c.check("sync")
	for _, n := range nodeList.nodes {
		nodeList.join(n)
	}
	nodeList.sync()
	log.Println("nodelist.join finished")

	cl := newCacheList()
	cl.cleanRecords()
	log.Println("cachelist.cleanRecords finished")

	cl.removeRemoved()
	log.Println("cachelist.removeRemoved finished")

	userTagList.updateAll()
	log.Println("userTagList.updateAll finished")

	recentList.getAll()
	log.Println("recentList.getall finished")

	cl.getall(c.timelimit)
	log.Println("cacheList.getall finished")
}
