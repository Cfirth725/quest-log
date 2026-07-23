// Package main coordinates system bootstrapping, network socket bindings,
// and automated background scheduling loops for the application lifecycle.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"quest-log/internal/database"
	"quest-log/internal/web"

	"github.com/robfig/cron/v3"
)

func main() {
	// Create a root context for the application bootstrap lifetime
	ctx, cancelBootstrap := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelBootstrap()

	// ====================================================================
	// -- PHASE 1: DATA PERSISTENCE INITIALIZATION --
	// ====================================================================

	log.Println("[INIT] Initializing system runtime: Bootstrapping SQLite persistence layer")
	db, err := database.Connect(ctx)
	if err != nil {
		log.Fatalf("[ERROR] CRITICAL ENGINE BREAKDOWN: Connection suite failed: %v", err)
	}

	database.DB = db
	log.Println("[OK] Data persistence engine successfully verified and bound to active runtime")

	// STORAGE HYGIENE SWEEP: Fire sector compaction and retention pruning asynchronously on boot.
	go database.OptimizeDatabase(context.Background(), db)

	// ====================================================================
	// -- PHASE 2: BACKGROUND AUTOMATION POOL (CRON ORCHESTRATION) --
	// ====================================================================

	log.Println("[INIT] Mounting automated Cron schedule worker daemon")
	c := cron.New()

	// Schedule the Master Spawner to run daily at 04:03 AM.
	_, err = c.AddFunc("3 4 * * *", func() {
		database.RunMasterSpawner(db)
	})

	if err != nil {
		log.Fatalf("[ERROR] Initialization barrier: Failed to configure background scheduling loops: %v", err)
	}

	c.Start()
	log.Println("[IDLE] Cron automation engine running safely in background thread pool")

	// COLD START EXECUTION: Immediately run the spawner on launch to ensure data consistency.
	log.Println("[REALTIME] Executing cold-start idempotent quest activation check")
	database.RunMasterSpawner(db)

	// ====================================================================
	// -- PHASE 3: HTTP ROUTE CONFIGURATION MATRIX --
	// ====================================================================

	log.Println("[INIT] Spawning HTTP router multiplexer and matching paths")
	mux := http.NewServeMux()

	// Register all HTTP handlers and API endpoints from internal/web
	web.RegisterRoutes(mux, db)

	log.Println("[OK] Inbound web traffic pathways successfully mapped to router multiplexer")

	// ====================================================================
	// -- PHASE 4: NETWORK SOCKET BINDING & EXECUTION --
	// ====================================================================

	port := os.Getenv("APP_PORT")
	if port == "" {
		log.Fatalf("[ERROR] CRITICAL LAUNCH BREAKDOWN: The infrastructure 'APP_PORT' environment variable is unassigned. The container engine must explicitly pass this parameter to map local network sockets.")
	}
	addr := ":" + port

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("[SERVER] Launching Quest Log network hub listening at port %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[ERROR] CRITICAL SERVER CRASH: Socket exception binding port: %v", err)
		}
	}()

	// ====================================================================
	// -- PHASE 5: GRACEFUL SHUTDOWN ORCHESTRATION LAYER --
	// ====================================================================

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Block main execution thread until an operating system signal is intercepted
	<-stop
	log.Println("[SHUTDOWN] OS signal intercepted: Commencing reverse-order resource drainage cascade")

	// 1. Drain background job workers
	c.Stop()
	log.Println("[SHUTDOWN] Background automated cron threads systematically unmounted")

	// 2. Terminate server socket bindings gracefully (5-second drainage limit)
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[ERROR] In-flight network drainage failed, forcing thread break: %v", err)
	} else {
		log.Println("[SHUTDOWN] Active network connection pipeline cleared and closed cleanly")
	}

	// 3. Finalize transactional state and close SQLite connection
	if db != nil {
		log.Println("[SHUTDOWN] Flushing journal blocks to clean disk sectors...")
		if err := db.Close(); err != nil {
			log.Printf("[ERROR] SQLite connection termination encountered faults: %v", err)
		} else {
			log.Println("[SHUTDOWN] SQLite database WAL logs merged safely. Connection severed cleanly.")
		}
	}

	log.Println("[SHUTDOWN] System offline. All hardware nodes and resources fully released.")
}
