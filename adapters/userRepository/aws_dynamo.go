package userRepository

import (
	"UserService/domain"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"log"
	"time"
)

const dynamoTimeout = 10 * time.Second

//awsErrorsIs returns true is err is and awsError with code awsErrCode. Safe to call on nil err value
func awsErrorIs(err error, awsErrCode string) bool {
	if err == nil {
		return false
	}
	awsErr, ok := err.(awserr.Error)
	if !ok {
		return false
	}
	log.Printf("code %v", awsErr.Code())
	return awsErr.Code() == awsErrCode
}

const TableUser = "Users"
const TableUserPkName = "PublicKeyPKIX"

const TableEmailToPublicKey = "EmailToUserPk"
const TableEmailToPublicKeyPkName = "Email"

var createRequests = []*dynamodb.CreateTableInput{
	{
		TableName: aws.String(TableUser),
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String(TableUserPkName),
				AttributeType: aws.String("B"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String(TableUserPkName),
				KeyType:       aws.String("HASH"),
			},
		},
		BillingMode: aws.String("PAY_PER_REQUEST"),
	},
	{
		TableName: aws.String(TableEmailToPublicKey),
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String(TableEmailToPublicKeyPkName),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String(TableEmailToPublicKeyPkName),
				KeyType:       aws.String("HASH"),
			},
		},
		BillingMode: aws.String("PAY_PER_REQUEST"),
	},
}

type EmailToPkEntry struct {
	Email      string
	PrimaryKey []byte
}

func (a AwsDynamoUserRepo) doesUserExist(ctx context.Context, email string) error {
	//check if user already exists
	_, err := a.db.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(TableEmailToPublicKey),
		Key: map[string]*dynamodb.AttributeValue{
			TableEmailToPublicKeyPkName: {
				S: aws.String(email),
			},
		},
	})
	if err != nil {
		if awsErrorIs(err, dynamodb.ErrCodeDuplicateItemException) {
			return fmt.Errorf("failed to insert user : %w", ErrAlreadyExists)
		}
		return err
	}
	return nil
}

func createTable(db *dynamodb.DynamoDB, createIn *dynamodb.CreateTableInput) error {
	descTableIn := &dynamodb.DescribeTableInput{
		TableName: aws.String(*createIn.TableName),
	}
	descReq, _ := db.DescribeTableRequest(descTableIn)
	err := descReq.Send()
	//if table exists, abort here
	if err == nil {
		return nil
	}

	createReq, createResp := db.CreateTableRequest(createIn)
	err = createReq.Send()
	if err != nil {
		return fmt.Errorf("CreateTable failed: err=%v resp=%v", err, createResp)
	}
	return nil
}

type AwsDynamoUserRepo struct {
	db *dynamodb.DynamoDB
}

func (a AwsDynamoUserRepo) GetByPk(ctx context.Context, PKIXPublicKey []byte) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, dynamoTimeout)
	defer cancel()

	result, err := a.db.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(TableUser),
		Key: map[string]*dynamodb.AttributeValue{
			TableUserPkName: {
				B: PKIXPublicKey,
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch user : %v", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("failed to fetch user : %w", ErrNotFound)
	}

	dbUser := &UserDTODB{}
	if err := dynamodbattribute.UnmarshalMap(result.Item, dbUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dynamodb entry to UserDTODB : %v", err)
	}

	user, err := dbUser.toUser()
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal UserDTODB entry to user : %v", err)
	}
	return user, nil

}

func (a AwsDynamoUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*dynamoTimeout)
	defer cancel()

	//user TableEmailToPublicKey to get Public Key for email address, then
	result, err := a.db.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(TableEmailToPublicKey),
		Key: map[string]*dynamodb.AttributeValue{
			TableEmailToPublicKeyPkName: {
				S: aws.String(email),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user by email : %v", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("failed to fetch user : %w", ErrNotFound)
	}

	emailToPk := &EmailToPkEntry{}
	if err := dynamodbattribute.UnmarshalMap(result.Item, emailToPk); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dynamodb entry to EmailToPkEntry : %v", err)
	}

	//User Public Key to get User
	return a.GetByPk(ctx, emailToPk.PrimaryKey)

}

func (a AwsDynamoUserRepo) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, dynamoTimeout)
	defer cancel()

	if err := a.doesUserExist(ctx, u.Email); err != nil {
		return nil, err
	}

	//update time stamps
	u.CreatedAt = time.Now()
	u.UpdatedAt = u.CreatedAt

	//convert for insert
	dbUser, err := userToDTODB(u)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize user for DB : %v", err)
	}
	userAwsMap, err := dynamodbattribute.MarshalMap(dbUser)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize user for dynamodb : %v", err)
	}
	userToPkAwsMap, err := dynamodbattribute.MarshalMap(&EmailToPkEntry{
		Email:      dbUser.Email,
		PrimaryKey: dbUser.PublicKeyPKIX,
	})

	//do atomic insert
	_, err = a.db.TransactWriteItemsWithContext(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []*dynamodb.TransactWriteItem{
			{
				Put: &dynamodb.Put{
					Item:      userAwsMap,
					TableName: aws.String(TableUser),
				},
			},
			{
				Put: &dynamodb.Put{
					Item:      userToPkAwsMap,
					TableName: aws.String(TableEmailToPublicKey),
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert user : %v", err)
	}

	return u, nil

}

func (a AwsDynamoUserRepo) DeleteByEmail(ctx context.Context, email string) error {

	user, err := a.GetByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to delete user : %v", err)
	}

	userDB, err := userToDTODB(user)
	if err != nil {
		return fmt.Errorf("failed to convert user to db representation : %v", err)
	}

	//do atomic delete
	_, err = a.db.TransactWriteItemsWithContext(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []*dynamodb.TransactWriteItem{
			{
				Delete: &dynamodb.Delete{
					Key: map[string]*dynamodb.AttributeValue{
						TableEmailToPublicKeyPkName: {
							S: aws.String(email),
						},
					},
					TableName: aws.String(TableEmailToPublicKey),
				},
			},
			{
				Delete: &dynamodb.Delete{
					Key: map[string]*dynamodb.AttributeValue{
						TableUserPkName: {
							B: userDB.PublicKeyPKIX,
						},
					},
					TableName: aws.String(TableUser),
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete user : %v", err)
	}

	return nil
}

func NewAwsDynamoUserRepo(sess *session.Session) (*AwsDynamoUserRepo, error) {
	db := dynamodb.New(sess)
	return newAwsDynamoUserRepo(db)
}

func NewAwsLocalDynamoUserRepo(sess *session.Session) (*AwsDynamoUserRepo, error) {
	db := dynamodb.New(sess, aws.NewConfig().WithEndpoint("http://localhost:8000"))
	return newAwsDynamoUserRepo(db)
}

func newAwsDynamoUserRepo(db *dynamodb.DynamoDB) (*AwsDynamoUserRepo, error) {
	for _, v := range createRequests {
		if err := createTable(db, v); err != nil {
			return nil, fmt.Errorf("error setting up tables : %v", err)
		}
	}
	repo := &AwsDynamoUserRepo{db: db}
	return repo, nil
}
