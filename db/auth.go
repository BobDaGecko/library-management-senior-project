package db

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

const JWT_LIFETIME_MS = WEEK * 4 // One month login time
const JWT_LIFETIME = int64(JWT_LIFETIME_MS / time.Second)

var JWT_ENC = base64.RawURLEncoding

// jwtSecret is the HMAC-SHA256 signing key for session tokens. It is loaded
// from the JWT_SECRET environment variable when set; otherwise a random key is
// generated and persisted to .jwt-secret (gitignored) so sessions survive
// restarts. It must never be hardcoded in source.
var jwtSecret = loadJWTSecret()

func loadJWTSecret() []byte {
	if env := os.Getenv("JWT_SECRET"); env != "" {
		return []byte(env)
	}

	if testing.Testing() {
		// Ephemeral per-process key; tests never need cross-process tokens.
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			panic(err)
		}
		return key
	}

	const path = ".jwt-secret"
	if data, err := os.ReadFile(path); err == nil && len(data) >= 32 {
		return data
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, key, 0o600); err != nil {
		panic(err)
	}
	return key
}

func CookieAuth(r *http.Request) *UserPartial {
	cookie, err := r.Cookie("tok")
	if err != nil {
		return nil
	}

	data, _, err := ValidateJWT(cookie.Value)
	if err != nil {
		return nil
	}

	return data
}

var _ = Migrate(JwtEntry{})

type JwtEntry struct {
	BaseModel
	Token     string
	User      User
	UserID    SqlUUID
	ExpiresAt time.Time
}

func (u *User) IssueJWT() (JwtEntry, error) {
	db := Db()
	if err := db.Save(u).Error; err != nil {
		return JwtEntry{}, err
	}
	header := ToJsonB64(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})

	issuedAt := time.Now()
	partial := u.Partial()
	partial.SetTimestamp(issuedAt.Unix())
	claims := ToJsonB64(partial)

	token := header + "." + claims
	sig := hmac.New(sha256.New, jwtSecret)
	sig.Write([]byte(token))
	sum := JWT_ENC.EncodeToString(sig.Sum(nil))
	token += "." + sum

	ret := JwtEntry{
		Token:     token,
		UserID:    u.ID,
		ExpiresAt: issuedAt.Add(JWT_LIFETIME_MS),
	}

	if err := db.Save(&ret).Error; err != nil {
		return JwtEntry{}, err
	}
	return ret, nil
}

// RevokeJWT deletes the stored entry for a token, invalidating it server-side.
func RevokeJWT(token string) error {
	return Db().Where("token = ?", token).Delete(&JwtEntry{}).Error
}

func ToJsonB64(dataMap any) string {
	dataStr, err := json.Marshal(dataMap)
	if err != nil {
		// Should be unreachable
		panic(err)
	}

	return JWT_ENC.EncodeToString(dataStr)
}

func ValidateJWT(jwt string) (*UserPartial, string, error) {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return nil, "jwt.split", errors.New("malformed jwt")
	}

	sig := hmac.New(sha256.New, jwtSecret)
	_, _ = sig.Write([]byte(parts[0] + "." + parts[1]))
	expected := JWT_ENC.EncodeToString(sig.Sum(nil))
	if !hmac.Equal([]byte(parts[2]), []byte(expected)) {
		return nil, "jwt.sig", errors.New("signature mismatch")
	}

	// Only HMAC-SHA256 is ever issued; reject anything else so a future
	// multi-algorithm change can't introduce algorithm-confusion bugs.
	headerData, err := JWT_ENC.DecodeString(parts[0])
	if err != nil {
		return nil, "jwt.header", err
	}
	header := struct {
		Alg string `json:"alg"`
	}{}
	if err := json.Unmarshal(headerData, &header); err != nil {
		return nil, "jwt.header", err
	}
	if header.Alg != "HS256" {
		return nil, "jwt.alg", errors.New("unsupported algorithm")
	}

	data, err := JWT_ENC.DecodeString(parts[1])
	if err != nil {
		return nil, "jwt.b64decode", err
	}

	ret := UserPartial{}
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, "jwt.unmarshal", err
	}

	expiresAt := time.Unix(ret.ExpiresAt, 0)
	if time.Now().After(expiresAt) {
		return nil, "user.security", errors.New("expired")
	}

	// Server-side revocation: a token is only valid while its entry exists.
	var count int64
	if err := Db().Model(&JwtEntry{}).Where("token = ?", jwt).Count(&count).Error; err != nil {
		return nil, "jwt.lookup", err
	}
	if count == 0 {
		return nil, "jwt.revoked", errors.New("token revoked")
	}

	return &ret, "", nil
}
