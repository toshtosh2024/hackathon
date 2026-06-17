package main

import (
	"database/sql"
	"net/http"
	"strings"
)

func (a *app) listConversations(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	rows, err := a.dbHandle().QueryContext(r.Context(), `
		SELECT c.id, c.item_id, i.title, i.price, i.status,
		       COALESCE((SELECT image_url FROM item_images WHERE item_id = i.id ORDER BY sort_order LIMIT 1), ''),
		       i.category, c.buyer_id, c.seller_id,
		       u_counter.id, u_counter.name, COALESCE(u_counter.avatar_url, ''),
		       COALESCE(p.id, 0) as purchase_id, COALESCE(p.status, '') as purchase_status, c.updated_at
		FROM conversations c
		JOIN items i ON i.id = c.item_id
		JOIN users u_counter ON u_counter.id = IF(c.buyer_id = ?, c.seller_id, c.buyer_id)
		LEFT JOIN purchases p ON p.item_id = c.item_id AND (p.buyer_id = c.buyer_id OR p.seller_id = c.seller_id)
		WHERE c.buyer_id = ? OR c.seller_id = ?
		ORDER BY c.updated_at DESC`, u.ID, u.ID, u.ID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load conversations: "+err.Error())
		return
	}
	defer rows.Close()

	conversations := []conversation{}
	for rows.Next() {
		var c conversation
		if err := rows.Scan(
			&c.ID, &c.ItemID, &c.ItemTitle, &c.ItemPrice, &c.ItemStatus,
			&c.ItemImageURL, &c.ItemCategory, &c.BuyerID, &c.SellerID,
			&c.CounterpartID, &c.CounterpartName, &c.CounterpartAvatarURL,
			&c.PurchaseID, &c.PurchaseStatus, &c.UpdatedAt,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read conversation")
			return
		}
		conversations = append(conversations, c)
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversations": conversations})
}

func (a *app) createConversation(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		ItemID int64 `json:"itemId"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	it, err := a.findItem(r.Context(), req.ItemID)
	if err != nil {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}

	if it.SellerID == u.ID {
		writeError(w, http.StatusBadRequest, "cannot start conversation with yourself")
		return
	}

	db := a.dbHandle()
	var conversationID int64
	err = db.QueryRowContext(r.Context(), "SELECT id FROM conversations WHERE item_id = ? AND buyer_id = ?", req.ItemID, u.ID).Scan(&conversationID)
	if err == sql.ErrNoRows {
		res, err := db.ExecContext(r.Context(),
			"INSERT INTO conversations (item_id, buyer_id, seller_id) VALUES (?, ?, ?)",
			req.ItemID, u.ID, it.SellerID,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create conversation")
			return
		}
		conversationID, _ = res.LastInsertId()
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query conversation")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"conversationId": conversationID})
}

func (a *app) listMessages(w http.ResponseWriter, r *http.Request) {
	conversationID, ok := pathID(w, r)
	if !ok {
		return
	}

	u := currentUser(r)
	var buyerID, sellerID int64
	err := a.dbHandle().QueryRowContext(r.Context(), "SELECT buyer_id, seller_id FROM conversations WHERE id = ?", conversationID).Scan(&buyerID, &sellerID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "conversation not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to check conversation context")
		}
		return
	}
	if buyerID != u.ID && sellerID != u.ID {
		writeError(w, http.StatusForbidden, "you are not a participant in this conversation")
		return
	}

	rows, err := a.dbHandle().QueryContext(r.Context(),
		"SELECT id, conversation_id, sender_id, body, created_at FROM messages WHERE conversation_id = ? ORDER BY created_at ASC",
		conversationID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load messages")
		return
	}
	defer rows.Close()

	messages := []message{}
	for rows.Next() {
		var m message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read message")
			return
		}
		messages = append(messages, m)
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": messages})
}

func (a *app) createMessage(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	conversationID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req struct {
		Body string `json:"body"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Body) == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}

	var buyerID, sellerID int64
	err := a.dbHandle().QueryRowContext(r.Context(), "SELECT buyer_id, seller_id FROM conversations WHERE id = ?", conversationID).Scan(&buyerID, &sellerID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "conversation not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to check conversation context")
		}
		return
	}
	if buyerID != u.ID && sellerID != u.ID {
		writeError(w, http.StatusForbidden, "you are not a participant in this conversation")
		return
	}

	res, err := a.dbHandle().ExecContext(r.Context(),
		"INSERT INTO messages (conversation_id, sender_id, body) VALUES (?, ?, ?)",
		conversationID, u.ID, req.Body,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create message")
		return
	}
	_, _ = a.dbHandle().ExecContext(r.Context(), "UPDATE conversations SET updated_at = CURRENT_TIMESTAMP WHERE id = ?", conversationID)
	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]any{"messageId": id})
}
