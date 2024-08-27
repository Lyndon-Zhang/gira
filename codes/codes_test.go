package codes

import (
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
)

func TestCodes_Trace(t *testing.T) {
	fmt.Println(ErrInvalidArgs.Trace("name", "ljw"))
}

func TestCodes_TraceMongoErrNoDocuments(t *testing.T) {
	fmt.Println(Trace(mongo.ErrNoDocuments, "aa", "bb"))
}

func TestErrors_Is1(t *testing.T) {
	arr := []struct {
		err     error
		checker func(err error) bool
		expect  bool
	}{
		{
			// TraceError -> CodeError
			ErrInvalidPassword.Trace("name", "aa"),
			func(err error) bool {
				return Is(err, ErrInvalidPassword)
			},
			true,
		},
		{
			// TraceError -> TraceError -> CodeError
			Trace(ErrInvalidPassword.Trace("name", "aa")),
			func(err error) bool {
				return Is(err, ErrInvalidPassword)
			},
			true,
		},
		{
			// TraceError -> TraceError -> CodeError
			Trace(ErrInvalidPassword.Trace("name", "aa")),
			func(err error) bool {
				return Is(err, ErrPayOrderStatusInvalid)
			},
			false,
		},
		{
			// 错误码相同，也判断为相同
			Trace(ErrInvalidPassword.Trace("name", "aa")),
			func(err error) bool {
				return Is(err, New(CodeInvalidPassword, ""))
			},
			true,
		},
		{
			// TraceError -> mongo.ErrNoDocuments
			Trace(mongo.ErrNoDocuments),
			func(err error) bool {
				return Is(err, mongo.ErrNoDocuments)
			},
			true,
		},
	}
	for index, v := range arr {
		if v.checker(v.err) != v.expect {
			t.Fatalf("check err(%d) fail, expect:%v got:%v", index, v.expect, v.checker(v.err))
		}
	}
}
