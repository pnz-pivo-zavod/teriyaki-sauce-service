package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const _botToken = "123456:TEST-TOKEN"

var _now = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

func TestValidate(t *testing.T) {
	type testCase struct {
		name string

		giveBotToken string
		giveInitData string

		wantUserID      int64
		wantErr         error
		wantErrContains string
	}

	tests := []testCase{
		{
			name:         "валидная подпись, свежий auth_date -> ID пользователя",
			giveBotToken: _botToken,
			giveInitData: sign(_botToken, _now.Add(-time.Hour), `{"id":42,"first_name":"Ilya"}`),
			wantUserID:   42,
		},
		{
			name:            "подписано другим токеном -> ErrInvalidInitData",
			giveBotToken:    _botToken,
			giveInitData:    sign("654321:OTHER", _now, `{"id":42}`),
			wantErr:         ErrInvalidInitData,
			wantErrContains: "invalid hash",
		},
		{
			name:         "нет hash -> ErrInvalidInitData",
			giveBotToken: _botToken,
			giveInitData: "auth_date=1&user=%7B%22id%22%3A42%7D",
			wantErr:      ErrInvalidInitData,
		},
		{
			name:         "auth_date старше 24ч -> ErrExpired, даже при валидной подписи",
			giveBotToken: _botToken,
			giveInitData: sign(_botToken, _now.Add(-25*time.Hour), `{"id":42}`),
			wantErr:      ErrExpired,
		},
		{
			name:            "подпись валидна, но нет user -> ErrInvalidInitData",
			giveBotToken:    _botToken,
			giveInitData:    sign(_botToken, _now, ""),
			wantErr:         ErrInvalidInitData,
			wantErrContains: "no user",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			userID, err := Validate(tt.giveBotToken, tt.giveInitData, _now)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				if tt.wantErrContains != "" {
					require.Contains(t, err.Error(), tt.wantErrContains)
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantUserID, userID)
		})
	}
}

// sign собирает initData так же, как Telegram:
// https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
func sign(botToken string, authDate time.Time, user string) string {
	values := url.Values{"auth_date": {strconv.FormatInt(authDate.Unix(), 10)}}
	if user != "" {
		values.Set("user", user)
	}

	pairs := make([]string, 0, len(values))
	for k := range values {
		pairs = append(pairs, k+"="+values.Get(k))
	}
	slices.Sort(pairs)

	secret := hmacSHA256([]byte("WebAppData"), []byte(botToken))
	values.Set("hash", hex.EncodeToString(hmacSHA256(secret, []byte(strings.Join(pairs, "\n")))))

	return values.Encode()
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
