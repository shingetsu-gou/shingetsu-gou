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
	"time"
)

//cron runs cron, and update everything if it is after specified cycle.
func cron() {
	nodeManager.initialize()
	doSync()

	for {
		select {
		case <-time.After(clientCycle):
			nodeManager.rejoin()

		case <-time.After(pingCycle):
			nodeManager.pingAll()
			nodeManager.initialize()
			nodeManager.sync()
			doSync()
			log.Println("nodelist.pingall finished")

		case <-time.After(initCycle * time.Duration(nodeManager.listLen())):
			nodeManager.initialize()

		case <-time.After(syncCycle):
			doSync()
		}
	}
}

//doSync checks nodes in the nodelist are alive, reloads cachelist, removes old removed files,
//reloads all tags from cachelist,reload srecent list from nodes in search list,
//and reloads cache info from files in the disk.
func doSync() {
	if nodeManager.listLen() == 0 {
		return
	}
	nodeManager.rejoinList()
	log.Println("lookupTable.join finished")

	nodeManager.sync()
	log.Println("lookupTable.join finished")

	cl := newCacheList()
	cl.cleanRecords()
	log.Println("cachelist.cleanRecords finished")

	cl.removeRemoved()
	log.Println("cachelist.removeRemoved finished")

	recentList.getAll()
	log.Println("recentList.getall finished")

	cl.getall()
	log.Println("cacheList.getall finished")
}
