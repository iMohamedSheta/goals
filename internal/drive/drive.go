// Package drive implements Google Drive backup/restore for the Goals database.
//
// Sign-in uses imohamedsheta/xsocial's Google Device Authorization Grant:
// the app shows a short code, the user approves it at google.com/device, and
// the app polls for tokens. No localhost server, no firewall issues. The token
// (with offline refresh) is stored in the local settings table.
package drive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"goals/internal/store"

	"github.com/imohamedsheta/xsocial"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	keyClientID     = "data.google_client_id"
	keyClientSecret = "data.google_client_secret"
	keyToken        = "data.google_token"
	keyEmail        = "data.google_email"

	backupFolder = "Goals Backups"
)

// Baked-in OAuth client (Desktop-app type) so connecting is one click and no
// console visit is needed. Google requires *some* client ID to identify the app;
// these are baked at build time. Users can still override via settings (advanced).
// NOTE: fill these in before building a distributable exe.
const (
	defaultClientID     = ""
	defaultClientSecret = ""
)

var (
	flowMu       sync.Mutex
	pendingCode  string
	pendingFlow  *xsocial.GoogleDeviceFlow
	pendingSince time.Time
)

func newDeviceFlow(id, secret string) *xsocial.GoogleDeviceFlow {
	return xsocial.NewGoogleDeviceFlow(id, secret, []string{drive.DriveFileScope, "email"})
}

type Status struct {
	HasClient bool   `json:"hasClient"`
	Embedded  bool   `json:"embedded"`
	Connected bool   `json:"connected"`
	Email     string `json:"email"`
}

type BackupFile struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Created string `json:"createdTime"`
	Size    int64  `json:"size"`
}

func settings(s *store.Store) (map[string]string, error) {
	m, err := s.GetSettings()
	if err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]string{}
	}
	return m, nil
}

func oauthConfig(clientID, secret, redirect string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: secret,
		RedirectURL:  redirect,
		Scopes:       []string{drive.DriveFileScope},
		Endpoint:     google.Endpoint,
	}
}

// effectiveCredentials prefers a manual override, else the baked-in client.
func effectiveCredentials(m map[string]string) (id, secret string) {
	if m[keyClientID] != "" {
		return m[keyClientID], m[keyClientSecret]
	}
	return defaultClientID, defaultClientSecret
}

// HasEmbeddedClient reports whether one-click connect is available.
func HasEmbeddedClient() bool { return defaultClientID != "" }

func loadToken(s *store.Store) (*oauth2.Token, error) {
	m, err := settings(s)
	if err != nil {
		return nil, err
	}
	raw := m[keyToken]
	if raw == "" {
		return nil, fmt.Errorf("not connected")
	}
	var tok oauth2.Token
	if err := json.Unmarshal([]byte(raw), &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

func clientFor(s *store.Store) (*http.Client, error) {
	m, err := settings(s)
	if err != nil {
		return nil, err
	}
	id, secret := effectiveCredentials(m)
	if id == "" {
		return nil, fmt.Errorf("google client not configured")
	}
	tok, err := loadToken(s)
	if err != nil {
		return nil, err
	}
	cfg := oauthConfig(id, secret, "")
	return cfg.Client(context.Background(), tok), nil
}

func driveService(s *store.Store) (*drive.Service, error) {
	cl, err := clientFor(s)
	if err != nil {
		return nil, err
	}
	return drive.NewService(context.Background(), option.WithHTTPClient(cl))
}

// GetStatus reports credentials + connection state.
func GetStatus(s *store.Store) (Status, error) {
	m, err := settings(s)
	if err != nil {
		return Status{}, err
	}
	st := Status{}
	id, _ := effectiveCredentials(m)
	st.HasClient = id != ""
	st.Embedded = defaultClientID != "" && m[keyClientID] == ""
	if m[keyToken] == "" {
		return st, nil
	}
	var tok oauth2.Token
	if err := json.Unmarshal([]byte(m[keyToken]), &tok); err != nil {
		return st, nil
	}
	st.Connected = true
	st.Email = m[keyEmail]
	return st, nil
}

// SaveCredentials stores the user's own OAuth client id/secret.
func SaveCredentials(s *store.Store, id, secret string) error {
	if err := s.SetSetting(keyClientID, id); err != nil {
		return err
	}
	return s.SetSetting(keyClientSecret, secret)
}

// Disconnect forgets the token (backups on Drive stay untouched).
func Disconnect(s *store.Store) error {
	if err := s.DeleteSetting(keyToken); err != nil {
		return err
	}
	return s.DeleteSetting(keyEmail)
}

// DeviceAuth is what the UI shows: a short code the user approves at Google.
type DeviceAuth struct {
	UserCode        string `json:"userCode"`
	VerificationURL string `json:"verificationUrl"`
	ExpiresIn       int    `json:"expiresIn"`
	Interval        int    `json:"interval"`
}

// DevicePoll is one poll attempt's outcome.
type DevicePoll struct {
	Status string `json:"status"` // pending|success|expired|denied
	Email  string `json:"email"`
}

// StartDeviceAuth begins an xsocial device flow and returns the user code.
// The frontend displays it, opens the verification URL, then polls PollDeviceAuth.
func StartDeviceAuth(s *store.Store) (DeviceAuth, error) {
	m, err := settings(s)
	if err != nil {
		return DeviceAuth{}, err
	}
	id, secret := effectiveCredentials(m)
	if id == "" {
		return DeviceAuth{}, fmt.Errorf("google sign-in is not configured in this build")
	}
	flowMu.Lock()
	defer flowMu.Unlock()
	flow := newDeviceFlow(id, secret)
	resp, err := flow.RequestDeviceCode()
	if err != nil {
		return DeviceAuth{}, err
	}
	pendingFlow = flow
	pendingCode = resp.DeviceCode
	pendingSince = time.Now()
	interval := resp.Interval
	if interval <= 0 {
		interval = 5
	}
	return DeviceAuth{
		UserCode:        resp.UserCode,
		VerificationURL: resp.VerificationURL,
		ExpiresIn:       resp.ExpiresIn,
		Interval:        interval,
	}, nil
}

// PollDeviceAuth checks once whether the user approved the pending code.
func PollDeviceAuth(s *store.Store) (DevicePoll, error) {
	flowMu.Lock()
	flow, code := pendingFlow, pendingCode
	flowMu.Unlock()
	if flow == nil || code == "" {
		return DevicePoll{Status: xsocial.DeviceFlowExpired}, nil
	}
	tok, status, err := flow.PollForToken(code)
	if err != nil {
		return DevicePoll{}, err
	}
	switch status {
	case xsocial.DeviceFlowSuccess:
		stored := &oauth2.Token{
			AccessToken:  tok.AccessToken,
			RefreshToken: tok.RefreshToken,
			TokenType:    tok.TokenType,
			Expiry:       time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second),
		}
		raw, _ := json.Marshal(stored)
		if err := s.SetSetting(keyToken, string(raw)); err != nil {
			return DevicePoll{}, err
		}
		email := ""
		if user, err := flow.GetUserInfo(tok.AccessToken); err == nil {
			if e, ok := user["email"].(string); ok {
				email = e
				_ = s.SetSetting(keyEmail, email)
			}
		}
		CancelAuthFlow()
		return DevicePoll{Status: xsocial.DeviceFlowSuccess, Email: email}, nil
	case xsocial.DeviceFlowExpired, xsocial.DeviceFlowDenied:
		CancelAuthFlow()
		return DevicePoll{Status: status}, nil
	default:
		return DevicePoll{Status: xsocial.DeviceFlowPending}, nil
	}
}

// CancelAuthFlow abandons a pending device-code sign-in.
func CancelAuthFlow() {
	flowMu.Lock()
	defer flowMu.Unlock()
	pendingFlow = nil
	pendingCode = ""
}

func ensureFolder(svc *drive.Service) (string, error) {
	list, err := svc.Files.List().
		Q(fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder' and trashed=false", backupFolder)).
		Fields("files(id)").Do()
	if err != nil {
		return "", err
	}
	if len(list.Files) > 0 {
		return list.Files[0].Id, nil
	}
	f, err := svc.Files.Create(&drive.File{
		Name:     backupFolder,
		MimeType: "application/vnd.google-apps.folder",
	}).Fields("id").Do()
	if err != nil {
		return "", err
	}
	return f.Id, nil
}

// BackupNow snapshots the live DB (VACUUM INTO, no close needed) and uploads it.
func BackupNow(s *store.Store) (BackupFile, error) {
	svc, err := driveService(s)
	if err != nil {
		return BackupFile{}, err
	}
	folderID, err := ensureFolder(svc)
	if err != nil {
		return BackupFile{}, err
	}
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("goals-backup-%d.db", time.Now().Unix()))
	defer os.Remove(tmp)
	if err := s.VacuumInto(tmp); err != nil {
		return BackupFile{}, err
	}
	f, err := os.Open(tmp)
	if err != nil {
		return BackupFile{}, err
	}
	defer f.Close()
	name := fmt.Sprintf("goals-backup-%s.db", time.Now().Format("2006-01-02-150405"))
	created, err := svc.Files.Create(&drive.File{
		Name:    name,
		Parents: []string{folderID},
	}).Media(f).Fields("id,name,createdTime,size").Do()
	if err != nil {
		return BackupFile{}, err
	}
	return BackupFile{ID: created.Id, Name: created.Name, Created: created.CreatedTime, Size: created.Size}, nil
}

// ListBackups returns newest-first backup files from the app folder.
func ListBackups(s *store.Store) ([]BackupFile, error) {
	svc, err := driveService(s)
	if err != nil {
		return nil, err
	}
	folderID, err := ensureFolder(svc)
	if err != nil {
		return nil, err
	}
	list, err := svc.Files.List().
		Q(fmt.Sprintf("'%s' in parents and trashed=false", folderID)).
		OrderBy("createdTime desc").PageSize(50).
		Fields("files(id,name,createdTime,size)").Do()
	if err != nil {
		return nil, err
	}
	out := []BackupFile{}
	for _, f := range list.Files {
		out = append(out, BackupFile{ID: f.Id, Name: f.Name, Created: f.CreatedTime, Size: f.Size})
	}
	return out, nil
}

// DownloadBackup fetches a backup's bytes (restore is completed by the caller,
// which must close the DB first).
func DownloadBackup(s *store.Store, fileID string) ([]byte, error) {
	svc, err := driveService(s)
	if err != nil {
		return nil, err
	}
	resp, err := svc.Files.Get(fileID).Download()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// DeleteBackup removes one backup file from Drive.
func DeleteBackup(s *store.Store, fileID string) error {
	svc, err := driveService(s)
	if err != nil {
		return err
	}
	return svc.Files.Delete(fileID).Do()
}

// ValidSQLite reports whether data looks like a SQLite database file.
func ValidSQLite(data []byte) bool {
	return len(data) > 16 && string(data[:16]) == "SQLite format 3\x00"
}
