package smached

import (

	"math/rand"
	"runtime/debug"
	"sort"
	"time"
)

func EvictRandom() {
	var mapKeys []string

	for _, i := range mainCache {
		mapKeys = append(mapKeys, i.hashedValue)
	}
	eId := rand.Intn(len(mapKeys))
	delete(mainCache, mapKeys[eId])
	debug.FreeOSMemory()

}

func EvictByLRU(){
	sortedCache := sortCacheByHits()
	delete(mainCache, sortedCache[0].hashedValue)
	sortedCache = nil
	debug.FreeOSMemory()
}

func EvictByExpirationTime() {
	// can be slow if waiting on records to expire.
	// utilize fallback policy to regain memory
	for _, i := range mainCache{
		if i.Expires == true {
			diff := time.Now().Sub(i.created)
			maxTtl := config.maxTTL
			if i.Ttl != ""{
				maxTtl = i.Ttl
			}
			ttl, _ := time.ParseDuration(maxTtl)
			if  diff.Seconds() > ttl.Seconds() {
				delete(mainCache, i.hashedValue)
				debug.FreeOSMemory()
			}
		}
	}

}

func EvictByFILO(){
	r := sortCacheByCreatedTime()
	delete(mainCache,r[0].hashedValue)
	r = nil
	debug.FreeOSMemory()
}


func sortCacheByHits() ByHitCountAndHitTime{
	s:=make(ByHitCountAndHitTime,0,len(mainCache))
	for _,d := range mainCache{
		s = append(s, d)
	}
	sort.Reverse(s)
	return s
}

func sortCacheByCreatedTime() ByCreatedTime{
	s:=make(ByCreatedTime,0,len(mainCache))
	for _,d := range mainCache{
		s = append(s, d)
	}
	sort.Sort(s)
	return s
}

type ByHitCountAndHitTime []Record
type ByCreatedTime []Record

func (a ByHitCountAndHitTime) Len() int {
	return len(a)
}
func (a ByHitCountAndHitTime) Swap(i, j int){
	a[i], a[j] =  a[j], a[i]
}

func (a ByHitCountAndHitTime) Less(i, j int) bool{
	return a[i].hitCount > a[j].hitCount && a[i].lastHit.After(a[j].lastHit)
}

func (a ByCreatedTime) Len() int {
	return len(a)
}
func (a ByCreatedTime) Swap(i, j int){
	a[i], a[j] =  a[j], a[i]
}

func (a ByCreatedTime) Less(i, j int) bool{
	return a[i].created.Before(a[j].created)
}



