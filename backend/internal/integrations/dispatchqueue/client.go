package dispatchqueue

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
)

const (
	ReferenceVersion            = 1
	LongPollSeconds       int32 = 20
	VisibilitySeconds     int32 = 90
	DeferredSeconds       int32 = 1
	maximumReferenceBytes       = 256
	sdkHTTPTimeout              = 30 * time.Second
)

type Delivery struct{ Body, Receipt string }
type SQSAPI interface {
	SendMessage(context.Context, *sqs.SendMessageInput, ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
	ReceiveMessage(context.Context, *sqs.ReceiveMessageInput, ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(context.Context, *sqs.DeleteMessageInput, ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
	ChangeMessageVisibility(context.Context, *sqs.ChangeMessageVisibilityInput, ...func(*sqs.Options)) (*sqs.ChangeMessageVisibilityOutput, error)
	GetQueueAttributes(context.Context, *sqs.GetQueueAttributesInput, ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error)
}
type Client struct {
	api   SQSAPI
	url   string
	close func()
}
type Reference struct {
	Version int    `json:"version"`
	BatchID string `json:"batchId"`
}
type SendReference struct {
	Version int    `json:"version"`
	SendID  string `json:"sendId"`
}

const SendReferenceVersion = 2

func New(ctx context.Context, region, queueURL string) (*Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = 16
	transport.MaxIdleConnsPerHost = 16
	// The standard credential chain supports task roles; never embed IAM keys in app config.
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region), awsconfig.WithHTTPClient(&http.Client{Transport: transport, Timeout: sdkHTTPTimeout}), awsconfig.WithRetryMaxAttempts(3))
	if err != nil {
		return nil, err
	}
	client := NewWithAPI(sqs.NewFromConfig(cfg), queueURL)
	client.close = transport.CloseIdleConnections
	return client, nil
}
func NewWithAPI(api SQSAPI, queueURL string) *Client {
	return &Client{api: api, url: queueURL, close: func() {}}
}
func (c *Client) Publish(ctx context.Context, id string) error {
	if parsed, err := uuid.Parse(id); err != nil || parsed == uuid.Nil {
		return errors.New("invalid dispatch reference")
	}
	body, err := json.Marshal(Reference{Version: ReferenceVersion, BatchID: id})
	if err != nil {
		return err
	}
	_, err = c.api.SendMessage(ctx, &sqs.SendMessageInput{QueueUrl: aws.String(c.url), MessageBody: aws.String(string(body))})
	return err
}
func (c *Client) PublishSend(ctx context.Context, id uint64) error {
	if id == 0 {
		return errors.New("invalid send reference")
	}
	body, err := json.Marshal(SendReference{Version: SendReferenceVersion, SendID: strconv.FormatUint(id, 10)})
	if err != nil {
		return err
	}
	_, err = c.api.SendMessage(ctx, &sqs.SendMessageInput{QueueUrl: aws.String(c.url), MessageBody: aws.String(string(body))})
	return err
}
func (c *Client) Receive(ctx context.Context) ([]Delivery, error) {
	result, err := c.api.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{QueueUrl: aws.String(c.url), MaxNumberOfMessages: 1, WaitTimeSeconds: LongPollSeconds, VisibilityTimeout: VisibilitySeconds})
	if err != nil {
		return nil, err
	}
	deliveries := make([]Delivery, len(result.Messages))
	for i, m := range result.Messages {
		deliveries[i] = Delivery{Body: aws.ToString(m.Body), Receipt: aws.ToString(m.ReceiptHandle)}
	}
	return deliveries, nil
}
func (c *Client) Ack(ctx context.Context, delivery Delivery) error {
	_, err := c.api.DeleteMessage(ctx, &sqs.DeleteMessageInput{QueueUrl: aws.String(c.url), ReceiptHandle: aws.String(delivery.Receipt)})
	return err
}
func (c *Client) Defer(ctx context.Context, delivery Delivery) error {
	_, err := c.api.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{QueueUrl: aws.String(c.url), ReceiptHandle: aws.String(delivery.Receipt), VisibilityTimeout: DeferredSeconds})
	return err
}

// Probe verifies that the configured queue is reachable with the worker identity.
func (c *Client) Probe(ctx context.Context) error {
	_, err := c.api.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{QueueUrl: aws.String(c.url)})
	return err
}
func ParseReference(body string) (string, error) {
	if len(body) > maximumReferenceBytes {
		return "", errors.New("invalid dispatch reference")
	}
	var ref Reference
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&ref); err != nil {
		return "", errors.New("invalid dispatch reference")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return "", errors.New("invalid dispatch reference")
	}
	id, err := uuid.Parse(ref.BatchID)
	if ref.Version != ReferenceVersion || err != nil || id == uuid.Nil {
		return "", errors.New("invalid dispatch reference")
	}
	return id.String(), nil
}
func ParseSendReference(body string) (uint64, error) {
	if len(body) > maximumReferenceBytes {
		return 0, errors.New("invalid send reference")
	}
	var ref SendReference
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&ref) != nil || decoder.Decode(new(any)) != io.EOF || ref.Version != SendReferenceVersion {
		return 0, errors.New("invalid send reference")
	}
	id, err := strconv.ParseUint(ref.SendID, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid send reference")
	}
	return id, nil
}

func (c *Client) Close() { c.close() }
