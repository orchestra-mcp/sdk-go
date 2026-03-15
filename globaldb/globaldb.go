package globaldb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	globalDB   *sql.DB
	globalOnce sync.Once
	globalErr  error
)

const globalSchema = `
CREATE TABLE IF NOT EXISTS current_user (
    project_slug TEXT PRIMARY KEY,
    person_id TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS accounts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    provider TEXT DEFAULT 'claude',
    auth_method TEXT NOT NULL,
    config TEXT DEFAULT '{}',
    default_model TEXT DEFAULT '',
    max_budget_usd REAL DEFAULT 0,
    alert_at_pct REAL DEFAULT 80,
    used_budget_usd REAL DEFAULT 0,
    total_tokens_in INTEGER DEFAULT 0,
    total_tokens_out INTEGER DEFAULT 0,
    total_sessions INTEGER DEFAULT 0,
    status TEXT DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS workspaces (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    folders TEXT DEFAULT '[]',
    primary_folder TEXT DEFAULT '',
    metadata TEXT DEFAULT '{}',
    last_used TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS session_locks (
    feature_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    locked_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (project_id, feature_id)
);
CREATE INDEX IF NOT EXISTS idx_session_locks_session ON session_locks(session_id);

CREATE TABLE IF NOT EXISTS secrets (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT DEFAULT 'general',
    value TEXT NOT NULL DEFAULT '',
    description TEXT DEFAULT '',
    tags TEXT DEFAULT '[]',
    scope TEXT DEFAULT 'global',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_secrets_category ON secrets(category);
CREATE INDEX IF NOT EXISTS idx_secrets_scope ON secrets(scope);
`

// DB returns the global database singleton at ~/.orchestra/db/global.db.
// Safe for concurrent use — initialized once with proper pragmas.
func DB() (*sql.DB, error) {
	globalOnce.Do(func() {
		dbPath := globalDBPath()
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			globalErr = fmt.Errorf("create global db dir: %w", err)
			return
		}

		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			globalErr = fmt.Errorf("open global db: %w", err)
			return
		}

		pragmas := []string{
			"PRAGMA journal_mode=WAL",
			"PRAGMA busy_timeout=5000",
			"PRAGMA foreign_keys=ON",
			"PRAGMA synchronous=NORMAL",
		}
		for _, p := range pragmas {
			if _, err := db.Exec(p); err != nil {
				db.Close()
				globalErr = fmt.Errorf("set pragma: %w", err)
				return
			}
		}

		db.SetMaxOpenConns(1)

		if _, err := db.Exec(globalSchema); err != nil {
			db.Close()
			globalErr = fmt.Errorf("init global schema: %w", err)
			return
		}

		globalDB = db
	})
	return globalDB, globalErr
}

// Close closes the global database connection.
func Close() {
	if globalDB != nil {
		globalDB.Close()
		globalDB = nil
		globalOnce = sync.Once{} // Allow re-initialization.
	}
}

func globalDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orchestra", "db", "global.db")
}

// --- Current User helpers ---

// GetCurrentUser returns the person_id for the given project, or empty string.
func GetCurrentUser(projectSlug string) string {
	db, err := DB()
	if err != nil {
		return ""
	}
	var personID string
	err = db.QueryRow(`SELECT person_id FROM current_user WHERE project_slug = ?`, projectSlug).Scan(&personID)
	if err != nil {
		return ""
	}
	return personID
}

// SetCurrentUser links a person to a project.
func SetCurrentUser(projectSlug, personID string) error {
	db, err := DB()
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO current_user (project_slug, person_id)
		VALUES (?, ?) ON CONFLICT(project_slug) DO UPDATE SET person_id = excluded.person_id`,
		projectSlug, personID)
	return err
}

// GetDefaultProject returns the default project slug, or empty string.
func GetDefaultProject() string {
	return GetConfig("default_project")
}

// SetDefaultProject sets the default project slug.
func SetDefaultProject(slug string) error {
	return SetConfig("default_project", slug)
}

// --- Config helpers ---

// GetConfig returns the value for a config key, or empty string.
func GetConfig(key string) string {
	db, err := DB()
	if err != nil {
		return ""
	}
	var value string
	db.QueryRow(`SELECT value FROM config WHERE key = ?`, key).Scan(&value)
	return value
}

// SetConfig stores a config key-value pair.
func SetConfig(key, value string) error {
	db, err := DB()
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO config (key, value)
		VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value)
	return err
}

// --- Account helpers ---

// Account represents an AI provider account with budget tracking.
type Account struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Provider       string            `json:"provider"`
	AuthMethod     string            `json:"auth_method"`
	Config         map[string]string `json:"config"`
	DefaultModel   string            `json:"default_model"`
	MaxBudgetUSD   float64           `json:"max_budget_usd"`
	AlertAtPct     float64           `json:"alert_at_pct"`
	UsedBudgetUSD  float64           `json:"used_budget_usd"`
	TotalTokensIn  int64             `json:"total_tokens_in"`
	TotalTokensOut int64             `json:"total_tokens_out"`
	TotalSessions  int               `json:"total_sessions"`
	Status         string            `json:"status"`
	CreatedAt      string            `json:"created_at"`
}

// CreateAccount inserts a new account.
func CreateAccount(acct *Account) error {
	db, err := DB()
	if err != nil {
		return err
	}
	if acct.Provider == "" {
		acct.Provider = "claude"
	}
	if acct.CreatedAt == "" {
		acct.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if acct.Status == "" {
		acct.Status = "active"
	}
	if acct.AlertAtPct == 0 {
		acct.AlertAtPct = 80
	}
	if acct.Config == nil {
		acct.Config = make(map[string]string)
	}
	cfgEncrypted, err := encryptConfig(acct.Config)
	if err != nil {
		return fmt.Errorf("encrypt config: %w", err)
	}
	_, err = db.Exec(`INSERT INTO accounts (id, name, provider, auth_method, config, default_model,
		max_budget_usd, alert_at_pct, used_budget_usd, total_tokens_in, total_tokens_out,
		total_sessions, status, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		acct.ID, acct.Name, acct.Provider, acct.AuthMethod, cfgEncrypted, acct.DefaultModel,
		acct.MaxBudgetUSD, acct.AlertAtPct, acct.UsedBudgetUSD, acct.TotalTokensIn,
		acct.TotalTokensOut, acct.TotalSessions, acct.Status, acct.CreatedAt)
	return err
}

// GetAccount returns a single account by ID.
func GetAccount(id string) (*Account, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	var acct Account
	var cfgJSON string
	err = db.QueryRow(`SELECT id, name, provider, auth_method, config, default_model,
		max_budget_usd, alert_at_pct, used_budget_usd, total_tokens_in, total_tokens_out,
		total_sessions, status, created_at FROM accounts WHERE id = ?`, id).Scan(
		&acct.ID, &acct.Name, &acct.Provider, &acct.AuthMethod, &cfgJSON, &acct.DefaultModel,
		&acct.MaxBudgetUSD, &acct.AlertAtPct, &acct.UsedBudgetUSD, &acct.TotalTokensIn,
		&acct.TotalTokensOut, &acct.TotalSessions, &acct.Status, &acct.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("account %q not found", id)
	}
	acct.Config, err = decryptConfig(cfgJSON)
	if err != nil {
		return nil, fmt.Errorf("decrypt config: %w", err)
	}
	return &acct, nil
}

// ListAccounts returns all accounts.
func ListAccounts() ([]*Account, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT id, name, provider, auth_method, config, default_model,
		max_budget_usd, alert_at_pct, used_budget_usd, total_tokens_in, total_tokens_out,
		total_sessions, status, created_at FROM accounts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*Account
	for rows.Next() {
		var acct Account
		var cfgJSON string
		if err := rows.Scan(&acct.ID, &acct.Name, &acct.Provider, &acct.AuthMethod, &cfgJSON,
			&acct.DefaultModel, &acct.MaxBudgetUSD, &acct.AlertAtPct, &acct.UsedBudgetUSD,
			&acct.TotalTokensIn, &acct.TotalTokensOut, &acct.TotalSessions, &acct.Status,
			&acct.CreatedAt); err != nil {
			continue
		}
		acct.Config, _ = decryptConfig(cfgJSON)
		result = append(result, &acct)
	}
	return result, nil
}

// UpdateAccount modifies an existing account via a mutation function.
func UpdateAccount(id string, fn func(acct *Account)) error {
	acct, err := GetAccount(id)
	if err != nil {
		return err
	}
	fn(acct)
	return SaveAccount(acct)
}

// SaveAccount writes the full account back to the database.
func SaveAccount(acct *Account) error {
	db, err := DB()
	if err != nil {
		return err
	}
	cfgEncrypted, err := encryptConfig(acct.Config)
	if err != nil {
		return fmt.Errorf("encrypt config: %w", err)
	}
	_, err = db.Exec(`UPDATE accounts SET name=?, provider=?, auth_method=?, config=?,
		default_model=?, max_budget_usd=?, alert_at_pct=?, used_budget_usd=?,
		total_tokens_in=?, total_tokens_out=?, total_sessions=?, status=? WHERE id=?`,
		acct.Name, acct.Provider, acct.AuthMethod, cfgEncrypted, acct.DefaultModel,
		acct.MaxBudgetUSD, acct.AlertAtPct, acct.UsedBudgetUSD, acct.TotalTokensIn,
		acct.TotalTokensOut, acct.TotalSessions, acct.Status, acct.ID)
	return err
}

// DeleteAccount removes an account by ID.
func DeleteAccount(id string) error {
	db, err := DB()
	if err != nil {
		return err
	}
	res, err := db.Exec(`DELETE FROM accounts WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("account %q not found", id)
	}
	return nil
}

// --- Workspace helpers ---

// Workspace represents a named collection of project folders.
type Workspace struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Folders       []string          `json:"folders"`
	PrimaryFolder string            `json:"primary_folder"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	LastUsed      string            `json:"last_used"`
	CreatedAt     string            `json:"created_at"`
}

// CreateWorkspace inserts a new workspace.
func CreateWorkspace(ws *Workspace) error {
	db, err := DB()
	if err != nil {
		return err
	}
	if ws.CreatedAt == "" {
		ws.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if ws.Metadata == nil {
		ws.Metadata = make(map[string]string)
	}
	foldersJSON, _ := json.Marshal(ws.Folders)
	metaJSON, _ := json.Marshal(ws.Metadata)
	_, err = db.Exec(`INSERT INTO workspaces (id, name, folders, primary_folder, metadata, last_used, created_at)
		VALUES (?,?,?,?,?,?,?)`,
		ws.ID, ws.Name, string(foldersJSON), ws.PrimaryFolder, string(metaJSON), ws.LastUsed, ws.CreatedAt)
	return err
}

// GetWorkspace returns a single workspace by ID.
func GetWorkspace(id string) (*Workspace, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	var ws Workspace
	var foldersJSON, metaJSON string
	err = db.QueryRow(`SELECT id, name, folders, primary_folder, metadata, last_used, created_at
		FROM workspaces WHERE id = ?`, id).Scan(
		&ws.ID, &ws.Name, &foldersJSON, &ws.PrimaryFolder, &metaJSON, &ws.LastUsed, &ws.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("workspace %q not found", id)
	}
	json.Unmarshal([]byte(foldersJSON), &ws.Folders)
	ws.Metadata = make(map[string]string)
	json.Unmarshal([]byte(metaJSON), &ws.Metadata)
	return &ws, nil
}

// ListWorkspaces returns all workspaces.
func ListWorkspaces() ([]*Workspace, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT id, name, folders, primary_folder, metadata, last_used, created_at FROM workspaces`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*Workspace
	for rows.Next() {
		var ws Workspace
		var foldersJSON, metaJSON string
		if err := rows.Scan(&ws.ID, &ws.Name, &foldersJSON, &ws.PrimaryFolder, &metaJSON,
			&ws.LastUsed, &ws.CreatedAt); err != nil {
			continue
		}
		json.Unmarshal([]byte(foldersJSON), &ws.Folders)
		ws.Metadata = make(map[string]string)
		json.Unmarshal([]byte(metaJSON), &ws.Metadata)
		result = append(result, &ws)
	}
	return result, nil
}

// SaveWorkspace writes the full workspace back to the database.
func SaveWorkspace(ws *Workspace) error {
	db, err := DB()
	if err != nil {
		return err
	}
	foldersJSON, _ := json.Marshal(ws.Folders)
	metaJSON, _ := json.Marshal(ws.Metadata)
	_, err = db.Exec(`UPDATE workspaces SET name=?, folders=?, primary_folder=?, metadata=?, last_used=?
		WHERE id=?`, ws.Name, string(foldersJSON), ws.PrimaryFolder, string(metaJSON), ws.LastUsed, ws.ID)
	return err
}

// DeleteWorkspace removes a workspace by ID.
func DeleteWorkspace(id string) error {
	db, err := DB()
	if err != nil {
		return err
	}
	res, err := db.Exec(`DELETE FROM workspaces WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("workspace %q not found", id)
	}
	return nil
}

// GetActiveWorkspaceID returns the active workspace ID from config.
func GetActiveWorkspaceID() string {
	return GetConfig("active_workspace_id")
}

// SetActiveWorkspaceID sets the active workspace ID in config.
func SetActiveWorkspaceID(id string) error {
	return SetConfig("active_workspace_id", id)
}

// MigrateAccountsJSON imports accounts from ~/.orchestra/agentops/accounts.json if globaldb is empty.
func MigrateAccountsJSON() {
	db, err := DB()
	if err != nil {
		return
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM accounts`).Scan(&count)
	if count > 0 {
		return
	}
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".orchestra", "agentops", "accounts.json"))
	if err != nil {
		return
	}
	var accounts map[string]*Account
	if json.Unmarshal(data, &accounts) != nil {
		return
	}
	for _, acct := range accounts {
		CreateAccount(acct)
	}
}

// MigrateWorkspacesJSON imports workspaces from ~/.orchestra/workspaces.json if globaldb is empty.
func MigrateWorkspacesJSON() {
	db, err := DB()
	if err != nil {
		return
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM workspaces`).Scan(&count)
	if count > 0 {
		return
	}
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".orchestra", "workspaces.json"))
	if err != nil {
		return
	}
	type jsonRegistry struct {
		Workspaces        []*Workspace `json:"workspaces"`
		ActiveWorkspaceID string       `json:"active_workspace_id"`
	}
	var reg jsonRegistry
	if json.Unmarshal(data, &reg) != nil {
		return
	}
	for _, ws := range reg.Workspaces {
		if ws.Folders == nil {
			ws.Folders = []string{}
		}
		if ws.Metadata == nil {
			ws.Metadata = make(map[string]string)
		}
		CreateWorkspace(ws)
	}
	if reg.ActiveWorkspaceID != "" {
		SetActiveWorkspaceID(reg.ActiveWorkspaceID)
	}
}

// --- Secret helpers ---

// Secret represents an encrypted key-value secret stored in globaldb.
type Secret struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Value       string   `json:"value"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Scope       string   `json:"scope"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// CreateSecret inserts a new secret with its value encrypted.
func CreateSecret(s *Secret) error {
	db, err := DB()
	if err != nil {
		return err
	}
	if s.CreatedAt == "" {
		now := time.Now().UTC().Format(time.RFC3339)
		s.CreatedAt = now
		s.UpdatedAt = now
	}
	if s.Category == "" {
		s.Category = "general"
	}
	if s.Scope == "" {
		s.Scope = "global"
	}
	if s.Tags == nil {
		s.Tags = []string{}
	}
	encValue, err := encryptString(s.Value)
	if err != nil {
		return fmt.Errorf("encrypt secret: %w", err)
	}
	tagsJSON, _ := json.Marshal(s.Tags)
	_, err = db.Exec(`INSERT INTO secrets (id, name, category, value, description, tags, scope, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		s.ID, s.Name, s.Category, encValue, s.Description, string(tagsJSON), s.Scope, s.CreatedAt, s.UpdatedAt)
	return err
}

// GetSecret returns a single secret by ID with its value decrypted.
func GetSecret(id string) (*Secret, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	var s Secret
	var encValue, tagsJSON string
	err = db.QueryRow(`SELECT id, name, category, value, description, tags, scope, created_at, updated_at
		FROM secrets WHERE id = ?`, id).Scan(
		&s.ID, &s.Name, &s.Category, &encValue, &s.Description, &tagsJSON, &s.Scope, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("secret %q not found", id)
	}
	s.Value, err = decryptString(encValue)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret: %w", err)
	}
	s.Tags = []string{}
	json.Unmarshal([]byte(tagsJSON), &s.Tags)
	return &s, nil
}

// ListSecrets returns all secrets (values are NOT decrypted — masked).
func ListSecrets() ([]*Secret, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT id, name, category, description, tags, scope, created_at, updated_at FROM secrets`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*Secret
	for rows.Next() {
		var s Secret
		var tagsJSON string
		if err := rows.Scan(&s.ID, &s.Name, &s.Category, &s.Description, &tagsJSON, &s.Scope, &s.CreatedAt, &s.UpdatedAt); err != nil {
			continue
		}
		s.Tags = []string{}
		json.Unmarshal([]byte(tagsJSON), &s.Tags)
		s.Value = "****" // Never return plaintext in list
		result = append(result, &s)
	}
	return result, nil
}

// ListSecretsByCategory returns secrets filtered by category.
func ListSecretsByCategory(category string) ([]*Secret, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT id, name, category, description, tags, scope, created_at, updated_at
		FROM secrets WHERE category = ?`, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*Secret
	for rows.Next() {
		var s Secret
		var tagsJSON string
		if err := rows.Scan(&s.ID, &s.Name, &s.Category, &s.Description, &tagsJSON, &s.Scope, &s.CreatedAt, &s.UpdatedAt); err != nil {
			continue
		}
		s.Tags = []string{}
		json.Unmarshal([]byte(tagsJSON), &s.Tags)
		s.Value = "****"
		result = append(result, &s)
	}
	return result, nil
}

// UpdateSecret modifies an existing secret via a mutation function.
func UpdateSecret(id string, fn func(s *Secret)) error {
	s, err := GetSecret(id)
	if err != nil {
		return err
	}
	fn(s)
	return SaveSecret(s)
}

// SaveSecret writes the full secret back to the database.
func SaveSecret(s *Secret) error {
	db, err := DB()
	if err != nil {
		return err
	}
	s.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	encValue, err := encryptString(s.Value)
	if err != nil {
		return fmt.Errorf("encrypt secret: %w", err)
	}
	tagsJSON, _ := json.Marshal(s.Tags)
	_, err = db.Exec(`UPDATE secrets SET name=?, category=?, value=?, description=?, tags=?, scope=?, updated_at=?
		WHERE id=?`, s.Name, s.Category, encValue, s.Description, string(tagsJSON), s.Scope, s.UpdatedAt, s.ID)
	return err
}

// DeleteSecret removes a secret by ID.
func DeleteSecret(id string) error {
	db, err := DB()
	if err != nil {
		return err
	}
	res, err := db.Exec(`DELETE FROM secrets WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("secret %q not found", id)
	}
	return nil
}

// SearchSecrets searches secrets by name or description substring.
func SearchSecrets(query string) ([]*Secret, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	like := "%" + query + "%"
	rows, err := db.Query(`SELECT id, name, category, description, tags, scope, created_at, updated_at
		FROM secrets WHERE name LIKE ? OR description LIKE ? OR tags LIKE ?`, like, like, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*Secret
	for rows.Next() {
		var s Secret
		var tagsJSON string
		if err := rows.Scan(&s.ID, &s.Name, &s.Category, &s.Description, &tagsJSON, &s.Scope, &s.CreatedAt, &s.UpdatedAt); err != nil {
			continue
		}
		s.Tags = []string{}
		json.Unmarshal([]byte(tagsJSON), &s.Tags)
		s.Value = "****"
		result = append(result, &s)
	}
	return result, nil
}

// GetSecretEnv returns secrets in a given scope as key=value pairs (for .env export).
func GetSecretEnv(scope string) (map[string]string, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	query := `SELECT name, value FROM secrets`
	args := []any{}
	if scope != "" && scope != "all" {
		query += ` WHERE scope = ?`
		args = append(args, scope)
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	env := make(map[string]string)
	for rows.Next() {
		var name, encValue string
		if rows.Scan(&name, &encValue) != nil {
			continue
		}
		val, err := decryptString(encValue)
		if err != nil {
			continue
		}
		env[name] = val
	}
	return env, nil
}

// MigrateEnvFile imports secrets from a .env file into globaldb.
func MigrateEnvFile(data []byte, category, scope string) (int, error) {
	if category == "" {
		category = "env"
	}
	if scope == "" {
		scope = "global"
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Strip surrounding quotes.
		val = strings.Trim(val, `"'`)
		id := "SEC-" + randomID(4)
		s := &Secret{
			ID:       id,
			Name:     key,
			Category: category,
			Value:    val,
			Scope:    scope,
			Tags:     []string{"imported"},
		}
		if err := CreateSecret(s); err != nil {
			continue
		}
		count++
	}
	return count, nil
}

// randomID generates n random uppercase ASCII letters.
func randomID(n int) string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}

// MigrateMeJSON imports me.json current user mappings into globaldb if empty.
func MigrateMeJSON() {
	db, err := DB()
	if err != nil {
		return
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM current_user`).Scan(&count)
	if count > 0 {
		return
	}
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".orchestra", "me.json"))
	if err != nil {
		return
	}
	type meMapping struct {
		PersonID string `json:"person_id"`
	}
	type meConfig struct {
		DefaultProject string                `json:"default_project"`
		Projects       map[string]meMapping  `json:"projects"`
	}
	var cfg meConfig
	if json.Unmarshal(data, &cfg) != nil {
		return
	}
	for slug, m := range cfg.Projects {
		SetCurrentUser(slug, m.PersonID)
	}
	if cfg.DefaultProject != "" {
		SetDefaultProject(cfg.DefaultProject)
	}
}
