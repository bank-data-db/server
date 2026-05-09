package internal

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/shadiestgoat/bankDataDB/config"
	"github.com/shadiestgoat/bankDataDB/db"
	"github.com/shadiestgoat/bankDataDB/db/store"
	"github.com/shadiestgoat/bankDataDB/snownode"
	"golang.org/x/crypto/bcrypt"
)

const (
	JWT_ISSUER     = "bank_data"
	TOKEN_DURATION = 2 * time.Hour
)

var (
	jwtParser = jwt.NewParser(
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(JWT_ISSUER),
		jwt.WithIssuedAt(),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Name}),
	)
)

// Exchanges a JWT auth token for a userID. Will return nil if the token is not valid
func ExchangeToken(ctx context.Context, store store.Store, t string) *string {
	tok, err := jwtParser.Parse(t, func(t *jwt.Token) (any, error) {
		return config.JWT_SECRET, nil
	})
	if err != nil {
		return nil
	}
	c, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		slog.ErrorContext(ctx, "Received a token w/ claims that aren't a MapClaims???")

		return nil
	}

	rawUserID, ok := c["usr"]
	if !ok {
		return nil
	}

	userID, ok := rawUserID.(string)
	if !ok {
		return nil
	}

	issuedAt, err := c.GetIssuedAt()
	if err != nil {
		return nil
	}

	userUpdatedAt, err := store.UserUpdatedAt(ctx, userID)
	if err != nil || userUpdatedAt.After(issuedAt.Time) {
		return nil
	}

	return &userID
}

var ErrBadAuth = errors.New("bad auth")

// Exchange a username and password for a JWT
func Login(ctx context.Context, s store.Store, username, inpPassword string) (string, error) {
	usr, err := s.UserByName(ctx, username)
	if err != nil {
		slog.Info("meow")
		slog.DebugContext(ctx, "User doesn't exist", "name", username)
		time.Sleep(time.Duration(rand.Float64()) * time.Second)
		return "", ErrBadAuth
	}

	if err := bcrypt.CompareHashAndPassword([]byte(usr.Password), []byte(inpPassword)); err != nil {
		slog.DebugContext(ctx, "User has a bad password", "name", username)
		time.Sleep(time.Duration(rand.Float64()) * time.Second)
		return "", ErrBadAuth
	}

	slog.DebugContext(ctx, "Logged in", "username", username)

	return newToken(ctx, usr.ID)
}

func utilPasswordGen(pass string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(pass), 0)
}

func CreateUser(ctx context.Context, db db.DBQuerier, name, password string) (string, error) {
	// TODO: Maybe this should be in store/db??

	id := snownode.NewID()

	pass, err := utilPasswordGen(password)
	if err != nil {
		return "", err
	}

	_, err = db.Exec(ctx, `INSERT INTO users (id, username, password) VALUES ($1, $2, $3)`, id, name, pass)

	return id, err
}

// Creates a new JWT for a specific userID. Does not do validations or anything like that on userID.
func newToken(ctx context.Context, userID string) (string, error) {
	str, err := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"iss": JWT_ISSUER,
		"iat": float64(time.Now().Unix()),
		"exp": float64(time.Now().Add(TOKEN_DURATION).Unix()),
		"usr": userID,
	}).SignedString(config.JWT_SECRET)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create a JWT token", "error", err)
	}

	return str, err
}
