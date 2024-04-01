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

package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"go.uber.org/atomic"

	wsactors "github.com/tochemey/goakt-ws/actors"
	"github.com/tochemey/goakt-ws/config"
	"github.com/tochemey/goakt/actors"
	"github.com/tochemey/goakt/log"
)

type Server struct {
	logger log.Logger
	config *config.Config

	actorSystem actors.ActorSystem
	coordinator actors.PID

	started *atomic.Bool
}

// NewServer creates an instance of Server
func NewServer(config *config.Config, logger log.Logger) *Server {
	return &Server{
		config:  config,
		logger:  logger,
		started: atomic.NewBool(false),
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("starting the socket server...")

	if s.started.Load() {
		return nil
	}

	s.actorSystem, _ = actors.NewActorSystem("SocketServer",
		actors.WithLogger(s.logger),
		actors.WithPassivationDisabled(),
		// TODO: add more options when needed
	)

	if err := s.actorSystem.Start(ctx); err != nil {
		s.logger.Error(errors.Wrap(err, "failed to start socket server"))
		return err
	}

	// create the rooms
	for i := 0; i <= s.config.RoomsCount; i++ {
		roomName := fmt.Sprintf("room-%d", i)
		roomActor := wsactors.NewRoom(roomName)
		if _, err := s.actorSystem.Spawn(ctx, roomName, roomActor); err != nil {
			s.logger.Error(errors.Wrap(err, "failed to start socket server"))
			return err
		}
	}

	// start the coordinator
	var err error
	if s.coordinator, err = s.actorSystem.Spawn(ctx, "coordinator", wsactors.NewCoordinator()); err != nil {
		s.logger.Error(errors.Wrap(err, "failed to start socket server"))
		return err
	}

	s.started.Store(true)

	go s.serve()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.started.Store(false)
	return s.actorSystem.Stop(ctx)
}

// serve http request
func (s *Server) serve() {
	go func() {
		http.HandleFunc(s.config.Path, s.serveWs)
		s.logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.config.Port), nil))
	}()
}

func (s *Server) serveWs(writer http.ResponseWriter, request *http.Request) {
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

	header := request.Header
	clientID := header.Get("clientId")
	client := wsactors.NewClient(conn, s.coordinator)

	_, err = s.actorSystem.Spawn(ctx, clientID, client)
	if err != nil {
		s.logger.Error(errors.Wrap(err, "failed to create a websocket client connection"))
		return
	}
}
