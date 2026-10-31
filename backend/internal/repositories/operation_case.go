package repositories

import (
	"context"
	"strings"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OperationCases struct{ db *gorm.DB }

func NewOperationCases(db *gorm.DB) *OperationCases { return &OperationCases{db: db} }

const operationCaseList = `WITH active_sources AS (
  SELECT * FROM operation_case_sources
  WHERE kind <> ? OR created_at < NOW() - (? * INTERVAL '1 second')
), sources AS (
  SELECT kind,source_id,attempt_id,company_id,created_at,source_changed_at,TRUE AS source_active FROM active_sources
  UNION ALL
  SELECT c.kind,c.source_id,c.attempt_id,c.company_id,c.created_at,c.updated_at,FALSE
  FROM operation_cases c
  WHERE NOT EXISTS (SELECT 1 FROM active_sources s WHERE s.kind=c.kind AND s.source_id=c.source_id)
), joined AS (
  SELECT s.*,co.name AS company_name,c.status AS case_status,c.owner_id,c.id IS NOT NULL AS tracked,
    COALESCE(u.name,'') AS owner_name,COALESCE(c.note,'') AS note,
    (s.source_active AND s.kind IN (?,?) AND c.status=? AND s.source_changed_at>c.updated_at) AS recurred
  FROM sources s JOIN companies co ON co.id=s.company_id
  LEFT JOIN operation_cases c ON c.kind=s.kind AND c.source_id=s.source_id
  LEFT JOIN system_users u ON u.id=c.owner_id
), cases AS (
  SELECT kind,source_id,attempt_id,company_id,created_at,source_active,company_name,tracked,
    CASE WHEN recurred THEN ? ELSE COALESCE(case_status,?) END AS status,
    CASE WHEN recurred THEN NULL ELSE owner_id END AS owner_id,
    CASE WHEN recurred THEN '' ELSE owner_name END AS owner_name,
    CASE WHEN recurred THEN '' ELSE note END AS note
  FROM joined
) `

func (r *OperationCases) List(ctx context.Context, q dtos.OperationCaseInput) (dtos.Page[dtos.OperationCaseItem], error) {
	result := dtos.Page[dtos.OperationCaseItem]{Items: []dtos.OperationCaseItem{}, Page: q.Page, PageSize: q.PageSize}
	where := []string{"1=1"}
	args := []any{models.OperationOverdueSchedule, int(models.OperationScheduleGrace.Seconds()),
		models.OperationDLRCallback, models.OperationMOCallback, models.OperationCaseCompleted,
		models.OperationCaseOpen, models.OperationCaseOpen}
	if q.CompanyID != 0 {
		if !security.AllowsCompany(ctx, q.CompanyID) {
			return result, gorm.ErrRecordNotFound
		}
		where = append(where, "company_id=?")
		args = append(args, q.CompanyID)
	} else if scope := security.Scope(ctx); scope.Restricted {
		if len(scope.IDs) == 0 {
			return result, nil
		}
		where = append(where, "company_id IN ?")
		args = append(args, scope.IDs)
	}
	if q.Kind != "" {
		where = append(where, "kind=?")
		args = append(args, q.Kind)
	}
	if q.Status != "" {
		where = append(where, "status=?")
		args = append(args, q.Status)
	}
	if q.Search != "" {
		where = append(where, "(company_name ILIKE ? OR source_id::text=?)")
		term := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(q.Search)
		args = append(args, "%"+term+"%", q.Search)
	}
	filter := " WHERE " + strings.Join(where, " AND ")
	db := r.db.WithContext(ctx)
	if err := db.Raw(operationCaseList+"SELECT COUNT(*) FROM cases"+filter, args...).Scan(&result.Total).Error; err != nil {
		return result, err
	}
	rows := append(append([]any{}, args...), q.PageSize, (q.Page-1)*q.PageSize)
	order := "created_at DESC,kind,source_id DESC"
	if column, ok := map[string]string{"createdAt": "created_at", "companyName": "company_name", "kind": "kind", "status": "status", "sourceId": "source_id"}[q.Sort]; ok {
		direction := "DESC"
		if q.Direction == dtos.SortAscending {
			direction = "ASC"
		}
		order = column + " " + direction + ",source_id DESC"
	}
	err := db.Raw(operationCaseList+`SELECT kind,source_id,attempt_id,company_id,company_name,created_at,status,owner_id,owner_name,note,tracked,source_active
    FROM cases`+filter+` ORDER BY `+order+` LIMIT ? OFFSET ?`, rows...).Scan(&result.Items).Error
	return result, err
}

type operationCaseRow struct {
	ID        uint64
	CompanyID uint64
	Status    models.OperationCaseStatus
	OwnerID   *uint64
	UpdatedAt time.Time
}

func (r *OperationCases) Update(ctx context.Context, actor uint64, kind models.OperationCaseKind, sourceID uint64, status models.OperationCaseStatus, note string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`INSERT INTO operation_cases(kind,source_id,attempt_id,company_id)
			SELECT kind,source_id,attempt_id,company_id FROM operation_case_sources WHERE kind=? AND source_id=?
      AND (kind <> ? OR created_at < NOW() - (? * INTERVAL '1 second'))
      ON CONFLICT(kind,source_id) DO NOTHING`, kind, sourceID, models.OperationOverdueSchedule, int(models.OperationScheduleGrace.Seconds())).Error; err != nil {
			return err
		}
		var current operationCaseRow
		if err := tx.Table("operation_cases").Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id,company_id,status,owner_id,updated_at").Where("kind=? AND source_id=?", kind, sourceID).Take(&current).Error; err != nil {
			return err
		}
		if !security.AllowsCompany(ctx, current.CompanyID) {
			return gorm.ErrRecordNotFound
		}
		if current.Status == models.OperationCaseCompleted && (kind == models.OperationDLRCallback || kind == models.OperationMOCallback) {
			var source struct{ SourceChangedAt time.Time }
			result := tx.Raw("SELECT source_changed_at FROM operation_case_sources WHERE kind=? AND source_id=?", kind, sourceID).Scan(&source)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 1 && source.SourceChangedAt.After(current.UpdatedAt) {
				current.Status = models.OperationCaseOpen
				current.OwnerID = nil
			}
		}
		if current.Status == models.OperationCaseCompleted {
			return apperrors.Wrap(apperrors.CodeConflict, nil)
		}
		if status == models.OperationCaseInProgress {
			if current.OwnerID != nil && *current.OwnerID != actor {
				return apperrors.Wrap(apperrors.CodeConflict, nil)
			}
		} else if current.Status != models.OperationCaseInProgress || current.OwnerID == nil || *current.OwnerID != actor {
			return apperrors.Wrap(apperrors.CodeConflict, nil)
		}
		var owner any = actor
		if status == models.OperationCaseOpen {
			owner = nil
		}
		if err := tx.Table("operation_cases").Where("id=?", current.ID).Updates(map[string]any{
			"status": status, "owner_id": owner, "note": note, "updated_at": time.Now().UTC(),
		}).Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO operation_case_events(case_id,actor_id,status,note) VALUES(?,?,?,?)`, current.ID, actor, status, note).Error
	})
}

func (r *OperationCases) History(ctx context.Context, kind models.OperationCaseKind, sourceID uint64) ([]dtos.OperationCaseEvent, error) {
	var current operationCaseRow
	if err := r.db.WithContext(ctx).Table("operation_cases").Select("id,company_id").Where("kind=? AND source_id=?", kind, sourceID).Take(&current).Error; err != nil {
		return nil, err
	}
	if !security.AllowsCompany(ctx, current.CompanyID) {
		return nil, gorm.ErrRecordNotFound
	}
	items := []dtos.OperationCaseEvent{}
	err := r.db.WithContext(ctx).Table("operation_case_events AS e").Joins("JOIN system_users AS u ON u.id=e.actor_id").
		Select("u.name AS actor_name,e.status,e.note,e.created_at").Where("e.case_id=?", current.ID).Order("e.id ASC").Find(&items).Error
	return items, err
}
