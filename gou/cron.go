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

	"github.com/shingetsu-gou/shingetsu-gou/node"
	"github.com/shingetsu-gou/shingetsu-gou/thread"
)

var running bool

//cron runs cron, and update everything if it is after specified cycle.
func cron(nodeManager *node.Manager, recentList *thread.RecentList, heavymoon bool, myself *node.Myself, rundir string) {
	const (
		shortCycle = 10 * time.Minute
		longCycle  = time.Hour
	)

	go func() {
		getall := true
		for {
			log.Println("short cycle cron started")
			ns := node.MustNewNodes(nodeManager.InitNode.GetData())
			if len(ns) == 0 {
				log.Fatal("not init nodes")
			}
			nodes := ns[0].GetherNodes()
			myself.CheckConnection(nodes)
			nodeManager.Initialize(nodes)
			nodeManager.Sync()
			doSync(nodeManager, recentList, heavymoon, getall)
			log.Println("short cycle cron finished")
			getall = false
			<-time.After(shortCycle)
		}
	}()
	go func() {
		for {
			<-time.After(longCycle)
			log.Println("long cycle cron started")
			recentList.Getall(true)
			cl := thread.NewCacheList()
			cl.CleanRecords()
			cl.RemoveRemoved()
			log.Println("long cycle cron finished")
		}
	}()

}

//doSync checks nodes in the nodelist are alive, reloads cachelist, removes old removed files,
//reloads all tags from cachelist,reload srecent list from nodes in search list,
//and reloads cache info from files in the disk.
func doSync(nodeManager *node.Manager, recentList *thread.RecentList, heavymoon bool, fullRecent bool) {
	if nodeManager.ListLen() == 0 {
		return
	}
	log.Println("recentList.getall start")
	recentList.Getall(fullRecent)
	log.Println("recentList.getall finished")

	if heavymoon && !running {
		running = true
		go func() {
			cl := thread.NewCacheList()
			log.Println("cacheList.getall start")
			cl.Getall()
			log.Println("cacheList.getall finished")
			running = false
		}()
	}
}
