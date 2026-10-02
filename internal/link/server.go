package link

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/EraseMC/Aegis/internal/config"
	"github.com/EraseMC/Aegis/internal/engine"
	"github.com/EraseMC/Aegis/internal/wire"
)

const outboundQueue = 4096

type Server struct {
	cfg config.Config
	log *slog.Logger
}

func NewServer(cfg config.Config, log *slog.Logger) *Server {
	return &Server{cfg: cfg, log: log}
}

func (s *Server) Serve(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.cfg.Listen)
	if err != nil {
		return err
	}
	s.log.Info("listening", "address", listener.Addr().String())
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	var wg sync.WaitGroup
	defer wg.Wait()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.handle(ctx, conn)
		}()
	}
}

func (s *Server) handle(ctx context.Context, conn net.Conn) {
	log := s.log.With("remote", conn.RemoteAddr().String())
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetNoDelay(true)
	}
	out := make(chan []byte, outboundQueue)
	done := make(chan struct{})
	go write(conn, out, done, log)

	eng := engine.New(s.cfg, log, func(frame []byte) {
		select {
		case out <- frame:
		default:
			log.Warn("outbound queue is full, dropping a verdict")
		}
	})

	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	defer func() {
		_ = conn.Close()
		close(out)
		<-done
		log.Info("server disconnected", "players", eng.Players())
	}()

	reader := bufio.NewReaderSize(conn, 256<<10)
	var scratch []byte
	for {
		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		t, body, err := wire.ReadFrame(reader, scratch)
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				log.Warn("read failed", "err", err)
			}
			return
		}
		scratch = body[:0]
		if err := eng.Handle(t, body); err != nil {
			log.Warn("bad frame", "type", t, "err", err)
			if t == wire.TypeHello {
				return
			}
		}
	}
}

func write(conn net.Conn, out <-chan []byte, done chan<- struct{}, log *slog.Logger) {
	defer close(done)
	defer conn.Close()
	writer := bufio.NewWriter(conn)
	for frame := range out {
		_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if _, err := writer.Write(frame); err != nil {
			log.Warn("write failed", "err", err)
			return
		}
		if len(out) == 0 {
			if err := writer.Flush(); err != nil {
				log.Warn("flush failed", "err", err)
				return
			}
		}
	}
	_ = writer.Flush()
}
