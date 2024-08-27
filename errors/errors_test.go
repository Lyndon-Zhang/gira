package errors

import (
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
)

func TestErrors_Trace(t *testing.T) {
	fmt.Println(ErrConfigHandlerNotImplement.Trace("name", "aa"))
}

func TestErrors_Trace2(t *testing.T) {
	err := New("test error", "k1", "v1")
	fmt.Println(err.Trace("k2", "v2"))
}

func TestErrors_Trace3(t *testing.T) {
	err := New("test error", "k1", "v1").Trace("k2", "v2")
	fmt.Println(Trace(err, "k3", "v3"))
}

func TestErrors_Is1(t *testing.T) {
	arr := []struct {
		err     error
		checker func(err error) bool
		expect  bool
	}{
		{
			// TraceError -> Error
			ErrConfigHandlerNotImplement.Trace("name", "aa"),
			func(err error) bool {
				return Is(err, ErrConfigHandlerNotImplement)
			},
			true,
		},
		{
			// TraceError -> TraceError -> Error
			Trace(ErrConfigHandlerNotImplement.Trace("name", "aa")),
			func(err error) bool {
				return Is(err, ErrConfigHandlerNotImplement)
			},
			true,
		},
		{
			Trace(ErrConfigHandlerNotImplement.Trace("name", "aa")),
			func(err error) bool {
				return Is(err, ErrAdminClientNotImplement)
			},
			false,
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
