package main

import (
	"testing"

	"github.com/73ai/openbotkit/store"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	db, err := store.Open(store.SQLiteConfig(":memory:"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	return NewStore(db)
}

func TestUserUpsertAndGet(t *testing.T) {
	st := testStore(t)

	user, err := st.UpsertUser("u1", "alice@example.com", "Alice", "https://example.com/avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "alice@example.com" {
		t.Fatalf("expected alice@example.com, got %s", user.Email)
	}

	got, err := st.GetUser(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Alice" {
		t.Fatalf("expected Alice, got %s", got.Name)
	}
	if got.AvatarURL != "https://example.com/avatar.png" {
		t.Fatalf("expected avatar, got %s", got.AvatarURL)
	}
}

func TestUserUpsertUpdatesExisting(t *testing.T) {
	st := testStore(t)

	st.UpsertUser("u1", "bob@example.com", "Bob", "")
	updated, err := st.UpsertUser("u1", "bob@example.com", "Robert", "https://new-avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Robert" {
		t.Fatalf("expected Robert, got %s", updated.Name)
	}
}

func TestUpdateUserOrg(t *testing.T) {
	st := testStore(t)

	st.UpsertUser("u1", "org@example.com", "Org User", "")
	if err := st.UpdateUserOrg("u1", "Acme Corp"); err != nil {
		t.Fatal(err)
	}

	user, _ := st.GetUser("u1")
	if user.OrgName != "Acme Corp" {
		t.Fatalf("expected Acme Corp, got %s", user.OrgName)
	}
}

func TestCreateAndGetUseCase(t *testing.T) {
	st := testStore(t)
	st.UpsertUser("u1", "test@example.com", "Test", "")

	uc := &UseCase{
		ID:           "uc1",
		Title:        "Test Use Case",
		Slug:         "test-use-case-uc1",
		Description:  "A test description",
		Domain:       "Engineering",
		RiskLevel:    "low",
		ROIPotential: "high",
		Status:       "published",
		ImplStatus:   "evaluating",
		Visibility:   "public",
		AuthorID:     "u1",
	}
	if err := st.CreateUseCase(uc); err != nil {
		t.Fatal(err)
	}

	got, err := st.GetUseCase("uc1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Test Use Case" {
		t.Fatalf("expected Test Use Case, got %s", got.Title)
	}
	if got.Author == nil || got.Author.Email != "test@example.com" {
		t.Fatal("expected author to be populated")
	}
}

func TestListUseCasesFilters(t *testing.T) {
	st := testStore(t)
	st.UpsertUser("u1", "test@example.com", "Test", "")

	for i, domain := range []string{"Engineering", "Sales", "Engineering"} {
		uc := &UseCase{
			ID:           newID(),
			Title:        "UC " + domain,
			Slug:         "uc-" + newID()[:8],
			Description:  "Description",
			Domain:       domain,
			RiskLevel:    "low",
			ROIPotential: "medium",
			Status:       "published",
			ImplStatus:   "evaluating",
			Visibility:   "public",
			AuthorID:     "u1",
		}
		_ = i
		st.CreateUseCase(uc)
	}

	result, err := st.ListUseCases(UseCaseFilter{Domain: "Engineering"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 {
		t.Fatalf("expected 2 engineering use cases, got %d", result.Total)
	}
}

func TestUpdateUseCase(t *testing.T) {
	st := testStore(t)
	st.UpsertUser("u1", "test@example.com", "Test", "")

	uc := &UseCase{
		ID: "uc1", Title: "Original", Slug: "original-uc1",
		Description: "Desc", Domain: "Sales",
		RiskLevel: "low", ROIPotential: "medium",
		Status: "published", ImplStatus: "evaluating",
		Visibility: "public", AuthorID: "u1",
	}
	st.CreateUseCase(uc)

	uc.Title = "Updated"
	uc.Slug = "updated-uc1"
	if err := st.UpdateUseCase(uc); err != nil {
		t.Fatal(err)
	}

	got, _ := st.GetUseCase("uc1")
	if got.Title != "Updated" {
		t.Fatalf("expected Updated, got %s", got.Title)
	}
}

func TestDeleteUseCase(t *testing.T) {
	st := testStore(t)
	st.UpsertUser("u1", "test@example.com", "Test", "")

	uc := &UseCase{
		ID: "uc1", Title: "To Delete", Slug: "to-delete-uc1",
		Description: "Desc", Domain: "Sales",
		RiskLevel: "low", ROIPotential: "medium",
		Status: "published", ImplStatus: "evaluating",
		Visibility: "public", AuthorID: "u1",
	}
	st.CreateUseCase(uc)

	if err := st.DeleteUseCase("uc1", "u1"); err != nil {
		t.Fatal(err)
	}

	_, err := st.GetUseCase("uc1")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestDeleteUseCaseWrongAuthor(t *testing.T) {
	st := testStore(t)
	st.UpsertUser("u1", "test@example.com", "Test", "")
	st.UpsertUser("u2", "other@example.com", "Other", "")

	uc := &UseCase{
		ID: "uc1", Title: "Protected", Slug: "protected-uc1",
		Description: "Desc", Domain: "Sales",
		RiskLevel: "low", ROIPotential: "medium",
		Status: "published", ImplStatus: "evaluating",
		Visibility: "public", AuthorID: "u1",
	}
	st.CreateUseCase(uc)

	err := st.DeleteUseCase("uc1", "u2")
	if err == nil {
		t.Fatal("expected error when deleting as wrong author")
	}
}

func TestForkUseCase(t *testing.T) {
	st := testStore(t)
	st.UpsertUser("u1", "author@example.com", "Author", "")
	st.UpsertUser("u2", "forker@example.com", "Forker", "")

	original := &UseCase{
		ID: "uc1", Title: "Original", Slug: "original-uc1",
		Description: "Desc", Domain: "Engineering",
		RiskLevel: "medium", ROIPotential: "high",
		Status: "published", ImplStatus: "production",
		Visibility: "public", AuthorID: "u1",
	}
	st.CreateUseCase(original)

	fork, err := st.ForkUseCase("uc1", "uc2", "fork-uc2", "u2")
	if err != nil {
		t.Fatal(err)
	}
	if fork.AuthorID != "u2" {
		t.Fatalf("expected fork author u2, got %s", fork.AuthorID)
	}
	if fork.ForkedFrom == nil || *fork.ForkedFrom != "uc1" {
		t.Fatal("expected forked_from to be uc1")
	}
	if fork.Visibility != "private" {
		t.Fatalf("expected fork to be private, got %s", fork.Visibility)
	}

	updated, _ := st.GetUseCase("uc1")
	if updated.ForkCount != 1 {
		t.Fatalf("expected fork count 1, got %d", updated.ForkCount)
	}
}

func TestListUserUseCases(t *testing.T) {
	st := testStore(t)
	st.UpsertUser("u1", "test@example.com", "Test", "")
	st.UpsertUser("u2", "other@example.com", "Other", "")

	for _, authorID := range []string{"u1", "u1", "u2"} {
		uc := &UseCase{
			ID: newID(), Title: "UC", Slug: "uc-" + newID()[:8],
			Description: "Desc", Domain: "Sales",
			RiskLevel: "low", ROIPotential: "medium",
			Status: "published", ImplStatus: "evaluating",
			Visibility: "public", AuthorID: authorID,
		}
		st.CreateUseCase(uc)
	}

	list, err := st.ListUserUseCases("u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2, got %d", len(list))
	}
}
