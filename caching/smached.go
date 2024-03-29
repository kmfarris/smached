/**
Copyright 2020 Kelly Farris
kmfarris23@gmail.com
*/

package smached

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/x/mongo/driver/uuid"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var version = "0.0.1"

/*
	This is the memory store.  The heart of darkness.  The beating heart of the..
	Crap.  I already used 'heart'.
	This is the core of the thingy.
*/
var mainCache = make(map[string]Record)

var config = Config{}
var evictionPolicies = evictionPolicy{}

func initEvictionPolicies() {
	evictionPolicies.EVICT_EXPIRATION_TIME = 0
	evictionPolicies.EVICT_FILO = 1
	evictionPolicies.EVICT_RANDOM = 2
	evictionPolicies.EVICT_LRU_LEAST = 3
}

func loadConfig() {
	viper.SetConfigName("config") // name of config.toml file (without extension)
	viper.SetConfigType("toml")   // REQUIRED if the config.toml file does not have the extension in the name
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/smached/")  // path to look for the config.toml file in
	viper.AddConfigPath("$HOME/.smached") // call multiple times to add many search paths
	// optionally look for config.toml.bak in the working directory
	err := viper.ReadInConfig() // Find and read the config.toml.bak file
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			writeDefaultConfig()
		} else {
			// Handle errors reading the config.toml.bak file
			panic(fmt.Errorf("Fatal error config.toml.bak file: %s \n", err))
		}
	}

	//d := viper.Get("default")
	err = viper.Unmarshal(&config)
	log.Printf("Config loaded.")
	if err != nil {
		log.Fatal(err)
	}

}

func writeDefaultConfig() {
	defaultConfig := Config{}
	defaultConfig.MaxTtl = "3m00s"
	defaultConfig.EvictionPolicy = evictionPolicies.EVICT_LRU_LEAST
	defaultConfig.MemoryThreshold = 2048
	uuid, err := uuid.New()
	defaultConfig.AuthToken = getHashedValue(fmt.Sprintf("%d", uuid))

	fmt.Println("No Configuration file was found.  Would you like to create a default config? Y/n :")

	reader := bufio.NewReader(os.Stdin)
	char, _, err := reader.ReadRune()

	if err != nil {
		log.Fatal(err)
	}

	switch strings.ToLower(string(char)) {
	case "n":
		return
		panic(fmt.Errorf("Unable to start without config file.  Please add a config or use the default generated one to continue."))
	default:
		viper.SetDefault("MemoryThreshold", defaultConfig.MemoryThreshold)
		viper.SetDefault("EvictionPolicy", defaultConfig.EvictionPolicy)
		viper.SetDefault("MaxTtl", defaultConfig.MaxTtl)
		viper.SetDefault("AuthToken", defaultConfig.AuthToken)

		err = viper.SafeWriteConfig()
		if err != nil {
			panic(fmt.Errorf("There was an error creating a new config", err))
		}
	}

	config = defaultConfig
}

func showLoadingInfo() {
	log.Printf("Smached: version %v", version)
}

func GetAuthToken() string {
	return config.AuthToken
}

func InitSmached() {
	showLoadingInfo()
	loadConfig()
	initEvictionPolicies()
	go initCronJobs()

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

func getMemoryUsage() uint64 {
	m := runtime.MemStats{}
	runtime.ReadMemStats(&m)
	return m.HeapInuse
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func Find(key string) *Record {
	record := mainCache[key]
	if record.hashedValue == key {
		record.hitCount++
		return &record
	}
	return nil
}

/*
	Shotguns every record.  Useful for seeing every record.
*/
func GetAll() []Record {
	type exportRecords []Record
	s := make(exportRecords, 0, len(mainCache))
	for _, i := range mainCache {
		s = append(s, i)
	}
	return s
}

/*
	If the record already exists, it updates the lastHit time and returns the existing hash.
	Otherwise, it will create the record and perform cleanup checks.
*/
func Add(record Record) (hashedValue string) {
	if len(mainCache) > 0 {
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
func updateRecord(record Record) (hashedValue string) {
	record.lastHit = time.Now()
	record.hitCount += 1
	mainCache[record.hashedValue] = record
	return record.hashedValue
}

func createNewRecord(record Record) (hashedValue string) {
	record.hashedValue = getHashedValue(record.Value)
	record.created = time.Now()
	record.lastHit = time.Now()
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
	if err != nil {
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
func checkMemoryUsage(record *Record) {
	mu := getMemoryUsage()
	if mu > (config.MemoryThreshold * 1024 * 1024) {
		recordUsage := uint64(unsafe.Sizeof(&record))
		memTarget := mu - recordUsage
		rCount := len(mainCache)
		for getMemoryUsage() > memTarget {
			log.Printf("Memory usage triggered cleaning: %d MB \n\r", bToMb(getMemoryUsage()))

			switch config.EvictionPolicy {
			case evictionPolicies.EVICT_RANDOM:
				EvictRandom()
			case evictionPolicies.EVICT_LRU_LEAST:
				EvictByLRU()
			case evictionPolicies.EVICT_FILO:
				EvictByFILO()
			case evictionPolicies.EVICT_EXPIRATION_TIME:
				EvictByExpirationTime()
			default:
				EvictByExpirationTime()
			}
		}
		newCount := len(mainCache)
		log.Printf("Memory usage now at: %d MB \n\r", bToMb(getMemoryUsage()))
		log.Printf("%d records evicted. \n\r", rCount-newCount)
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
	log.Printf("%d records evicted. \n\r", rCount-newCount)
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
