package lockclient

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

func (d *DynamoDBLockClient) dynamoGetLock() error {

	condition := expression.Or(
		expression.Name("key").NotEqual(expression.Value(d.LockName)),
		expression.Name("expiry").LessThan(expression.Value(time.Now().UnixNano())),
		expression.Name("identifier").Equal(expression.Value(d.Identifier)),
	)

	expr, err := expression.NewBuilder().WithCondition(condition).Build()
	if err != nil {
		return err
	}

	var itemValue map[string]*dynamodb.AttributeValue

	itemValue, err = dynamodbattribute.MarshalMap(map[string]interface{}{
		"expiry":     time.Now().UnixNano() + int64(d.LeaseDuration/time.Nanosecond),
		"key":        d.LockName,
		"identifier": d.Identifier,
		"heartbeats": d.heartbeatCount,
	})

	if d.heartbeatCount == 0 {
		itemValue["set_at"] = &dynamodb.AttributeValue{
			S: aws.String(time.Now().Format(time.RFC3339)),
		}
	}

	input := &dynamodb.PutItemInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ConditionExpression:       expr.Condition(),
		TableName:                 aws.String(d.TableName),
		Item:                      itemValue,
	}

	_, err = d.Client.PutItem(input)
	if err != nil {
		return err
	}
	return nil

}

func (d *DynamoDBLockClient) dynamoExamineLock() (map[string]*dynamodb.AttributeValue, error) {
	itemValue, err := dynamodbattribute.MarshalMap(map[string]interface{}{
		"key": d.LockName,
	})
	input := &dynamodb.GetItemInput{
		TableName: aws.String(d.TableName),
		Key:       itemValue,
	}

	response, err := d.Client.GetItem(input)
	if err != nil {
		return nil, err
	}
	return response.Item, nil
}

func (d *DynamoDBLockClient) dynamoRemoveLock() error {

	condition := expression.Or(
		expression.Name("key").Equal(expression.Value(d.LockName)),
		expression.Name("identifier").Equal(expression.Value(d.Identifier)),
	)

	expr, err := expression.NewBuilder().WithCondition(condition).Build()
	if err != nil {
		return err
	}

	itemValue, err := dynamodbattribute.MarshalMap(map[string]interface{}{
		"key": d.LockName,
	})

	input := &dynamodb.DeleteItemInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ConditionExpression:       expr.Condition(),
		TableName:                 aws.String(d.TableName),
		Key:                       itemValue,
	}

	_, err = d.Client.DeleteItem(input)
	if err != nil {
		return err
	}
	return nil

}

func (d *DynamoDBLockClient) dynamoHasLock() (bool, error) {

	condition := expression.And(
		expression.Name("key").Equal(expression.Value(d.LockName)),
		expression.Name("expiry").GreaterThan(expression.Value(time.Now().UnixNano())),
		expression.Name("identifier").Equal(expression.Value(d.Identifier)),
	)

	proj := expression.NamesList(expression.Name("key"), expression.Name("identifier"), expression.Name("expiry"))
	expr, err := expression.NewBuilder().WithFilter(condition).WithProjection(proj).Build()
	if err != nil {
		return false, err
	}

	params := &dynamodb.ScanInput{
		ConsistentRead:            aws.Bool(true),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		TableName:                 aws.String(d.TableName),
	}

	result, err := d.Client.Scan(params)
	if err != nil {
		return false, err
	}
	return *result.Count > 0, d.lockError

}
