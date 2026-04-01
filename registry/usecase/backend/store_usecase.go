package main

import (
	"database/sql"
	"fmt"
	"strings"
)

type UseCaseFilter struct {
	Query    string
	Domain   string
	Industry string
	Risk     string
	Page     int
	Limit    int
}

type UseCaseListResult struct {
	UseCases []UseCase `json:"use_cases"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	Limit    int       `json:"limit"`
}

func (s *Store) ListUseCases(f UseCaseFilter) (*UseCaseListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 20
	}

	var where []string
	var args []any

	where = append(where, "uc.visibility = 'public'")
	where = append(where, "uc.status = 'published'")

	if f.Query != "" {
		where = append(where, "(uc.title LIKE ? OR uc.description LIKE ?)")
		q := "%" + f.Query + "%"
		args = append(args, q, q)
	}
	if f.Domain != "" {
		where = append(where, "uc.domain = ?")
		args = append(args, f.Domain)
	}
	if f.Industry != "" {
		where = append(where, "uc.industry_tags LIKE ?")
		args = append(args, "%"+f.Industry+"%")
	}
	if f.Risk != "" {
		where = append(where, "uc.risk_level = ?")
		args = append(args, f.Risk)
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	countQ := s.db.Rebind("SELECT COUNT(*) FROM use_cases uc " + whereClause)
	var total int
	if err := s.db.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count use cases: %w", err)
	}

	offset := (f.Page - 1) * f.Limit
	q := s.db.Rebind(`
		SELECT uc.id, uc.title, uc.slug, uc.description, uc.domain, uc.industry_tags,
		       uc.risk_level, uc.roi_potential, uc.status, uc.impl_status, uc.visibility,
		       uc.safety_pii, uc.safety_autonomous, uc.safety_blast_radius, uc.safety_oversight,
		       uc.forked_from, uc.fork_count, uc.author_id, uc.created_at, uc.updated_at,
		       u.id, u.email, u.name, u.avatar_url
		FROM use_cases uc
		JOIN users u ON u.id = uc.author_id
		` + whereClause + `
		ORDER BY uc.created_at DESC
		LIMIT ? OFFSET ?
	`)
	args = append(args, f.Limit, offset)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list use cases: %w", err)
	}
	defer rows.Close()

	var usecases []UseCase
	for rows.Next() {
		uc, err := scanUseCase(rows)
		if err != nil {
			return nil, fmt.Errorf("scan use case: %w", err)
		}
		usecases = append(usecases, uc)
	}

	return &UseCaseListResult{
		UseCases: usecases,
		Total:    total,
		Page:     f.Page,
		Limit:    f.Limit,
	}, nil
}

func (s *Store) GetUseCase(id string) (*UseCase, error) {
	q := s.db.Rebind(`
		SELECT uc.id, uc.title, uc.slug, uc.description, uc.domain, uc.industry_tags,
		       uc.risk_level, uc.roi_potential, uc.status, uc.impl_status, uc.visibility,
		       uc.safety_pii, uc.safety_autonomous, uc.safety_blast_radius, uc.safety_oversight,
		       uc.forked_from, uc.fork_count, uc.author_id, uc.created_at, uc.updated_at,
		       u.id, u.email, u.name, u.avatar_url
		FROM use_cases uc
		JOIN users u ON u.id = uc.author_id
		WHERE uc.id = ?
	`)
	row := s.db.QueryRow(q, id)
	uc, err := scanUseCaseRow(row)
	if err != nil {
		return nil, fmt.Errorf("get use case: %w", err)
	}
	return &uc, nil
}

func (s *Store) CreateUseCase(uc *UseCase) error {
	q := s.db.Rebind(`
		INSERT INTO use_cases (id, title, slug, description, domain, industry_tags,
			risk_level, roi_potential, status, impl_status, visibility,
			safety_pii, safety_autonomous, safety_blast_radius, safety_oversight,
			forked_from, fork_count, author_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	_, err := s.db.Exec(q,
		uc.ID, uc.Title, uc.Slug, uc.Description, uc.Domain, uc.IndustryTags,
		uc.RiskLevel, uc.ROIPotential, uc.Status, uc.ImplStatus, uc.Visibility,
		uc.SafetyPII, uc.SafetyAutonomous, uc.SafetyBlastRadius, uc.SafetyOversight,
		uc.ForkedFrom, uc.ForkCount, uc.AuthorID,
	)
	if err != nil {
		return fmt.Errorf("create use case: %w", err)
	}
	return nil
}

func (s *Store) UpdateUseCase(uc *UseCase) error {
	q := s.db.Rebind(`
		UPDATE use_cases SET
			title = ?, slug = ?, description = ?, domain = ?, industry_tags = ?,
			risk_level = ?, roi_potential = ?, status = ?, impl_status = ?, visibility = ?,
			safety_pii = ?, safety_autonomous = ?, safety_blast_radius = ?, safety_oversight = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND author_id = ?
	`)
	result, err := s.db.Exec(q,
		uc.Title, uc.Slug, uc.Description, uc.Domain, uc.IndustryTags,
		uc.RiskLevel, uc.ROIPotential, uc.Status, uc.ImplStatus, uc.Visibility,
		uc.SafetyPII, uc.SafetyAutonomous, uc.SafetyBlastRadius, uc.SafetyOversight,
		uc.ID, uc.AuthorID,
	)
	if err != nil {
		return fmt.Errorf("update use case: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("use case not found or not owned by user")
	}
	return nil
}

func (s *Store) DeleteUseCase(id, authorID string) error {
	q := s.db.Rebind(`DELETE FROM use_cases WHERE id = ? AND author_id = ?`)
	result, err := s.db.Exec(q, id, authorID)
	if err != nil {
		return fmt.Errorf("delete use case: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("use case not found or not owned by user")
	}
	return nil
}

func (s *Store) ForkUseCase(originalID, newID, newSlug, authorID string) (*UseCase, error) {
	original, err := s.GetUseCase(originalID)
	if err != nil {
		return nil, fmt.Errorf("get original: %w", err)
	}

	fork := &UseCase{
		ID:                newID,
		Title:             original.Title,
		Slug:              newSlug,
		Description:       original.Description,
		Domain:            original.Domain,
		IndustryTags:      original.IndustryTags,
		RiskLevel:         original.RiskLevel,
		ROIPotential:      original.ROIPotential,
		Status:            "draft",
		ImplStatus:        "evaluating",
		Visibility:        "private",
		SafetyPII:         original.SafetyPII,
		SafetyAutonomous:  original.SafetyAutonomous,
		SafetyBlastRadius: original.SafetyBlastRadius,
		SafetyOversight:   original.SafetyOversight,
		ForkedFrom:        &originalID,
		AuthorID:          authorID,
	}

	if err := s.CreateUseCase(fork); err != nil {
		return nil, fmt.Errorf("create fork: %w", err)
	}

	incrQ := s.db.Rebind(`UPDATE use_cases SET fork_count = fork_count + 1 WHERE id = ?`)
	s.db.Exec(incrQ, originalID)

	return s.GetUseCase(newID)
}

func (s *Store) ListUserUseCases(authorID string) ([]UseCase, error) {
	q := s.db.Rebind(`
		SELECT uc.id, uc.title, uc.slug, uc.description, uc.domain, uc.industry_tags,
		       uc.risk_level, uc.roi_potential, uc.status, uc.impl_status, uc.visibility,
		       uc.safety_pii, uc.safety_autonomous, uc.safety_blast_radius, uc.safety_oversight,
		       uc.forked_from, uc.fork_count, uc.author_id, uc.created_at, uc.updated_at,
		       u.id, u.email, u.name, u.avatar_url
		FROM use_cases uc
		JOIN users u ON u.id = uc.author_id
		WHERE uc.author_id = ?
		ORDER BY uc.created_at DESC
	`)
	rows, err := s.db.Query(q, authorID)
	if err != nil {
		return nil, fmt.Errorf("list user use cases: %w", err)
	}
	defer rows.Close()

	var usecases []UseCase
	for rows.Next() {
		uc, err := scanUseCase(rows)
		if err != nil {
			return nil, fmt.Errorf("scan use case: %w", err)
		}
		usecases = append(usecases, uc)
	}
	return usecases, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUseCaseFields(s scanner) (UseCase, error) {
	var uc UseCase
	var industryTags, blastRadius, oversight, forkedFrom, authorAvatar sql.NullString
	var author User

	err := s.Scan(
		&uc.ID, &uc.Title, &uc.Slug, &uc.Description, &uc.Domain, &industryTags,
		&uc.RiskLevel, &uc.ROIPotential, &uc.Status, &uc.ImplStatus, &uc.Visibility,
		&uc.SafetyPII, &uc.SafetyAutonomous, &blastRadius, &oversight,
		&forkedFrom, &uc.ForkCount, &uc.AuthorID, &uc.CreatedAt, &uc.UpdatedAt,
		&author.ID, &author.Email, &author.Name, &authorAvatar,
	)
	if err != nil {
		return uc, err
	}

	if industryTags.Valid {
		uc.IndustryTags = industryTags.String
	}
	if blastRadius.Valid {
		uc.SafetyBlastRadius = blastRadius.String
	}
	if oversight.Valid {
		uc.SafetyOversight = oversight.String
	}
	if forkedFrom.Valid {
		uc.ForkedFrom = &forkedFrom.String
	}
	if authorAvatar.Valid {
		author.AvatarURL = authorAvatar.String
	}
	uc.Author = &author

	return uc, nil
}

func scanUseCase(rows *sql.Rows) (UseCase, error) {
	return scanUseCaseFields(rows)
}

func scanUseCaseRow(row *sql.Row) (UseCase, error) {
	return scanUseCaseFields(row)
}
