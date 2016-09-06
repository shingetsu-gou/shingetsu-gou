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
	"sort"

	"github.com/shingetsu-gou/shingetsu-gou/recentlist"
)

//Caches is a slice of *cache
type Caches []*Cache

//Has return true is Caches has cache cc
func (c Caches) Has(cc *Cache) bool {
	for _, c := range c {
		if c.Datfile == cc.Datfile {
			return true
		}
	}
	return false
}

//Len returns size of cache slice.
func (c Caches) Len() int {
	return len(c)
}

//SortByStamp is for sorting by stamp.
type SortByStamp struct {
	Caches
	stamp []int64
}

//NewSortByStamp makes stamps for caches and returns SortByStamp obj.
func NewSortByStamp(cs Caches, recentStamp bool) *SortByStamp {
	s := &SortByStamp{
		Caches: cs,
		stamp:  make([]int64, cs.Len()),
	}
	for i, v := range cs {
		if recentStamp {
			s.stamp[i] = v.RecentStamp()
		} else {
			s.stamp[i] = v.Stamp()
		}
	}
	return s
}

//Less returns true if cache[i].stamp < cache[j].stamp.
func (c *SortByStamp) Less(i, j int) bool {
	return c.stamp[i] < c.stamp[j]
}

//Swap swaps order of cache slice.
func (c *SortByStamp) Swap(i, j int) {
	c.Caches[i], c.Caches[j] = c.Caches[j], c.Caches[i]
	c.stamp[i], c.stamp[j] = c.stamp[j], c.stamp[i]
}

//SortByVelocity is for sorting by velocity.
type SortByVelocity struct {
	Caches
	velocity []int
	size     []int64
}

//NewSortByVelocity makes velocity for caches and returns SortByVelocity obj.
func NewSortByVelocity(cs Caches) *SortByVelocity {
	s := &SortByVelocity{
		Caches:   cs,
		velocity: make([]int, cs.Len()),
		size:     make([]int64, cs.Len()),
	}
	for i, v := range cs {
		s.velocity[i] = v.Velocity()
		s.size[i] = v.Size()
	}
	return s
}

//Less returns true if cache[i].velocity < cache[j].velocity.
//if velocity[i]==velocity[j],  returns true if cache[i].size< cache[j].size.
func (c *SortByVelocity) Less(i, j int) bool {
	if c.velocity[i] != c.velocity[j] {
		return c.velocity[i] < c.velocity[j]
	}
	return c.size[i] < c.size[j]
}

//Swap swaps order of cache slice.
func (c *SortByVelocity) Swap(i, j int) {
	c.Caches[i], c.Caches[j] = c.Caches[j], c.Caches[i]
	c.velocity[i], c.velocity[j] = c.velocity[j], c.velocity[i]
	c.size[i], c.size[j] = c.size[j], c.size[i]
}

//MakeRecentCachelist returns sorted cachelist copied from Recentlist.
//which doens't contain duplicate Caches.
func MakeRecentCachelist() Caches {
	var cl Caches
	for _, datfile := range recentlist.Datfiles() {
		ca := NewCache(datfile)
		cl = append(cl, ca)
	}
	sort.Sort(sort.Reverse(NewSortByStamp(cl, true)))
	return cl
}
