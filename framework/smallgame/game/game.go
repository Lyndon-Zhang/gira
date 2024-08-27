package game

import (
	"context"

	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/actor"
	"github.com/Lyndon-Zhang/gira/framework/smallgame/game/config"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 玩家接口
type Player interface {
	// 加载数据
	Load(ctx context.Context, memberId string, userId string) error
	// 断开连接,登出
	Logout(ctx context.Context) error
	// 保存数据
	BgSave(ctx context.Context) error
	Update()
}

type GameFramework interface {
	gira.Framework
	GetConfig() *config.GameConfig
	SessionCount() int64
	// 将消息推送给玩家, 不保证消息已经被处理，如果玩家当前不在线，消息将会推送失败，但不会返回错误
	Push(ctx context.Context, userId string, req gira.ProtoPush) error
	// 会将消息推送到玩家的消息队列中，但不等待结果，如果玩家不在线，会返回错误
	MustPush(ctx context.Context, userId string, req gira.ProtoPush) (err error)
}

type Session interface {
	// 推送消息给当前session
	Push(resp gira.ProtoPush) (err error)
	Kick(ctx context.Context, reason string) (err error)
	Inbox() chan actor.Request
	// 让出session的控制权后执行f函数， 返回后将重新获得控制权
	Yield(f func() error) error
	// 抢占session的控制台执行f函数，返回后释放控制台
	Go(f func() error) error
}

type HallHandler interface {
	// 由memberId创建账号
	NewUser(ctx context.Context, memberId string) (avatar UserAvatar, err error)
	// 创建player
	NewPlayer(ctx context.Context, session Session, memberId string, avatar UserAvatar) (player Player, err error)
}

type UserAvatar interface {
	GetUserId() primitive.ObjectID
}
