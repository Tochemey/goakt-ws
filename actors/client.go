/*
 * MIT License
 *
 * Copyright (c) 2022-2024 Tochemey
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package actors

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/tochemey/goakt-ws/goaktwspb"
	"github.com/tochemey/goakt/actors"
	"github.com/tochemey/goakt/goaktpb"
	"github.com/tochemey/goakt/log"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = pongWait * 9 / 10
	maxMessageSize = 1024 * 1024 * 1024
)

type Client struct {
	conn   *websocket.Conn
	id     string
	server actors.PID
	self   actors.PID

	logger log.Logger

	rooms map[string]actors.PID
}

// compilation error
var _ actors.Actor = (*Client)(nil)

func NewClient(conn *websocket.Conn, clientID string, server actors.PID) *Client {
	return &Client{
		conn:   conn,
		id:     clientID,
		server: server,
		logger: log.DefaultLogger,
		rooms:  make(map[string]actors.PID),
	}
}

// PreStart handles pre-start process
func (s *Client) PreStart(ctx context.Context) error {
	return nil
}

// Receive handle messages sent to the server
func (s *Client) Receive(ctx actors.ReceiveContext) {
	switch m := ctx.Message().(type) {
	case *goaktpb.PostStart:
		s.self = ctx.Self()
		go s.readPump()

	case *goaktwspb.JoinRoom:
		roomID := m.GetRoomId()
		_, ok := s.rooms[roomID]

		if !ok {
			ctx.Tell(s.server, &goaktwspb.JoinRoom{RoomId: roomID})
			return
		}

		// ignore the message because already joined the room
		ctx.Unhandled()

	case *goaktwspb.LeaveRoom:
		roomID := m.GetRoomId()
		_, ok := s.rooms[roomID]

		if !ok {
			ctx.Tell(s.server, &goaktwspb.LeaveRoom{RoomId: roomID})
			return
		}

		// ignore the message because already joined the room
		ctx.Unhandled()

	case *goaktwspb.RoomJoined:
		roomID := ctx.Sender()
		roomPath := roomID.ActorPath().String()
		s.rooms[roomPath] = roomID

	case *goaktwspb.RoomLeft:
		roomID := ctx.Sender()
		roomPath := roomID.ActorPath().String()
		if _, ok := s.rooms[roomPath]; !ok {
			delete(s.rooms, roomPath)
		}

	case *goaktwspb.Message:
	default:
		ctx.Unhandled()
	}
}

// PostStop handles post shutdown process
func (s *Client) PostStop(ctx context.Context) error {
	for _, room := range s.rooms {
		c := context.WithoutCancel(ctx)
		if err := s.self.Tell(c, room, new(goaktwspb.LeaveRoom)); err != nil {
			return err
		}
	}

	return s.conn.Close()
}

func (s *Client) readPump() {
	defer func() {
		_ = s.self.Shutdown(context.Background())
	}()

	s.conn.SetReadLimit(maxMessageSize)
	s.conn.SetReadDeadline(time.Now().Add(pongWait))
	s.conn.SetPongHandler(func(string) error { s.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, bytea, err := s.conn.ReadMessage()
		if err != nil {
			s.logger.Error(errors.Wrap(err, "failed to read message"))
			return
		}

		var message proto.Message
		if err := protojson.Unmarshal(bytea, message); err != nil {
			s.logger.Error(errors.Wrap(err, "failed to read message"))
			return
		}

		switch m := message.(type) {
		case *goaktwspb.JoinRoom:
			if err := actors.Tell(context.Background(), s.self, m); err != nil {
				s.logger.Error(err)
				return
			}

		case *goaktwspb.LeaveRoom:
			if err := actors.Tell(context.Background(), s.self, m); err != nil {
				s.logger.Error(err)
				return
			}
		case *goaktwspb.Message:
			// TODO: handle message
		}
	}
}
