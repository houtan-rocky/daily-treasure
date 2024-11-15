package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	pb "github.com/houtan-rocky/daily-treasure/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

const (
	TARGET_GEO_ZONE = 100.0 // diameter of target zone
	WIN_GEO_ZONE    = 1.0   // diameter of win zone
	MAP_SIZE        = 1000.0 // size of the map
	defaultPort     = "50051"

	// Fixed positions for target and win zones
	TARGET_X = 500.0  // center of target zone X coordinate
	TARGET_Y = 500.0  // center of target zone Y coordinate
	WIN_X    = 510.0  // center of win zone X coordinate
	WIN_Y    = 510.0  // center of win zone Y coordinate

	// Game mode: true for fixed positions, false for random positions
	USE_FIXED_POSITIONS = true
)

type Position struct {
	X, Y float64
}

type GameServer struct {
	pb.UnimplementedDailyTreasureServer
	mu         sync.RWMutex
	targetZone Position
	winZone    Position
	stats      struct {
		totalUpdates  uint64
		playersInGame uint64
	}
}

// NewGameServer creates and initializes a new game server
func NewGameServer() *GameServer {
	s := &GameServer{}
	s.generateZones()
	return s
}

// generateZones creates new target and win zones
func (s *GameServer) generateZones() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if USE_FIXED_POSITIONS {
		// Use fixed positions
		s.targetZone = Position{X: TARGET_X, Y: TARGET_Y}
		s.winZone = Position{X: WIN_X, Y: WIN_Y}

		// Validate fixed positions
		if !isPositionValid(s.targetZone) {
			log.Printf("Warning: Fixed target position (%.2f, %.2f) is outside map bounds, using random position", TARGET_X, TARGET_Y)
			s.targetZone = generateRandomPosition()
		}

		if !isInZone(s.winZone, s.targetZone, TARGET_GEO_ZONE/2-WIN_GEO_ZONE/2) {
			log.Printf("Warning: Fixed win zone position (%.2f, %.2f) is outside target zone, generating new position", WIN_X, WIN_Y)
			s.winZone = s.generateWinZone()
		}

		log.Printf("Using fixed positions - Target: (%.2f, %.2f), Win: (%.2f, %.2f)", 
			s.targetZone.X, s.targetZone.Y, s.winZone.X, s.winZone.Y)
	} else {
		// Use random positions
		s.targetZone = generateRandomPosition()
		s.winZone = s.generateWinZone()
		log.Printf("Using random positions - Target: (%.2f, %.2f), Win: (%.2f, %.2f)", 
			s.targetZone.X, s.targetZone.Y, s.winZone.X, s.winZone.Y)
	}
}

func (s *GameServer) generateWinZone() Position {
	maxAttempts := 100
	for i := 0; i < maxAttempts; i++ {
		winZone := Position{
			X: s.targetZone.X + (rand.Float64()*TARGET_GEO_ZONE - TARGET_GEO_ZONE/2),
			Y: s.targetZone.Y + (rand.Float64()*TARGET_GEO_ZONE - TARGET_GEO_ZONE/2),
		}
		if isInZone(winZone, s.targetZone, TARGET_GEO_ZONE/2-WIN_GEO_ZONE/2) {
			return winZone
		}
	}
	// Fallback to a safe position if random generation fails
	return Position{
		X: s.targetZone.X,
		Y: s.targetZone.Y,
	}
}

func (s *GameServer) GetGameConfig(ctx context.Context, empty *pb.Empty) (*pb.GameConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &pb.GameConfig{
		TargetZoneDiameter: TARGET_GEO_ZONE,
		WinZoneDiameter:    WIN_GEO_ZONE,
		TargetZoneX:        s.targetZone.X,
		TargetZoneY:        s.targetZone.Y,
		WinZoneX:          s.winZone.X,
		WinZoneY:          s.winZone.Y,
	}, nil
}

func (s *GameServer) UpdatePosition(ctx context.Context, pos *pb.Position) (*pb.GameState, error) {
	if err := validatePosition(pos); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	roverPos := Position{X: pos.X, Y: pos.Y}
	
	// Update stats
	s.stats.totalUpdates++

	// Check if rover is in win zone
	if isInZone(roverPos, s.winZone, WIN_GEO_ZONE/2) {
		return &pb.GameState{
			Status: pb.GameState_IN_WIN_ZONE,
			Won:    true,
		}, nil
	}

	// Check if rover is in target zone
	if isInZone(roverPos, s.targetZone, TARGET_GEO_ZONE/2) {
		return &pb.GameState{
			Status: pb.GameState_IN_TARGET_ZONE,
			Won:    false,
		}, nil
	}

	// Rover is outside both zones
	distance := calculateDistance(roverPos, s.targetZone)
	return &pb.GameState{
		Status:         pb.GameState_OUTSIDE,
		TargetDistance: &distance,
		Won:           false,
	}, nil
}

func validatePosition(pos *pb.Position) error {
	if pos == nil {
		return fmt.Errorf("position cannot be nil")
	}
	if !isPositionValid(Position{X: pos.X, Y: pos.Y}) {
		return fmt.Errorf("position (%.2f, %.2f) is outside the map bounds [0, %.2f]", pos.X, pos.Y, MAP_SIZE)
	}
	return nil
}

func isPositionValid(pos Position) bool {
	return pos.X >= 0 && pos.X <= MAP_SIZE && pos.Y >= 0 && pos.Y <= MAP_SIZE
}

func isInZone(pos, center Position, radius float64) bool {
	return calculateDistance(pos, center) <= radius
}

func calculateDistance(pos1, pos2 Position) float64 {
	dx := pos1.X - pos2.X
	dy := pos1.Y - pos2.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func generateRandomPosition() Position {
	return Position{
		X: rand.Float64() * MAP_SIZE,
		Y: rand.Float64() * MAP_SIZE,
	}
}

func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	return ":" + defaultPort
}

func setupGrpcServer() *grpc.Server {
	return grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Time:             20 * time.Second,
			Timeout:          1 * time.Second,
		}),
	)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	port := getPort()
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	

	gameServer := NewGameServer()
	grpcServer := setupGrpcServer()
	pb.RegisterDailyTreasureServer(grpcServer, gameServer)
	reflection.Register(grpcServer)



	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Server starting on port %s", port)
	log.Printf("Target Zone: (%.2f, %.2f)", gameServer.targetZone.X, gameServer.targetZone.Y)
	log.Printf("Win Zone: (%.2f, %.2f)", gameServer.winZone.X, gameServer.winZone.Y)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	<-stop

	log.Println("Shutting down server...")
	grpcServer.GracefulStop()
	log.Printf("Server stopped. Total position updates processed: %d", gameServer.stats.totalUpdates)
}