package mygo

import (
	"encoding/json"
	"net"
	"os"

	"github.com/zncoder/check"
)

type Handler[Arg, Result any] interface {
	Handle(*Arg) *Result
}

type UnixRPCServer[Arg, Result any] struct {
	lr      net.Listener
	handler Handler[Arg, Result]
}

func NewUnixRPCServer[Arg, Result any](sockAddr string, handler Handler[Arg, Result]) *UnixRPCServer[Arg, Result] {
	os.Remove(sockAddr)
	lr := check.V(net.Listen("unix", sockAddr)).F("listen", "sockAddr", sockAddr)
	return &UnixRPCServer[Arg, Result]{lr: lr, handler: handler}
}

func (rs *UnixRPCServer[Arg, Result]) Loop() {
	defer rs.lr.Close()

	for {
		if conn, ok := check.V(rs.lr.Accept()).K("accept"); ok {
			rs.handleConn(conn)
		}
	}
}

func (rs *UnixRPCServer[Arg, Result]) Close() error {
	return rs.lr.Close()
}

func (rs *UnixRPCServer[Arg, Result]) handleConn(conn net.Conn) {
	defer conn.Close()

	var arg Arg
	if !check.E(json.NewDecoder(conn).Decode(&arg)).L("read arg", "arg_t", Type(arg)) {
		return
	}

	result := rs.handler.Handle(&arg)

	check.E(json.NewEncoder(conn).Encode(&result)).L("write result", "result", result)
}

type UnixRPC[Arg, Result any] struct {
	conn net.Conn
}

func NewUnixRPC[Arg, Result any](sockAddr string) UnixRPC[Arg, Result] {
	conn := check.V(net.Dial("unix", sockAddr)).F("dial unix", "sockAddr", sockAddr)
	return UnixRPC[Arg, Result]{conn: conn}
}

func (ur UnixRPC[Arg, Result]) Call(arg *Arg) *Result {
	check.E(json.NewEncoder(ur.conn).Encode(arg)).F("write arg", "arg_t", Type(arg), "arg", arg)
	var result Result
	check.E(json.NewDecoder(ur.conn).Decode(&result)).F("read result", "result_t", Type(result))
	return &result
}

func (ur *UnixRPC[Arg, Result]) Close() {
	check.E(ur.conn.Close()).F("close conn")
}
