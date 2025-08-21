package asynclogger

import (
	"context"
	"log"
	"log/slog"
	"os"
	"sync"
	"time"
)

// AsyncHandler is a wrapper over slog.Handler that sends log records to a buffered channel
// and flushes them in a background goroutine.
type AsyncHandler struct {
	inner    slog.Handler
	queue    chan slog.Record
	wg       sync.WaitGroup
	shutdown chan struct{}
	once     sync.Once
}

const (
	defaultBufferSize = 1000
	flushTimeout      = 2 * time.Second
)

var defaultHandler *AsyncHandler

// Init initializes async logging to both stdout and logs.txt.
func Init() {
	logFile, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Fatalf("[AsyncLogger] Failed to open logs.txt: %v", err)
	}

	fileHandler := slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: slog.LevelInfo})
	consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})

	multi := NewMultiHandler(consoleHandler, fileHandler)
	defaultHandler = New(multi)
	slog.SetDefault(slog.New(defaultHandler))
}

// Shutdown flushes logs and shuts down the async logger.
func Shutdown() {
	if defaultHandler != nil {
		defaultHandler.Shutdown()
	}
}

// New creates a new AsyncHandler with the given slog.Handler as the underlying handler.
func New(inner slog.Handler) *AsyncHandler {
	h := &AsyncHandler{
		inner:    inner,
		queue:    make(chan slog.Record, defaultBufferSize),
		shutdown: make(chan struct{}),
	}

	h.wg.Add(1)
	go h.backgroundFlusher()

	return h
}

func (h *AsyncHandler) backgroundFlusher() {
	defer h.wg.Done()
	for {
		select {
		case r := <-h.queue:
			err := h.inner.Handle(context.Background(), r)
			if err != nil {
				return
			}
		case <-h.shutdown:
			// Drain queue before exiting
			for {
				select {
				case r := <-h.queue:
					err := h.inner.Handle(context.Background(), r)
					if err != nil {
						return
					}
				default:
					return
				}
			}
		}
	}
}

// Handle enqueues the log record to be flushed asynchronously.
func (h *AsyncHandler) Handle(_ context.Context, r slog.Record) error {
	select {
	case h.queue <- r:
		return nil
	default:
		// Drop the log if buffer is full
		log.Printf("[AsyncLogger] Dropped log: %s", r.Message)
		return nil
	}
}

func (h *AsyncHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *AsyncHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return New(h.inner.WithAttrs(attrs))
}

func (h *AsyncHandler) WithGroup(name string) slog.Handler {
	return New(h.inner.WithGroup(name))
}

// Shutdown flushes the queue and stops the background goroutine.
func (h *AsyncHandler) Shutdown() {
	h.once.Do(func() {
		close(h.shutdown)
		c := make(chan struct{})
		go func() {
			h.wg.Wait()
			close(c)
		}()
		select {
		case <-c:
		case <-time.After(flushTimeout):
			log.Println("[AsyncLogger] Timed out waiting for shutdown")
		}
	})
}

type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) slog.Handler {
	return &MultiHandler{handlers: handlers}
}

func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, h := range m.handlers {
		if err := h.Handle(ctx, r); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: newHandlers}
}

func (m *MultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return &MultiHandler{handlers: newHandlers}
}

// ---- Usage example in main.go ----

// func main() {
// 	asynclogger.Init()
// 	defer asynclogger.Shutdown()
//
// 	slog.Info("TinyStoreDB server started")
// 	startGRPCServer()
// }
