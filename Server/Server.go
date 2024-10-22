package main

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"sync"

	pb "homework3/Proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedChittyChatServer
	participants map[string]chan *pb.BroadcastMessage
	lamportTime  int64
	mu           sync.Mutex
}

func (s *server) Join(ctx context.Context, req *pb.JoinRequest) (*pb.JoinResponse, error) {
	log.Printf("Join request received from participant: %s", req.ParticipantId)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lamportTime++
	s.participants[req.ParticipantId] = make(chan *pb.BroadcastMessage, 100) // Buffered channel to avoid blocking
	message := &pb.BroadcastMessage{
		Message:     "Participant " + req.ParticipantId + " joined Chitty-Chat",
		LamportTime: s.lamportTime,
	}
	log.Printf("Broadcasting join message: %s", message.Message)
	s.broadcast(message)
	log.Printf("Join request processed for participant: %s", req.ParticipantId)
	return &pb.JoinResponse{Message: message.Message, LamportTime: s.lamportTime}, nil
}

func (s *server) Leave(ctx context.Context, req *pb.LeaveRequest) (*pb.LeaveResponse, error) {
	log.Printf("Leave request received from participant: %s", req.ParticipantId)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lamportTime++
	delete(s.participants, req.ParticipantId)
	message := &pb.BroadcastMessage{
		Message:     "Participant " + req.ParticipantId + " left Chitty-Chat",
		LamportTime: s.lamportTime,
	}
	log.Printf("Broadcasting leave message: %s", message.Message)
	s.broadcast(message)
	log.Printf("Leave request processed for participant: %s", req.ParticipantId)
	return &pb.LeaveResponse{Message: message.Message, LamportTime: s.lamportTime}, nil
}

func (s *server) Publish(ctx context.Context, req *pb.PublishRequest) (*pb.PublishResponse, error) {
	log.Printf("Publish request received from participant: %s", req.ParticipantId)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lamportTime++
	message := &pb.BroadcastMessage{
		Message:     req.ParticipantId + ": " + req.Message,
		LamportTime: s.lamportTime,
	}
	log.Printf("Broadcasting message: %s", message.Message)
	s.broadcast(message)
	log.Printf("Publish request processed for participant: %s", req.ParticipantId)
	return &pb.PublishResponse{Message: message.Message, LamportTime: s.lamportTime}, nil
}

func (s *server) Subscribe(req *pb.SubscribeRequest, stream pb.ChittyChat_SubscribeServer) error {
	participantId := req.ParticipantId
	log.Printf("Subscribe request received from participant: %s", participantId)
	if _, ok := s.participants[participantId]; !ok {
		return status.Errorf(codes.NotFound, "Participant %s not found", participantId)
	}
	for message := range s.participants[participantId] {
		log.Printf("Sending message to participant %s: %s", participantId, message.Message)
		if err := stream.Send(message); err != nil {
			log.Printf("Error sending message to participant %s: %v", participantId, err)
			return err
		}
	}
	return nil
}

func (s *server) broadcast(message *pb.BroadcastMessage) {
	log.Printf("Broadcasting message to all participants: %s", message.Message)
	for participantId, ch := range s.participants {
		log.Printf("Sending message to participant %s", participantId)
		ch <- message
	}
}

func main() {
	// Open a file for logging
	logFile, err := os.OpenFile("server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
	defer logFile.Close()

	// Create a multi-writer that writes to both the file and the terminal
	multiWriter := io.MultiWriter(os.Stderr, logFile)
	log.SetOutput(multiWriter)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterChittyChatServer(s, &server{
		participants: make(map[string]chan *pb.BroadcastMessage),
	})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
