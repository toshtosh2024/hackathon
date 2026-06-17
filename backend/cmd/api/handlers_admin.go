package main

import (
	"net/http"
)

func (a *app) getAdminStats(w http.ResponseWriter, r *http.Request) {
	db := a.dbHandle()

	var totalUsers, totalItems, totalPurchases int
	var totalSales int64

	_ = db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM users").Scan(&totalUsers)
	_ = db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM items").Scan(&totalItems)
	_ = db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM purchases").Scan(&totalPurchases)
	_ = db.QueryRowContext(r.Context(), "SELECT COALESCE(SUM(price), 0) FROM purchases").Scan(&totalSales)

	// User registration trend over the last 30 days
	uRows, err := db.QueryContext(r.Context(), `
		SELECT DATE_FORMAT(created_at, '%Y-%m-%d') as dt, COUNT(*) 
		FROM users 
		WHERE created_at >= DATE_SUB(CURRENT_TIMESTAMP, INTERVAL 30 DAY) 
		GROUP BY dt 
		ORDER BY dt ASC`,
	)
	var userTrends []map[string]any
	if err == nil {
		defer uRows.Close()
		for uRows.Next() {
			var dt string
			var count int
			if err := uRows.Scan(&dt, &count); err == nil {
				userTrends = append(userTrends, map[string]any{"date": dt, "count": count})
			}
		}
	}

	// Sales trend over the last 30 days
	sRows, err := db.QueryContext(r.Context(), `
		SELECT DATE_FORMAT(created_at, '%Y-%m-%d') as dt, COALESCE(SUM(price), 0) 
		FROM purchases 
		WHERE created_at >= DATE_SUB(CURRENT_TIMESTAMP, INTERVAL 30 DAY) 
		GROUP BY dt 
		ORDER BY dt ASC`,
	)
	var salesTrends []map[string]any
	if err == nil {
		defer sRows.Close()
		for sRows.Next() {
			var dt string
			var sum int64
			if err := sRows.Scan(&dt, &sum); err == nil {
				salesTrends = append(salesTrends, map[string]any{"date": dt, "amount": sum})
			}
		}
	}

	// Category distribution stats
	cRows, err := db.QueryContext(r.Context(), `
		SELECT category, COUNT(*), COALESCE(SUM(price), 0) 
		FROM items 
		GROUP BY category`,
	)
	var categoryStats []map[string]any
	if err == nil {
		defer cRows.Close()
		for cRows.Next() {
			var cat string
			var count int
			var sum int64
			if err := cRows.Scan(&cat, &count, &sum); err == nil {
				categoryStats = append(categoryStats, map[string]any{
					"category":     cat,
					"itemCount":    count,
					"totalRevenue": sum,
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"totalUsers":       totalUsers,
		"totalItems":       totalItems,
		"totalPurchases":   totalPurchases,
		"totalSales":       totalSales,
		"userTrends30Days": userTrends,
		"sales30Days":      salesTrends,
		"categoryStats":    categoryStats,
	})
}

func (a *app) getAdminModerations(w http.ResponseWriter, r *http.Request) {
	rows, err := a.dbHandle().QueryContext(r.Context(), `
		SELECT im.id, im.item_id, i.title, u.name, im.prohibited, im.risk_level, im.reasons, im.blocked_keywords, im.created_at
		FROM item_moderations im
		JOIN items i ON i.id = im.item_id
		JOIN users u ON u.id = i.seller_id
		ORDER BY im.created_at DESC`,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query moderations")
		return
	}
	defer rows.Close()

	moderations := []map[string]any{}
	for rows.Next() {
		var m struct {
			ID              int64
			ItemID          int64
			ItemTitle       string
			SellerName      string
			Prohibited      bool
			RiskLevel       string
			Reasons         string
			BlockedKeywords string
			CreatedAt       string
		}
		if err := rows.Scan(&m.ID, &m.ItemID, &m.ItemTitle, &m.SellerName, &m.Prohibited, &m.RiskLevel, &m.Reasons, &m.BlockedKeywords, &m.CreatedAt); err == nil {
			moderations = append(moderations, map[string]any{
				"id":              m.ID,
				"itemId":          m.ItemID,
				"itemTitle":       m.ItemTitle,
				"sellerName":      m.SellerName,
				"prohibited":      m.Prohibited,
				"riskLevel":       m.RiskLevel,
				"reasons":         m.Reasons,
				"blockedKeywords": m.BlockedKeywords,
				"createdAt":       m.CreatedAt,
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"moderations": moderations})
}

func (a *app) getAdminUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := a.dbHandle().QueryContext(r.Context(), "SELECT id, name, email, role, COALESCE(avatar_url, ''), created_at FROM users ORDER BY created_at DESC")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load users")
		return
	}
	defer rows.Close()

	users := []user{}
	for rows.Next() {
		var u user
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.AvatarURL, &u.CreatedAt); err == nil {
			users = append(users, u)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"users": users})
}

func (a *app) updateUserRole(w http.ResponseWriter, r *http.Request) {
	db := a.dbHandle()
	if db == nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}

	userID, ok := pathID(w, r)
	if !ok {
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.Role != "user" && req.Role != "admin" {
		writeError(w, http.StatusBadRequest, "invalid role: must be user or admin")
		return
	}

	res, err := db.ExecContext(r.Context(), "UPDATE users SET role = ? WHERE id = ?", req.Role, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user role")
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
