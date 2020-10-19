/**
	Copyright 2020 Kelly Farris
	kmfarris23@gmail.com
 */

package smached

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"runtime"
	"strconv"
	"unsafe"

	//"os"
	//"log"
	"time"
)

/*
	This is the memory store.  The heart of darkness.  The beating heart of the..
	Crap.  I already used 'heart'.
	This is the core of the thingy.
 */
var mainCache = make(map[string]Record)


type Config struct{
	memoryThreshold uint64
	evictionPolicy int
	maxTTL string
}

type evictionPolicy struct {
	evictExpirationTime int
	evictLruLeast int
	evictRandom int
	evictFILO int
}

type Record struct{
	Value interface{}
	hashedValue string
	created, lastHit time.Time
	hitCount int
	Expires, ForceDb interface{}
	Ttl string
}

type AddRequest struct{
	Value string
	expires, forceDb bool
	ttl float32
}

type DbStuffs struct {
	clientOptions *options.ClientOptions
	client mongo.Client
	collection mongo.Collection
}

var config = Config{}
var evictionPolicies = evictionPolicy{}

func initEvictionPolicies() {
	evictionPolicies.evictExpirationTime=0
	evictionPolicies.evictFILO=1
	evictionPolicies.evictRandom=2
	evictionPolicies.evictLruLeast=3
}

func InitSmached() {
	initEvictionPolicies()
	config.memoryThreshold = 3
	config.maxTTL = "30s"
	config.evictionPolicy = evictionPolicies.evictFILO


	//dbStuffs := DbStuffs{
	//	clientOptions: options.Client().ApplyURI("monogdb://localhost:27017"),
	//}
	//c := dbStuffs.client: mongo.Connect(context.TODO(),dbStuffs.clientOptions)
	//collection := dbStuffs.collection: c.Database("smached").Collection("main_cache")
	//clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")

	//client, err := mongo.Connect(context.TODO(), clientOptions)
	//
	//if err != nil{
	//	log.Fatal(err)
	//}
	//
	//err = client.Ping(context.TODO(), nil)
	//
	//if err != nil{
	//	log.Fatal(err)
	//}
	//
	//fmt.Println("Connected to mongodb")
	//
	//if err != nil{
	//	log.Fatal(err)
	//}
	//
	//fmt.Println("Connection to mongodb closed")


	//test()

	//fmt.Println(mainCache)

	//fmt.Println("Inserted document ", insertMany.InsertedIDs)
	//err  = client.Disconnect(context.TODO())

	//update(trainers, dbStuffs)
}

func ShowServerStats() (uint64, int) {
	return getMemoryUsage(), len(mainCache)
}

func getMemoryUsage() (uint64) {
	m:= runtime.MemStats{}
	runtime.ReadMemStats(&m)
	return m.HeapInuse
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}


func Find(key string) *Record {
	record := mainCache[key]
	if record.hashedValue == key {
		record.hitCount ++
		return &record
	}
	return  nil
}

/*
	Shotguns every record.  Useful for seeing every record.
 */
func GetAll() []Record {
	type exportRecords []Record
	s := make(exportRecords,0,len(mainCache))
	for _, i := range mainCache{
		s = append(s, i)
	}
	return s
}

/*
	If the record already exists, it updates the lastHit time and returns the existing hash.
	Otherwise, it will create the record and perform cleanup checks.
 */
func Add(record Record) (hashedValue string){
	if len(mainCache) >0{
		result := Find(getHashedValue(record.Value))
		if result == nil {
			return createNewRecord(record)
		} else {
			return updateRecord(*result)
		}
	}
	return createNewRecord(record)
}

/*
	Updates the record with a hitCount, lastHit time update and returns the hashedValue.
 */
func updateRecord(record Record) (hashedValue string){
	record.lastHit=time.Now()
	record.hitCount +=1
	mainCache[record.hashedValue] = record
	return record.hashedValue
}

func createNewRecord(record Record) (hashedValue string) {
	record.hashedValue = getHashedValue(record.Value)
	record.created =  time.Now()
	record.lastHit =  time.Now()
	if record.Expires == nil {
		record.Expires = true
	} else {
		record.Expires, _ = strconv.ParseBool(fmt.Sprintf("%v", record.Expires))
	}
	mainCache[record.hashedValue] = record
	go cleanCache(record)
	return record.hashedValue
}

func getHashedValue(value interface{}) (hashedValue string) {
	encodedValue, err := json.Marshal(value)
	if err != nil{
		log.Fatal(err)
		return
	}
	hash := md5.New()
	hash.Size()
	hash.Write(encodedValue)
	return hex.EncodeToString(hash.Sum(nil))
}

func cleanCache(record Record) {
	cleanExpiredRecords()
	checkMemoryUsage(&record)
}

/*
	 This will determine a memory target based on the size of the incoming data.
	 One record at a time will be evicted based on the evictionPolicy until
	 the memory usage target has been reached.
 */
func checkMemoryUsage(record *Record){

	mu := getMemoryUsage()
	if  mu > (config.memoryThreshold * 1024 * 1024) {
		recordUsage := uint64(unsafe.Sizeof(&record))
		memTarget := mu - recordUsage
		rCount := len(mainCache)
		for getMemoryUsage() > memTarget {
			log.Printf("Memory usage triggered cleaning: %d MB \n\r", bToMb(getMemoryUsage()))

			switch config.evictionPolicy {
			case evictionPolicies.evictRandom:
				EvictRandom()
			case evictionPolicies.evictLruLeast:
				EvictByLRU()
			case evictionPolicies.evictFILO:
				EvictByFILO()
			case evictionPolicies.evictExpirationTime:
				EvictByExpirationTime()
			default:
				EvictByExpirationTime()
			}
		}
		newCount := len(mainCache)
		log.Printf("Memory usage now at: %d MB \n\r", bToMb(getMemoryUsage()))
		log.Printf("%d records evicted. \n\r", rCount - newCount)
	}
}


/*
	This will evict records regardless of current memory usage.
 */
func cleanExpiredRecords() {
	rCount := len(mainCache)
	EvictByExpirationTime()
	newCount := len(mainCache)
	log.Printf("Memory usage now at: %d MB \n\r", bToMb(getMemoryUsage()))
	log.Printf("%d records evicted. \n\r", rCount - newCount)
}





//func update(records interface{}, db DbStuffs)  {
//	filter := bson.D{{"name",records.Name}}
//	update := bson.D{
//		{"$inc", bson.D{
//			{"age",1},
//		}},
//	}
//	result, err := db.collection.UpdateMany(context.TODO(), filter, update)
//	if err != nil{
//		log.Fatal(err)
//	}
//	fmt.Println("Updated records", result.UpsertedID)
//}