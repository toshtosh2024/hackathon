package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type authUserKey struct{}

func (a *app) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			writeError(w, http.StatusUnauthorized, "authorization required")
			return
		}
		u, err := a.verifyToken(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authUserKey{}, u)))
	}
}

func currentUser(r *http.Request) user {
	u, _ := r.Context().Value(authUserKey{}).(user)
	return u
}

func (a *app) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := currentUser(r)
		if u.ID == 0 {
			writeError(w, http.StatusUnauthorized, "authorization required")
			return
		}
		if u.Role != "admin" {
			writeError(w, http.StatusForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeErrorDetail(w, status, msg, nil)
}

func writeErrorDetail(w http.ResponseWriter, status int, msg string, detail map[string]any) {
	res := map[string]any{"error": msg}
	for k, v := range detail {
		res[k] = v
	}
	writeJSON(w, status, res)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	val := r.PathValue("id")
	if val == "" {
		writeError(w, http.StatusBadRequest, "missing id parameter")
		return 0, false
	}
	id, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id parameter")
		return 0, false
	}
	return id, true
}
