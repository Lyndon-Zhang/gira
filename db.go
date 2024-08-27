package gira

import (
	"database/sql"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	GAMEDB_NAME       = "gamedb"
	RESOURCEDB_NAME   = "resourcedb"
	STATDB_NAME       = "statdb"
	ACCOUNTDB_NAME    = "accountdb"
	LOGDB_NAME        = "logdb"
	BEHAVIORDB_NAME   = "behaviordb"
	ACCOUNTCACHE_NAME = "accountcache"
	GAMECACHE_NAME    = "gamecache"
	ADMINCACHE_NAME   = "admincache"
	ADMINDB_NAME      = "admindb"
)
const (
	MONGODB_NAME = "mongodb"
	REDIS_NAME   = "redis"
	MYSQL_NAME   = "mysql"
)

type DbClient interface {
	Uri() string
}

type RedisClient interface {
	DbClient
	GetRedisClient() *redis.Client
}

type MysqlClient interface {
	DbClient
	GetMysqlClient() *sql.DB
}

type MongoClient interface {
	DbClient
	GetMongoClient() *mongo.Client
	GetMongoDatabase() *mongo.Database
}

type DbClientComponent interface {
	GetGameDbClient() DbClient
	GetStatDbClient() DbClient
	GetAccountDbClient() DbClient
	GetResourceDbClient() DbClient
	GetAdminDbClient() DbClient
	GetAccountCacheClient() DbClient
	GetGameCacheClient() DbClient
	GetLogDbClient() DbClient
	GetBehaviorDbClient() DbClient
	GetAdminCacheClient() DbClient
}

type DbDao interface {
	UseClient(client DbClient) error
}
