package spectest

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/priyanshujain/openbotkit/agent"
	"github.com/priyanshujain/openbotkit/agent/tools"
	"github.com/priyanshujain/openbotkit/internal/skills"
	"github.com/priyanshujain/openbotkit/internal/testutil"
	"github.com/priyanshujain/openbotkit/memory"
	"github.com/priyanshujain/openbotkit/provider"
	"github.com/priyanshujain/openbotkit/provider/anthropic"
	"github.com/priyanshujain/openbotkit/provider/gemini"
	"github.com/priyanshujain/openbotkit/provider/openai"
	embeddedSkills "github.com/priyanshujain/openbotkit/skills"
	gmail "github.com/priyanshujain/openbotkit/source/gmail"
	"github.com/priyanshujain/openbotkit/source/whatsapp"
	"github.com/priyanshujain/openbotkit/store"
)

type LocalFixture struct {
	dir           string
	Provider      provider.Provider
	Model         string
	JudgeProvider provider.Provider
	JudgeModel    string
}

type providerCase struct {
	Name     string
	Provider provider.Provider
	Model    string
}

// NewLocalFixture creates a fixture using the first available provider.
func NewLocalFixture(t *testing.T) *LocalFixture {
	t.Helper()
	cases := availableProviders(t)
	if len(cases) == 0 {
		t.Skip("no LLM API keys set — skipping spec tests")
	}
	judge := pickJudge(cases)
	fx := newLocalFixtureWith(t, cases[0].Provider, cases[0].Model)
	fx.JudgeProvider = judge.Provider
	fx.JudgeModel = judge.Model
	return fx
}

// EachProvider runs fn as a subtest for every available LLM provider.
// A separate judge provider is chosen for evaluating responses to avoid
// self-evaluation bias (e.g., Gemini judging its own output).
func EachProvider(t *testing.T, fn func(t *testing.T, fx *LocalFixture)) {
	t.Helper()
	cases := availableProviders(t)
	if len(cases) == 0 {
		t.Skip("no LLM API keys set — skipping spec tests")
	}
	judge := pickJudge(cases)
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			fx := newLocalFixtureWith(t, tc.Provider, tc.Model)
			fx.JudgeProvider = judge.Provider
			fx.JudgeModel = judge.Model
			fn(t, fx)
		})
	}
}

func newLocalFixtureWith(t *testing.T, p provider.Provider, model string) *LocalFixture {
	t.Helper()

	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not in PATH")
	}

	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	createSourceDirs(t, dir)
	createGmailDB(t, dir)
	createWhatsAppDB(t, dir)
	createMemoryDB(t, dir)
	installSkills(t, dir)
	generateIndex(t)
	buildBinary(t, dir)

	t.Setenv("PATH", filepath.Join(dir, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))

	return &LocalFixture{
		dir:      dir,
		Provider: p,
		Model:    model,
	}
}

// pickJudge selects the best provider for LLM-as-judge evaluation.
// Prefers anthropic-vertex (Claude) > openai > gemini for reliability.
func pickJudge(cases []providerCase) providerCase {
	priority := map[string]int{"anthropic-vertex": 0, "openai": 1, "gemini": 2}
	best := cases[0]
	for _, c := range cases[1:] {
		if priority[c.Name] < priority[best.Name] {
			best = c
		}
	}
	return best
}

func availableProviders(t *testing.T) []providerCase {
	t.Helper()
	testutil.LoadEnv(t)
	var cases []providerCase

	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		cases = append(cases, providerCase{
			Name:     "gemini",
			Provider: gemini.New(key),
			Model:    "gemini-2.5-flash",
		})
	}
	if project := os.Getenv("GOOGLE_CLOUD_PROJECT"); project != "" {
		region := os.Getenv("GOOGLE_CLOUD_REGION")
		if region == "" {
			region = "us-east5"
		}
		account := os.Getenv("GOOGLE_CLOUD_ACCOUNT")
		model := os.Getenv("VERTEX_CLAUDE_MODEL")
		if model == "" {
			model = "claude-sonnet-4@20250514"
		}
		cases = append(cases, providerCase{
			Name:     "anthropic-vertex",
			Provider: anthropic.New("", anthropic.WithVertexAI(project, region), anthropic.WithTokenSource(provider.GcloudTokenSource(account))),
			Model:    model,
		})
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		cases = append(cases, providerCase{
			Name:     "openai",
			Provider: openai.New(key),
			Model:    "gpt-4.1-mini",
		})
	}

	return cases
}

func createSourceDirs(t *testing.T, dir string) {
	t.Helper()
	for _, src := range []string{"gmail", "whatsapp", "history", "user_memory", "applenotes"} {
		if err := os.MkdirAll(filepath.Join(dir, src), 0700); err != nil {
			t.Fatalf("mkdir %s: %v", src, err)
		}
	}
}

func createGmailDB(t *testing.T, dir string) {
	t.Helper()
	dbPath := filepath.Join(dir, "gmail", "data.db")
	db, err := store.Open(store.SQLiteConfig(dbPath))
	if err != nil {
		t.Fatalf("open gmail db: %v", err)
	}
	defer db.Close()
	if err := gmail.Migrate(db); err != nil {
		t.Fatalf("migrate gmail: %v", err)
	}
}

func createWhatsAppDB(t *testing.T, dir string) {
	t.Helper()
	dbPath := filepath.Join(dir, "whatsapp", "data.db")
	db, err := store.Open(store.SQLiteConfig(dbPath))
	if err != nil {
		t.Fatalf("open whatsapp db: %v", err)
	}
	defer db.Close()
	if err := whatsapp.Migrate(db); err != nil {
		t.Fatalf("migrate whatsapp: %v", err)
	}
}

func createMemoryDB(t *testing.T, dir string) {
	t.Helper()
	dbPath := filepath.Join(dir, "user_memory", "data.db")
	db, err := store.Open(store.SQLiteConfig(dbPath))
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}
	defer db.Close()
	if err := memory.Migrate(db); err != nil {
		t.Fatalf("migrate memory: %v", err)
	}
}

func installSkills(t *testing.T, dir string) {
	t.Helper()
	skillsDir := filepath.Join(dir, "skills")
	for _, name := range []string{"email-read", "whatsapp-read", "memory-read", "memory-save"} {
		destDir := filepath.Join(skillsDir, name)
		if err := os.MkdirAll(destDir, 0700); err != nil {
			t.Fatalf("mkdir skill %s: %v", name, err)
		}
		entries, err := fs.ReadDir(embeddedSkills.FS, name)
		if err != nil {
			t.Fatalf("read embedded skill %s: %v", name, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, err := fs.ReadFile(embeddedSkills.FS, name+"/"+entry.Name())
			if err != nil {
				t.Fatalf("read embedded file %s/%s: %v", name, entry.Name(), err)
			}
			if err := os.WriteFile(filepath.Join(destDir, entry.Name()), content, 0600); err != nil {
				t.Fatalf("write file %s/%s: %v", name, entry.Name(), err)
			}
		}
	}
}

func generateIndex(t *testing.T) {
	t.Helper()
	if err := skills.GenerateIndex(); err != nil {
		t.Fatalf("generate skill index: %v", err)
	}
}

func buildBinary(t *testing.T, dir string) {
	t.Helper()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0700); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}

	// Find project root (where go.mod lives).
	root, err := findProjectRoot()
	if err != nil {
		t.Fatalf("find project root: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", filepath.Join(binDir, "obk"), ".")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build: %v", err)
	}
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func (f *LocalFixture) Agent(t *testing.T) *agent.Agent {
	t.Helper()

	toolReg := tools.NewRegistry()
	toolReg.Register(tools.NewBashTool(30 * time.Second))
	toolReg.Register(&tools.FileReadTool{})
	toolReg.Register(&tools.LoadSkillsTool{})
	toolReg.Register(&tools.SearchSkillsTool{})

	identity := "You are a personal AI assistant powered by OpenBotKit.\n"
	blocks := tools.BuildSystemBlocks(identity, toolReg)

	return agent.New(f.Provider, f.Model, toolReg,
		agent.WithSystemBlocks(blocks),
		agent.WithMaxIterations(15),
	)
}

func (f *LocalFixture) GivenEmails(t *testing.T, emails []Email) {
	t.Helper()

	dbPath := filepath.Join(f.dir, "gmail", "data.db")
	db, err := store.Open(store.SQLiteConfig(dbPath))
	if err != nil {
		t.Fatalf("open gmail db: %v", err)
	}
	defer db.Close()

	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	for i, e := range emails {
		if e.MessageID == "" {
			e.MessageID = uuid.New().String()
		}
		if e.Account == "" {
			e.Account = "test@example.com"
		}

		gmailEmail := &gmail.Email{
			MessageID: e.MessageID,
			Account:   e.Account,
			From:      e.From,
			To:        e.To,
			Subject:   e.Subject,
			Body:      e.Body,
			Date:      baseTime.Add(time.Duration(i) * time.Hour),
		}
		if _, err := gmail.SaveEmail(db, gmailEmail); err != nil {
			t.Fatalf("save email %d: %v", i, err)
		}
	}
}

func (f *LocalFixture) GivenWhatsAppMessages(t *testing.T, messages []WhatsAppMessage) {
	t.Helper()

	dbPath := filepath.Join(f.dir, "whatsapp", "data.db")
	db, err := store.Open(store.SQLiteConfig(dbPath))
	if err != nil {
		t.Fatalf("open whatsapp db: %v", err)
	}
	defer db.Close()

	// Track chats so we upsert each once.
	seenChats := make(map[string]bool)
	baseTime := time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC)

	for i, m := range messages {
		if m.MessageID == "" {
			m.MessageID = uuid.New().String()
		}
		if m.ChatJID == "" {
			m.ChatJID = m.SenderJID
		}

		if !seenChats[m.ChatJID] {
			if err := whatsapp.UpsertChat(db, m.ChatJID, m.ChatName, false); err != nil {
				t.Fatalf("upsert chat %s: %v", m.ChatJID, err)
			}
			seenChats[m.ChatJID] = true
		}

		msg := &whatsapp.Message{
			MessageID:  m.MessageID,
			ChatJID:    m.ChatJID,
			SenderJID:  m.SenderJID,
			SenderName: m.SenderName,
			Text:       m.Text,
			Timestamp:  baseTime.Add(time.Duration(i) * 10 * time.Minute),
			IsFromMe:   m.IsFromMe,
		}
		if err := whatsapp.SaveMessage(db, msg); err != nil {
			t.Fatalf("save whatsapp message %d: %v", i, err)
		}
	}
}

func (f *LocalFixture) GivenMemories(t *testing.T, memories []UserMemory) {
	t.Helper()

	dbPath := filepath.Join(f.dir, "user_memory", "data.db")
	db, err := store.Open(store.SQLiteConfig(dbPath))
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}
	defer db.Close()

	for i, m := range memories {
		cat := m.Category
		if cat == "" {
			cat = "preference"
		}
		if _, err := memory.Add(db, m.Content, memory.Category(cat), "manual", ""); err != nil {
			t.Fatalf("add memory %d: %v", i, err)
		}
	}
}
