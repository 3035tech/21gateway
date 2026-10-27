package services

import (
	"context"
	"fmt"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"strings"
	"testing"
)

func BenchmarkSMSNormalization(b *testing.B) {
	text := strings.Repeat("Olá! 😊 ", 20)
	b.ReportAllocs()
	for b.Loop() {
		if _, code := prepareSMSText(text); code != apperrors.CodeNone {
			b.Fatal(code)
		}
	}
}
func BenchmarkSMSBatch200(b *testing.B) {
	messages := make([]dtos.LegacySMS, whitelabel.MaximumBatchMessages)
	outcomes := make([]whitelabel.BatchResult, len(messages))
	for i := range messages {
		messages[i] = validSMS()
		messages[i].MessageText = "Olá, ação! 😊"
		outcomes[i].MessageID = fmt.Sprintf("550e8400-e29b-41d4-a716-%012d", i+1)
	}
	b.ReportAllocs()
	for b.Loop() {
		provider := &batchStub{stubProvider: stubProvider{enabled: true}, outcomes: outcomes}
		service := NewSMSServiceWithLimiter(&batchMemory{}, provider, &countLimiter{allowed: true})
		result, err := service.Send(context.Background(), 7, messages)
		if err != nil || len(result) != len(messages) {
			b.Fatal("batch failed", err)
		}
	}
}
