package main

import "fmt"

func (s *Store) UpsertUser(id, email, name, avatarURL string) (*User, error) {
	q := s.db.Rebind(`
		INSERT INTO users (id, email, name, avatar_url)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(email) DO UPDATE SET
			name = excluded.name,
			avatar_url = excluded.avatar_url,
			updated_at = CURRENT_TIMESTAMP
	`)
	_, err := s.db.Exec(q, id, email, name, avatarURL)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return s.GetUserByEmail(email)
}

func (s *Store) GetUser(id string) (*User, error) {
	q := s.db.Rebind(`SELECT id, email, name, avatar_url, org_name, created_at, updated_at FROM users WHERE id = ?`)
	var u User
	var avatarURL, orgName *string
	err := s.db.QueryRow(q, id).Scan(&u.ID, &u.Email, &u.Name, &avatarURL, &orgName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if avatarURL != nil {
		u.AvatarURL = *avatarURL
	}
	if orgName != nil {
		u.OrgName = *orgName
	}
	return &u, nil
}

func (s *Store) GetUserByEmail(email string) (*User, error) {
	q := s.db.Rebind(`SELECT id, email, name, avatar_url, org_name, created_at, updated_at FROM users WHERE email = ?`)
	var u User
	var avatarURL, orgName *string
	err := s.db.QueryRow(q, email).Scan(&u.ID, &u.Email, &u.Name, &avatarURL, &orgName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	if avatarURL != nil {
		u.AvatarURL = *avatarURL
	}
	if orgName != nil {
		u.OrgName = *orgName
	}
	return &u, nil
}

func (s *Store) UpdateUserOrg(id, orgName string) error {
	q := s.db.Rebind(`UPDATE users SET org_name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)
	_, err := s.db.Exec(q, orgName, id)
	if err != nil {
		return fmt.Errorf("update user org: %w", err)
	}
	return nil
}
