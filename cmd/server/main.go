package main

import (
	"UserService/adapters/userRepository"
	"UserService/protobufs/UserServiceSchema"
	"UserService/services/UserService"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
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

func SetupGormDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&userRepository.UserDTODB{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate domain : %v", err)
	}
	return db, nil
}

func SetupGRPCServer(userRepo userRepository.UserRepo) *grpc.Server {
	grpcServer := grpc.NewServer()
	UserServiceSchema.RegisterUserServiceServer(grpcServer, UserService.NewUserService(userRepo))
	return grpcServer
}

func main() {
	//setup database
	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		log.Fatalf("Specify %v envvar!", EnvDSN)
	}
	var userRepo userRepository.UserRepo
	if dsn == "dynamo" {
		sess := session.Must(session.NewSession())
		var err error
		userRepo, err = userRepository.NewAwsDynamoUserRepo(sess)
		if err != nil {
			log.Fatalf("Failed to setup db : %v", err)
		}
	} else if dsn == "dynamo-local" {
		log.Printf("Setting up dynamo-local db")
		sess := session.Must(session.NewSession())
		var err error
		userRepo, err = userRepository.NewAwsLocalDynamoUserRepo(sess)
		if err != nil {
			log.Fatalf("Failed to setup db : %v", err)
		}
	} else {
		db, err := SetupGormDB(dsn)
		if err != nil {
			log.Fatalf("failed to setup db : %v", err)
		}
		userRepo = &userRepository.DefaultRepo{DB: db}
	}

	//start grpc server
	lis, err := net.Listen("tcp", os.Getenv(EnvListenAddr))
	if err != nil {
		log.Fatalf("failed to listen on %v : %v", os.Getenv(EnvListenAddr), err)
	}

	grpcServer := SetupGRPCServer(userRepo)
	log.Printf("Starting GRPC server")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("grpcServer termianted with :%v", err)
	}

}
