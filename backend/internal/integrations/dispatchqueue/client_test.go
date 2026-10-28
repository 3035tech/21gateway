package dispatchqueue

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

const testURL = "https://sqs.us-east-1.amazonaws.com/123456789012/scheduled"
const testBatch = "550e8400-e29b-41d4-a716-446655440001"

type fakeSQS struct {
	sent     *sqs.SendMessageInput
	received *sqs.ReceiveMessageInput
	deleted  *sqs.DeleteMessageInput
	deferred *sqs.ChangeMessageVisibilityInput
	probed   *sqs.GetQueueAttributesInput
}

func (f *fakeSQS) SendMessage(_ context.Context, in *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	f.sent = in
	return &sqs.SendMessageOutput{}, nil
}
func (f *fakeSQS) ReceiveMessage(_ context.Context, in *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	f.received = in
	return &sqs.ReceiveMessageOutput{Messages: []types.Message{{Body: aws.String(`{"version":1,"batchId":"` + testBatch + `"}`), ReceiptHandle: aws.String("private-receipt")}}}, nil
}
func (f *fakeSQS) DeleteMessage(_ context.Context, in *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	f.deleted = in
	return &sqs.DeleteMessageOutput{}, nil
}
func (f *fakeSQS) ChangeMessageVisibility(_ context.Context, in *sqs.ChangeMessageVisibilityInput, _ ...func(*sqs.Options)) (*sqs.ChangeMessageVisibilityOutput, error) {
	f.deferred = in
	return &sqs.ChangeMessageVisibilityOutput{}, nil
}
func (f *fakeSQS) GetQueueAttributes(_ context.Context, in *sqs.GetQueueAttributesInput, _ ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
	f.probed = in
	return &sqs.GetQueueAttributesOutput{}, nil
}

func TestQueueProbeUsesConfiguredQueue(t *testing.T) {
	api := &fakeSQS{}
	if err := NewWithAPI(api, testURL).Probe(context.Background()); err != nil {
		t.Fatal(err)
	}
	if api.probed == nil || aws.ToString(api.probed.QueueUrl) != testURL {
		t.Fatal("probe did not inspect the configured queue")
	}
}
func TestSQSReferenceOnlyAndBoundedReceive(t *testing.T) {
	api := &fakeSQS{}
	client := NewWithAPI(api, testURL)
	ctx := context.Background()
	if err := client.Publish(ctx, testBatch); err != nil {
		t.Fatal(err)
	}
	var ref map[string]any
	if err := json.Unmarshal([]byte(aws.ToString(api.sent.MessageBody)), &ref); err != nil {
		t.Fatal(err)
	}
	if len(ref) != 2 || ref["batchId"] != testBatch || ref["version"] != float64(ReferenceVersion) || aws.ToString(api.sent.QueueUrl) != testURL {
		t.Fatal("queue payload must contain only a durable reference")
	}
	messages, err := client.Receive(ctx)
	if err != nil || len(messages) != 1 {
		t.Fatal(err)
	}
	if api.received.MaxNumberOfMessages != 1 || api.received.WaitTimeSeconds != LongPollSeconds || api.received.VisibilityTimeout != VisibilitySeconds {
		t.Fatal("unbounded polling or prefetch")
	}
	if err := client.Defer(ctx, messages[0]); err != nil {
		t.Fatal(err)
	}
	if api.deferred.VisibilityTimeout != DeferredSeconds || aws.ToString(api.deferred.ReceiptHandle) != "private-receipt" {
		t.Fatal("incorrect deferred delivery")
	}
	if err := client.Ack(ctx, messages[0]); err != nil {
		t.Fatal(err)
	}
	if aws.ToString(api.deleted.ReceiptHandle) != "private-receipt" {
		t.Fatal("incorrect receipt acknowledged")
	}
}
func TestReferenceRejectsPoisonPayloads(t *testing.T) {
	for _, body := range []string{`{}`, `{"version":2,"batchId":"` + testBatch + `"}`, `{"version":1,"batchId":"bad"}`, `{"version":1,"batchId":"00000000-0000-0000-0000-000000000000"}`, `{"version":1,"batchId":"` + testBatch + `","message":"private"}`, `{"version":1,"batchId":"` + testBatch + `"} {}`, strings.Repeat("x", 257)} {
		if _, err := ParseReference(body); err == nil {
			t.Fatal("poison reference accepted")
		}
	}
	if id, err := ParseReference(`{"version":1,"batchId":"` + testBatch + `"}`); err != nil || id != testBatch {
		t.Fatal("valid reference rejected")
	}
	api := &fakeSQS{}
	if err := NewWithAPI(api, testURL).Publish(context.Background(), "invalid"); err == nil || api.sent != nil {
		t.Fatal("invalid reference reached SDK")
	}
}
func TestSendReferenceContainsOnlyAttemptID(t *testing.T) {
	api := &fakeSQS{}
	client := NewWithAPI(api, testURL)
	if err := client.PublishSend(context.Background(), 12345); err != nil {
		t.Fatal(err)
	}
	canonical := `{"version":2,"sendId":"12345"}`
	if aws.ToString(api.sent.MessageBody) != canonical {
		t.Fatal("unexpected SMS queue payload")
	}
	if id, err := ParseSendReference(canonical); err != nil || id != 12345 {
		t.Fatal("valid SMS reference rejected", err)
	}
	for _, body := range []string{`{"version":1,"sendId":"12345"}`, `{"version":2,"sendId":"0"}`, `{"version":2,"sendId":"-1"}`, `{"version":2,"sendId":"12345","destination":"11987654321"}`, canonical + ` {}`, strings.Repeat("a", 257)} {
		if _, err := ParseSendReference(body); err == nil {
			t.Fatal("invalid SMS reference accepted")
		}
	}
	if err := client.PublishSend(context.Background(), 0); err == nil {
		t.Fatal("zero ID published")
	}
}

type sdkHTTP func(*http.Request) (*http.Response, error)

func (f sdkHTTP) Do(r *http.Request) (*http.Response, error) { return f(r) }
func TestOfficialSDKPublishesThroughMockHTTPOnly(t *testing.T) {
	calls := 0
	transport := sdkHTTP(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodPost || r.URL.Scheme != "https" || r.URL.Host != "sqs.us-east-1.amazonaws.com" {
			t.Fatal("unexpected SDK destination")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		raw, ok := body["MessageBody"].(string)
		if !ok {
			t.Fatal("missing SQS reference")
		}
		if id, err := ParseReference(raw); err != nil || id != testBatch {
			t.Fatal("SDK changed reference")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/x-amz-json-1.0"}}, Body: io.NopCloser(strings.NewReader(`{"MessageId":"mock-message"}`))}, nil
	})
	client := NewWithAPI(sqs.NewFromConfig(aws.Config{Region: "us-east-1", Credentials: aws.AnonymousCredentials{}, HTTPClient: transport}), testURL)
	if err := client.Publish(context.Background(), testBatch); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("unexpected SDK calls", calls)
	}
}
