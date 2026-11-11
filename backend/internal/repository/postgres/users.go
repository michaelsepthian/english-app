// MIVA-DATA: persistence contracts and PostgreSQL implementations.
package postgres

import (
	"context"
	"encoding/json"
	"time"

	"english-app/internal/model"
	"english-app/internal/platform/jsonutil"
)

func (r *Users) find(ctx context.Context, field, value string) (model.User, string, error) {
	var u model.User
	var password string
	var raw []byte
	err := r.db.QueryRow(ctx, "SELECT id,email,password,settings FROM english_app.users WHERE "+field+"=$1", value).Scan(&u.ID, &u.Email, &password, &raw)
	if err != nil {
		return u, "", persistenceError(err)
	}
	err = json.Unmarshal(raw, &u.Settings)
	return u, password, err
}
func (r *Users) FindByID(ctx context.Context, id string) (model.User, error) {
	u, _, err := r.find(ctx, "id", id)
	return u, err
}
func (r *Users) FindByEmail(ctx context.Context, email string) (model.User, string, error) {
	return r.find(ctx, "email", email)
}
func (r *Users) Create(ctx context.Context, u model.User, password string) error {
	_, err := r.db.Exec(ctx, "INSERT INTO english_app.users(id,email,password,settings) VALUES($1,$2,$3,$4)", u.ID, u.Email, password, jsonutil.Marshal(u.Settings))
	return persistenceError(err)
}
func (r *Users) UpdateSettings(ctx context.Context, id string, settings model.Settings) error {
	_, err := r.db.Exec(ctx, "UPDATE english_app.users SET settings=$1 WHERE id=$2", jsonutil.Marshal(settings), id)
	return err
}
func (r *Sessions) FindUserID(ctx context.Context, hash string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, "SELECT user_id FROM english_app.sessions WHERE token=$1 AND expires>now()", hash).Scan(&id)
	return id, persistenceError(err)
}
func (r *Sessions) Create(ctx context.Context, id, hash string, expires time.Time) error {
	_, err := r.db.Exec(ctx, "INSERT INTO english_app.sessions(token,user_id,expires) VALUES($1,$2,$3)", hash, id, expires)
	return err
}
func (r *Sessions) Delete(ctx context.Context, hash string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM english_app.sessions WHERE token=$1", hash)
	return err
}
