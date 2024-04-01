package actors

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"github.com/tochemey/goakt-ws/config"
	"github.com/tochemey/goakt-ws/goaktwspb"
	"github.com/tochemey/goakt/actors"
	"github.com/tochemey/goakt/goaktpb"
	"github.com/tochemey/goakt/log"
)

const (
	wsPath = "/ws"
)

type Coordinator struct {
	// specifies the websocket server port
	port int32
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
	config, err := config.GetConfig()
	if err != nil {
		s.logger.Errorf("failed to load the Server configuration: (%s)", err.Error())
		return err
	}

	s.port = config.Port
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

// serve http request
func (s *Coordinator) serve() {
	go func() {
		http.HandleFunc(wsPath, s.handleWebsocket)
		s.logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil))
	}()
}

func (s *Coordinator) handleWebsocket(writer http.ResponseWriter, request *http.Request) {
	s.logger.Debug("new client connection")

	ctx := request.Context()

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		s.logger.Error(errors.Wrap(err, "failed to get the websocket client connection"))
		return
	}

	sessionID := request.URL.Query().Get("sessionId")
	session := NewClient(conn, sessionID, s.pid)
	cid, err := s.pid.SpawnChild(request.Context(), sessionID, session)
	if err != nil {
		s.logger.Error(errors.Wrap(err, "failed to create a websocket client connection"))
		return
	}

	s.sessions[sessionID] = cid

	header := request.Header
	clientID := header.Get("clientId")
	roomID, ok := s.rooms[clientID]
	if !ok {
		room := NewRoom(clientID)
		roomID, err = s.pid.SpawnChild(request.Context(), clientID, room)
		if err != nil {
			s.logger.Error(errors.Wrap(err, "failed to create a websocket room connection"))
			return
		}

		s.rooms[clientID] = roomID
	}

	if err := cid.Tell(ctx, roomID, &goaktwspb.JoinRoom{RoomId: clientID}); err != nil {
		s.logger.Errorf("session=(%s) failed to join room (%s): (%v)", sessionID, clientID, err)
		return
	}
}
