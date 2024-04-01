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

	"github.com/tochemey/goakt-ws/goaktwspb"
	"github.com/tochemey/goakt/actors"
)

// Room represents the room
type Room struct {
	name    string
	clients map[string]actors.PID
}

var _ actors.Actor = (*Room)(nil)

// NewRoom creates a new Room
func NewRoom(name string) *Room {
	return &Room{
		name:    name,
		clients: make(map[string]actors.PID),
	}
}

// PostStop implements actors.Actor.
func (r *Room) PostStop(ctx context.Context) error {
	return nil
}

// PreStart implements actors.Actor.
func (r *Room) PreStart(ctx context.Context) error {
	return nil
}

// Receive implements actors.Actor.
func (r *Room) Receive(ctx actors.ReceiveContext) {
	switch msg := ctx.Message().(type) {
	case *goaktwspb.JoinRoom:
		roomID := msg.GetRoomId()
		if roomID != r.name {
			ctx.Unhandled()
		}

		sender := ctx.Sender()
		clientID := sender.ActorPath().String()
		r.clients[clientID] = sender
		ctx.Tell(sender, &goaktwspb.RoomJoined{RoomId: roomID})

	case *goaktwspb.LeaveRoom:
		sender := ctx.Sender()
		clientID := sender.ActorPath().String()
		if _, ok := r.clients[clientID]; ok {
			delete(r.clients, clientID)
			ctx.Tell(sender, &goaktwspb.RoomLeft{RoomId: msg.RoomId})
		}

	case *goaktwspb.Message:

	default:
		ctx.Unhandled()
	}
}
