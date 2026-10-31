package services

import (
	"testing"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
)

func TestPostgresOperationCasesScopeAndHandoff(t *testing.T) {
	ctx, db := postgresFixture(t)
	first := models.Company{Name: "Operational first", Active: true}
	second := models.Company{Name: "Operational second", Active: true}
	var firstAttempt uint64
	for _, company := range []*models.Company{&first, &second} {
		if err := db.Create(company).Error; err != nil {
			t.Fatal(err)
		}
	}
	owner := models.SystemUser{Name: "Operator one", Email: "operator-one@example.test", PasswordHash: "unused", Role: models.RoleOperator, Active: true}
	other := models.SystemUser{Name: "Operator two", Email: "operator-two@example.test", PasswordHash: "unused", Role: models.RoleOperator, Active: true}
	for _, user := range []*models.SystemUser{&owner, &other} {
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}
	for index, company := range []*models.Company{&first, &second} {
		credential := models.APICredential{CompanyID: &company.ID, Username: "work-credential-" + company.Name, PasswordHash: "unused", Active: true}
		if err := db.Create(&credential).Error; err != nil {
			t.Fatal(err)
		}
		attempt := models.SMSAttempt{CompanyID: &company.ID, CredentialID: credential.ID, Destination: "5511999999999", Encoding: models.SMSEncodingGSM7, State: models.SubmissionUnknown}
		if err := db.Create(&attempt).Error; err != nil {
			t.Fatal(err)
		}
		if index == 0 {
			firstAttempt = attempt.ID
		}
	}
	repo := repositories.NewOperationCases(db)
	svc := NewOperationCaseService(repo)
	scoped := security.WithCompanyScope(ctx, []uint64{first.ID})
	query := dtos.OperationCaseInput{Page: 1, PageSize: 25}
	page, err := svc.List(scoped, query)
	if err != nil {
		_, rawErr := repo.List(scoped, query)
		t.Fatalf("repository list: %v", rawErr)
	}
	if err != nil || page.Total != 1 || len(page.Items) != 1 || page.Items[0].CompanyID != first.ID {
		t.Fatalf("scoped queue: %+v %v", page, err)
	}
	item := page.Items[0]
	if err := svc.Update(scoped, owner.ID, item.Kind, item.SourceID, models.OperationCaseInProgress, "Investigando retorno"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Update(scoped, other.ID, item.Kind, item.SourceID, models.OperationCaseCompleted, "Outro operador"); err == nil {
		t.Fatal("another operator completed claimed case")
	}
	if err := svc.Update(scoped, owner.ID, item.Kind, item.SourceID, models.OperationCaseCompleted, ""); err == nil {
		t.Fatal("completion without note")
	}
	if err := svc.Update(scoped, owner.ID, item.Kind, item.SourceID, models.OperationCaseOpen, "Passando para outro turno"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Update(scoped, other.ID, item.Kind, item.SourceID, models.OperationCaseInProgress, "Retomando análise"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Update(scoped, owner.ID, item.Kind, item.SourceID, models.OperationCaseCompleted, "Tentativa sem posse"); err == nil {
		t.Fatal("previous owner completed case")
	}
	if err := svc.Update(scoped, other.ID, item.Kind, item.SourceID, models.OperationCaseCompleted, "Conferido com fornecedor"); err != nil {
		t.Fatal(err)
	}
	page, err = svc.List(scoped, dtos.OperationCaseInput{Page: 1, PageSize: 25, Status: models.OperationCaseCompleted})
	if err != nil || page.Total != 1 || page.Items[0].OwnerID == nil || *page.Items[0].OwnerID != other.ID {
		t.Fatalf("completed queue: %+v %v", page, err)
	}
	history, err := svc.History(scoped, item.Kind, item.SourceID)
	if err != nil || len(history) != 4 || history[0].ActorName != owner.Name {
		t.Fatalf("history: %+v %v", history, err)
	}
	if _, err := svc.List(scoped, dtos.OperationCaseInput{Page: 1, PageSize: 25, CompanyID: second.ID}); err == nil {
		t.Fatal("out-of-scope company filter succeeded")
	}
	if _, err := svc.History(security.WithCompanyScope(ctx, []uint64{second.ID}), item.Kind, item.SourceID); err == nil {
		t.Fatal("out-of-scope history succeeded")
	}
	if err := svc.Update(security.WithCompanyScope(ctx, []uint64{second.ID}), other.ID, item.Kind, item.SourceID, models.OperationCaseInProgress, "fora do escopo"); err == nil {
		t.Fatal("out-of-scope update succeeded")
	}
	providerID := "550e8400-e29b-41d4-a716-446655440000"
	event := models.DeliveryEvent{EventKey: "operation-case-dlr", ProviderMessageID: providerID, Status: models.DeliverySent, OccurredAt: time.Now().UTC(), AttemptID: &firstAttempt}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	var credential models.APICredential
	if err := db.Where("company_id=?", first.ID).First(&credential).Error; err != nil {
		t.Fatal(err)
	}
	job := models.CallbackJob{EventID: event.ID, AttemptID: firstAttempt, CredentialID: credential.ID, URL: "https://example.test/callback", State: models.CallbackFailed, Status: models.DeliverySent, AvailableAt: time.Now().UTC()}
	if err := db.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.Update(scoped, owner.ID, models.OperationDLRCallback, job.ID, models.OperationCaseInProgress, "Checando callback"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Update(scoped, owner.ID, models.OperationDLRCallback, job.ID, models.OperationCaseCompleted, "Ação concluída"); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&job).Update("state", models.CallbackPending).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&job).Update("state", models.CallbackFailed).Error; err != nil {
		t.Fatal(err)
	}
	page, err = svc.List(scoped, dtos.OperationCaseInput{Page: 1, PageSize: 25, Kind: models.OperationDLRCallback, Status: models.OperationCaseOpen})
	if err != nil || page.Total != 1 || page.Items[0].OwnerID != nil {
		t.Fatalf("recurrent callback not reopened: %+v %v", page, err)
	}
	if err := svc.Update(scoped, other.ID, models.OperationDLRCallback, job.ID, models.OperationCaseInProgress, "Nova falha"); err != nil {
		t.Fatal(err)
	}
	moEvent := models.MOEvent{EventKey: "operation-case-mo", CredentialID: credential.ID, ProviderMessageID: providerID, AttemptID: &firstAttempt, EncryptedMessage: "encrypted", Fingerprint: "test-fingerprint", PhoneNumber: "5511999999999", ReceivedAt: time.Now().UTC()}
	if err := db.Create(&moEvent).Error; err != nil {
		t.Fatal(err)
	}
	moJob := models.MOCallbackJob{EventID: moEvent.ID, AttemptID: firstAttempt, CredentialID: credential.ID, URL: "https://example.test/mo", State: models.CallbackFailed, AvailableAt: time.Now().UTC()}
	if err := db.Create(&moJob).Error; err != nil {
		t.Fatal(err)
	}
	page, err = svc.List(scoped, dtos.OperationCaseInput{Page: 1, PageSize: 25, Kind: models.OperationMOCallback})
	if err != nil || page.Total != 1 || page.Items[0].AttemptID != firstAttempt {
		t.Fatalf("MO queue: %+v %v", page, err)
	}
	due := time.Now().UTC().Add(-models.OperationScheduleGrace - time.Minute)
	scheduledAttempt := models.SMSAttempt{CompanyID: &first.ID, CredentialID: credential.ID, Destination: "5511999999999", Encoding: models.SMSEncodingGSM7, State: models.SubmissionPending, ScheduledAt: &due}
	if err := db.Create(&scheduledAttempt).Error; err != nil {
		t.Fatal(err)
	}
	scheduled := models.ScheduledSMS{AttemptID: scheduledAttempt.ID, CredentialID: credential.ID, EventKey: "550e8400-e29b-41d4-a716-446655440001", EncryptedMessage: "encrypted", DueAt: due, AvailableAt: due, State: models.ScheduleQueued}
	if err := db.Create(&scheduled).Error; err != nil {
		t.Fatal(err)
	}
	page, err = svc.List(scoped, dtos.OperationCaseInput{Page: 1, PageSize: 25, Kind: models.OperationOverdueSchedule})
	if err != nil || page.Total != 1 || page.Items[0].SourceID != scheduledAttempt.ID {
		t.Fatalf("schedule queue: %+v %v", page, err)
	}
}
