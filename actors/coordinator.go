package actors

import (
	"context"

	"github.com/tochemey/goakt-ws/goaktwspb"
	"github.com/tochemey/goakt/actors"
	"github.com/tochemey/goakt/goaktpb"
	"github.com/tochemey/goakt/log"
)

type Coordinator struct {
	// pid represents the server PID
	pid actors.PID

	// specifies the rooms
	rooms    map[string]actors.PID
	sessions map[string]actors.PID

	logger log.Logger
}

var _ actors.Actor = (*Coordinator)(nil)

// NewCoordinator creates a new instance of Coordinator
func NewCoordinator() *Coordinator {
	return &Coordinator{
		rooms:    make(map[string]actors.PID),
		sessions: make(map[string]actors.PID),
		logger:   log.DefaultLogger,
	}
}

// PostStop implements actors.Actor.
func (s *Coordinator) PostStop(ctx context.Context) error {
	return nil
}

// PreStart implements actors.Actor.
func (s *Coordinator) PreStart(ctx context.Context) error {
	return nil
}

// Receive implements actors.Actor.
func (s *Coordinator) Receive(ctx actors.ReceiveContext) {
	switch msg := ctx.Message().(type) {
	case *goaktpb.PostStart:
		s.pid = ctx.Self()
	case *goaktwspb.JoinRoom:
		if roomPid, ok := s.rooms[msg.GetRoomId()]; ok {
			ctx.Forward(roomPid)
			return
		}

		// TODO: send some reply to the sender
	case *goaktwspb.LeaveRoom:
		if roomPid, ok := s.rooms[msg.GetRoomId()]; ok {
			ctx.Forward(roomPid)
			return
		}

		// TODO: send some reply to the sender
	default:
		ctx.Unhandled()
	}
}
