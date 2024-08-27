package behavior

import (
	"sync/atomic"
	"time"

	"github.com/Lyndon-Zhang/gira/facade"
)

var seqId int64

// 生成批次id
func GetSeqId() int64 {
	seqId = atomic.AddInt64(&seqId, 1)
	return time.Now().Unix()<<32 + int64(facade.GetAppId())<<24 + (seqId & 0xffffff)
}

// 设置允许drop index
func WithMigrateDropIndex(enabled bool) MigrateDropIndex {
	return MigrateDropIndex{
		enabled: enabled,
	}
}

// 设置连接超时时间
func WithMigrateConnectTimeout(timeout int64) MigrateConnectTimeout {
	return MigrateConnectTimeout{
		timeout: timeout,
	}
}

// 批量插入选项
func WitchBatchInsertOption(step int) BatchInsertOption {
	return BatchInsertOption{
		step: step,
	}
}

// ========= sync 选项 ================
type SyncOptions struct {
	Step int
}

type SyncOption interface {
	ConfigSyncOptions(opts *SyncOptions)
}

type BatchInsertOption struct {
	step int
}

func (opt BatchInsertOption) ConfigSyncOptions(opts *SyncOptions) {
	opts.Step = opt.step
}

// ========= migrate 选项 ================
type MigrateOptions struct {
	EnabledDropIndex bool
	ConnectTimeout   int64
}

type MigrateOption interface {
	ConfigMigrateOptions(opts *MigrateOptions)
}

type MigrateDropIndex struct {
	enabled bool
}

func (opt MigrateDropIndex) ConfigMigrateOptions(opts *MigrateOptions) {
	opts.EnabledDropIndex = opt.enabled
}

type MigrateConnectTimeout struct {
	timeout int64
}

func (opt MigrateConnectTimeout) ConfigMigrateOptions(opts *MigrateOptions) {
	opts.ConnectTimeout = opt.timeout
}
