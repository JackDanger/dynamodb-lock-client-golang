package lockclient

import (
	"time"

	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/sirupsen/logrus"
)

// DynamoDBLockClient describes the fields for a lock client
type DynamoDBLockClient struct {
	LockName        string
	LeaseDuration   time.Duration
	HeartbeatPeriod time.Duration
	TableName       string
	Identifier      string
	Client          dynamodbiface.DynamoDBAPI
	Logger          *logrus.Logger
	lockID          string
	sendHeartbeats  bool
	lockError       error
}
