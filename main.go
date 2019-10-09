package lockclient

import (
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// GetLock requests a lock, and returns a lock on failure
func (d *DynamoDBLockClient) GetLock() (bool, error) {

	if d.Identifier == "" {
		uuid, err := uuid.NewRandom()
		if err != nil {
			d.logger().Debugf("An error occured: %s", err)
			return false, err
		}
		d.Identifier = uuid.String()
	}
	d.logger().Debugf("Attempting to get lock '%s' for %s (in %#v)", d.LockName, d.LeaseDuration, d)

	err := d.dynamoGetLock()
	if err != nil {
		d.logger().Debugf("An error occured: %s - %#v", err, d)
		existing, _ := d.dynamoExamineLock()
		if existing != nil {
			d.logger().Debugf("Found existing lock: %#v", existing)
		}
		return false, err
	}

	go d.periodicallyRenewLease()
	return true, nil
}

// StopHeartbeat stops sending lock renewal heartbeats, consider using RemoveLock
func (d *DynamoDBLockClient) StopHeartbeat() {
	d.sendHeartbeats = false
	return
}

// RemoveLock removes the existing lock
func (d *DynamoDBLockClient) RemoveLock() error {
	d.StopHeartbeat()
	return d.dynamoRemoveLock()
}

// HasLock returns true if the lock is still valid
func (d *DynamoDBLockClient) HasLock() (bool, error) {
	return d.dynamoHasLock()
}

// LockError returns a lock error if the heartbeat thread found one
func (d *DynamoDBLockClient) LockError() error {
	return d.lockError
}

func (d *DynamoDBLockClient) periodicallyRenewLease() {

	d.sendHeartbeats = true

	for true {
		d.heartbeatCount++

		d.logger().Debugf("Waiting for %s - %#v", d.HeartbeatPeriod, d)
		time.Sleep(d.HeartbeatPeriod)

		if !d.sendHeartbeats {
			existing, _ := d.dynamoExamineLock()
			if existing != nil {
				d.logger().Debugf("No longer sending heartbeats for: %#v", existing)
			}
			break
		}

		d.logger().Debugf("Renewing lease on lock '%s' / %s for %s (after %d)", d.LockName, d.Identifier, d.LeaseDuration, d.heartbeatCount)
		err := d.dynamoGetLock()
		if err != nil {
			d.lockError = err // save so we can return why later
			d.logger().Debug(err)
			break
		}

	}
	d.logger().Debugf("Stopping heartbeats for %s / %s (after %d)", d.LockName, d.Identifier, d.heartbeatCount)
}

func (d *DynamoDBLockClient) logger() *logrus.Logger {
	if d.Logger == nil {
		d.Logger = logrus.StandardLogger()
	}

	return d.Logger
}
