package tracker

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"sync"
	"time"
)

// EntryTimeout - A miner entry expires after EntryTimeout, if no heartbeats are received.
const EntryTimeout = 500 * time.Millisecond

type PortJson struct {
	Port int `json:"port"`
}

type PortsJson struct {
	Ports []int `json:"ports"`
}

// Tracker - A Tracker in the blockchain system.
type Tracker struct {
	miners map[int]*time.Timer // maps each miner's port to its expiration timer
	lock   sync.Mutex          // protects miners for concurrent access
	router *gin.Engine         // http router
	server *http.Server        // http server
}

// NewTracker - creates a new Tracker, but does not start its http server yet.
func NewTracker(port int) *Tracker {
	tracker := &Tracker{
		miners: make(map[int]*time.Timer),
		router: gin.New(),
	}

	// register APIs
	tracker.router.POST("/register", func(ctx *gin.Context) {
		var request PortJson
		if err := ctx.BindJSON(&request); err != nil {
			ctx.JSON(http.StatusBadRequest, nil)
			return
		}
		statusCode, response := tracker.registerHandler(request)
		ctx.JSON(statusCode, response)
	})
	tracker.router.GET("/get_miners", func(ctx *gin.Context) {
		statusCode, response := tracker.getMinersHandler()
		ctx.JSON(statusCode, response)
	})

	tracker.server = &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", port),
		Handler: tracker.router,
	}

	return tracker
}

// Start - starts the Tracker's http server.
func (t *Tracker) Start() {
	go func() {
		if err := t.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("listen: %s\n", err)
		}
	}()
}

// Shutdown - shuts down the Tracker's http server.
func (t *Tracker) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := t.server.Shutdown(ctx); err != nil {
		log.Println("error when shutting down server: ", err)
	}
	select {
	case <-ctx.Done():
		log.Println("shutting down server timeout")
	default:
		break
	}
}

// registerHandler - handles request to /register API.
func (t *Tracker) registerHandler(request PortJson) (int, any) {
	port := request.Port
	t.lock.Lock()
	defer t.lock.Unlock()
	timer, ok := t.miners[port]
	if ok {
		// stop timer
		timer.Stop()
	}
	// register a new timer
	t.miners[port] = time.AfterFunc(EntryTimeout, func() {
		t.lock.Lock()
		defer t.lock.Unlock()
		delete(t.miners, port)
	})
	var response PortsJson
	for port := range t.miners {
		response.Ports = append(response.Ports, port)
	}
	return http.StatusOK, response
}

// getMinersHandler - handles request to /get_miners API.
func (t *Tracker) getMinersHandler() (int, any) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if len(t.miners) == 0 {
		// no miners currently
		return http.StatusNotFound, nil
	}
	ports := make([]int, 0)
	for port := range t.miners {
		ports = append(ports, port)
	}
	response := PortsJson{Ports: ports}
	return http.StatusOK, response
}
