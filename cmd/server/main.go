package main

import (
	"UserService/adapters/userRepository"
	"UserService/services/github.com/alyrot/UserServiceSchema"
	"fmt"
	"google.golang.org/grpc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"net"
	"os"
)

const (
	//EnvDSN connection string for database
	EnvDSN        string = "DSN"
	EnvListenAddr string = "LISTEN"
)

func SetupDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&userRepository.UserDTODB{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate domain : %v", err)
	}
	return db, nil
}

func SetupGRPCServer(db *gorm.DB) *grpc.Server {
	userRepo := &userRepository.DefaultRepo{DB: db}
	grpcServer := grpc.NewServer()
	UserServiceSchema.RegisterUserServiceServer(grpcServer, UserServiceSchema.NewUserService(userRepo))
	return grpcServer
}

func main() {
	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		log.Fatalf("Specify %v envvar!", EnvDSN)
	}
	db, err := SetupDB(dsn)
	if err != nil {
		log.Fatalf("failed to setup db : %v", err)
	}

	lis, err := net.Listen("tcp", os.Getenv(EnvListenAddr))
	if err != nil {
		log.Fatalf("failed to listen on %v : %v", os.Getenv(EnvListenAddr), err)
	}
	grpcServer := SetupGRPCServer(db)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("grpcServer termianted with :%v", err)
	}

}
