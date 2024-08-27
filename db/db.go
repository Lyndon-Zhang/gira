package db

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// 根据配置构造mongodb client
func NewConfigMongoDbClient(ctx context.Context, name string, config gira.DbConfig) (gira.DbClient, error) {
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	uri := config.Uri()
	client := &MongoDbClient{
		config:     config,
		cancelFunc: cancelFunc,
		ctx:        cancelCtx,
		uri:        uri,
	}
	clientOpts := options.Client().ApplyURI(uri)
	if config.ConnnectTimeout > 0 {
		// the default is 30 seconds.
		clientOpts.SetConnectTimeout(config.ConnnectTimeout)
	}
	conn, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		corelog.Errorw("connect database fail", "name", name, "uri", uri, "error", err)
		return nil, err
	}
	// ctx2, cancelFunc2 := context.WithTimeout(client.ctx, config.ConnnectTimeout*time.Second)
	// defer cancelFunc2()
	if err = conn.Ping(ctx, readpref.Primary()); err != nil {
		corelog.Errorw("ping database fail", "name", name, "uri", uri, "error", err)
		return nil, err
	}
	client.client = conn
	corelog.Debugw("connect database success", "name", name, "uri", uri)
	return client, nil
}

// 根据配置构造redis client
func NewConfigRedisClient(ctx context.Context, name string, config gira.DbConfig) (gira.DbClient, error) {
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	uri := config.Uri()
	client := &RedisClient{
		config:     config,
		cancelFunc: cancelFunc,
		ctx:        cancelCtx,
		uri:        uri,
	}
	var err error
	var db int
	db, err = strconv.Atoi(config.Db)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       db,
	})
	ctx1, cancelFunc1 := context.WithTimeout(client.ctx, 3*time.Second)
	defer cancelFunc1()
	if _, err := rdb.Ping(ctx1).Result(); err != nil {
		corelog.Errorw("connect database fail", "name", name, "uri", uri, "error", err)
		return nil, err
	}
	client.client = rdb
	corelog.Debugw("connect database success", "name", name, "uri", uri)
	return client, nil
}

// 根据配置构造mysql client
func NewConfigMysqlClient(ctx context.Context, name string, config gira.DbConfig) (gira.DbClient, error) {
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	uri := config.Uri()
	client := &MysqlClient{
		config:     config,
		cancelFunc: cancelFunc,
		ctx:        cancelCtx,
		uri:        uri,
	}
	db, err := sql.Open("mysql", uri)
	if err != nil {
		corelog.Errorw("connect database fail", "name", name, "uri", uri, "error", err)
		return nil, err
	}
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}
	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	}
	err = db.Ping()
	if err != nil {
		corelog.Errorw("connect database fail", "name", name, "uri", uri, "error", err)
		return nil, err
	}
	client.client = db
	corelog.Debugw("connect database success", "name", name, "uri", uri)
	return client, nil
}

// 根据url构造db client
func NewDbClientFromUri(ctx context.Context, name string, uri string) (gira.DbClient, error) {
	config := gira.DbConfig{}
	if err := config.Parse(uri); err != nil {
		return nil, err
	}
	return NewConfigDbClient(ctx, name, config)
}

// 根据配置构造db client
func NewConfigDbClient(ctx context.Context, name string, config gira.DbConfig) (gira.DbClient, error) {
	switch config.Driver {
	case gira.MONGODB_NAME:
		return NewConfigMongoDbClient(ctx, name, config)
	case gira.REDIS_NAME:
		return NewConfigRedisClient(ctx, name, config)
	case gira.MYSQL_NAME:
		return NewConfigMysqlClient(ctx, name, config)
	default:
		return nil, errors.ErrDbNotSupport
	}
}

// mongodb客户端
type MongoDbClient struct {
	cancelFunc context.CancelFunc
	ctx        context.Context
	client     *mongo.Client
	config     gira.DbConfig
	uri        string
}

func (self *MongoDbClient) GetMongoDatabase() *mongo.Database {
	return self.client.Database(self.config.Db)
}

func (self *MongoDbClient) GetMongoClient() *mongo.Client {
	return self.client
}

// mysql客户端
type MysqlClient struct {
	cancelFunc context.CancelFunc
	ctx        context.Context
	client     *sql.DB
	config     gira.DbConfig
	uri        string
}

func (self *MysqlClient) Uri() string {
	return self.uri
}

func (self *MysqlClient) GetMysqlClient() *sql.DB {
	return self.client
}

// redis客户端
type RedisClient struct {
	cancelFunc context.CancelFunc
	ctx        context.Context
	client     *redis.Client
	config     gira.DbConfig
	uri        string
}

func (self *RedisClient) Uri() string {
	return self.uri
}

func (self *RedisClient) GetRedisClient() *redis.Client {
	return self.client
}

func (self *MongoDbClient) Uri() string {
	return self.uri
}
