package smached

import (
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

/*
	Server statistics
*/
type ServerInfo struct {
	memoryUsage  uint64
	totalRecords int
	cpuUsage     uint64
}

type Config struct {
	MemoryThreshold                    uint64
	EvictionPolicy                     int
	MaxTtl                             string
	DbUsername, DbPass, DbHost, DbPort string
	AuthToken                          string
}

type evictionPolicy struct {
	evictExpirationTime int
	evictLruLeast       int
	evictRandom         int
	evictFILO           int
}

type Record struct {
	Value            interface{}
	hashedValue      string
	created, lastHit time.Time
	hitCount         int
	Expires, ForceDb interface{}
	Ttl              string
}

type AddRequest struct {
	Value            string
	expires, forceDb bool
	ttl              float32
}

type MongoDb struct {
	clientOptions *options.ClientOptions
	client        mongo.Client
	collection    mongo.Collection
}

type PostgresDb struct {
	//TODO
}
