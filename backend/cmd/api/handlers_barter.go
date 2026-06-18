package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type node struct {
	ItemID       int64
	UserID       int64
	Category     string
	WantCategory string
	Price        int
}

func (a *app) listBarterLoops(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	rows, err := a.dbHandle().QueryContext(r.Context(), `
		SELECT lm.loop_id 
		FROM barter_loop_members lm
		WHERE lm.user_id = ?
		ORDER BY lm.loop_id DESC`, u.ID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query loop IDs: "+err.Error())
		return
	}
	defer rows.Close()

	var loopIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			loopIDs = append(loopIDs, id)
		}
	}

	loops := []map[string]any{}
	for _, lid := range loopIDs {
		var bl struct {
			ID            int64
			Status        string
			Justification string
			CreatedAt     string
		}
		err = a.dbHandle().QueryRowContext(r.Context(),
			"SELECT id, status, justification, created_at FROM barter_loops WHERE id = ?", lid,
		).Scan(&bl.ID, &bl.Status, &bl.Justification, &bl.CreatedAt)
		if err != nil {
			continue
		}

		mRows, err := a.dbHandle().QueryContext(r.Context(), `
			SELECT lm.id, lm.user_id, u_m.name, lm.item_id, i_m.title, i_m.category, lm.shipping_status, lm.cash_adjustment
			FROM barter_loop_members lm
			JOIN users u_m ON u_m.id = lm.user_id
			JOIN items i_m ON i_m.id = lm.item_id
			WHERE lm.loop_id = ?
			ORDER BY lm.id ASC`, lid,
		)
		if err != nil {
			continue
		}
		defer mRows.Close()

		members := []map[string]any{}
		for mRows.Next() {
			var m struct {
				ID             int64
				UserID         int64
				UserName       string
				ItemID         int64
				ItemTitle      string
				ItemCategory   string
				ShippingStatus string
				Adjustment     int
			}
			if err := mRows.Scan(&m.ID, &m.UserID, &m.UserName, &m.ItemID, &m.ItemTitle, &m.ItemCategory, &m.ShippingStatus, &m.Adjustment); err == nil {
				members = append(members, map[string]any{
					"memberId":       m.ID,
					"userId":         m.UserID,
					"userName":       m.UserName,
					"itemId":         m.ItemID,
					"itemTitle":      m.ItemTitle,
					"itemCategory":   m.ItemCategory,
					"shippingStatus": m.ShippingStatus,
					"adjustment":     m.Adjustment,
				})
			}
		}

		loops = append(loops, map[string]any{
			"id":            bl.ID,
			"status":        bl.Status,
			"justification": bl.Justification,
			"createdAt":     bl.CreatedAt,
			"members":       members,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"loops": loops})
}

func (a *app) acceptBarterLoop(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	loopID, ok := pathID(w, r)
	if !ok {
		return
	}

	tx, err := a.dbHandle().BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Update member state to accepted
	_, err = tx.ExecContext(r.Context(), "UPDATE barter_loop_members SET shipping_status = 'accepted' WHERE loop_id = ? AND user_id = ?", loopID, u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to accept loop: "+err.Error())
		return
	}

	// If all members accepted, activate loop!
	var pendingCount int
	err = tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM barter_loop_members WHERE loop_id = ? AND shipping_status = 'pending'", loopID).Scan(&pendingCount)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to count state")
		return
	}

	if pendingCount == 0 {
		_, err = tx.ExecContext(r.Context(), "UPDATE barter_loops SET status = 'active' WHERE id = ?", loopID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to activate loop")
			return
		}
		// Mark items as sold
		_, err = tx.ExecContext(r.Context(), "UPDATE items SET status = 'sold' WHERE id IN (SELECT item_id FROM barter_loop_members WHERE loop_id = ?)", loopID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update items as sold")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (a *app) shipBarterLoop(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	loopID, ok := pathID(w, r)
	if !ok {
		return
	}

	_, err := a.dbHandle().ExecContext(r.Context(), "UPDATE barter_loop_members SET shipping_status = 'shipped' WHERE loop_id = ? AND user_id = ?", loopID, u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to ship: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (a *app) receiveBarterLoop(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	loopID, ok := pathID(w, r)
	if !ok {
		return
	}

	tx, err := a.dbHandle().BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(r.Context(), "UPDATE barter_loop_members SET shipping_status = 'received' WHERE loop_id = ? AND receiver_id = ?", loopID, u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mark as received: "+err.Error())
		return
	}

	var unreceivedCount int
	err = tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM barter_loop_members WHERE loop_id = ? AND shipping_status != 'received'", loopID).Scan(&unreceivedCount)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check shipping states")
		return
	}

	if unreceivedCount == 0 {
		_, err = tx.ExecContext(r.Context(), "UPDATE barter_loops SET status = 'completed' WHERE id = ?", loopID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to complete loop")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// Background Daemon: Barter Loop Matcher

func (a *app) initDBLoop() {
	time.Sleep(2 * time.Second) // wait for database startup
	db := a.dbHandle()
	if db == nil {
		log.Println("Database connection is nil inside background matcher loop. Skipping.")
		return
	}

	log.Println("Starting background Barter Matcher daemon (every 30 minutes)...")
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	// Initial run
	a.runBarterMatcher()

	for range ticker.C {
		a.runBarterMatcher()
	}
}

func (a *app) runBarterMatcher() {
	db := a.dbHandle()
	if db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Density optimization: Only execute matches if we have 2 or more active barter items
	var activeBarters int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM items WHERE barter_enabled = 1 AND status = 'active'").Scan(&activeBarters)
	if err != nil || activeBarters < 2 {
		log.Printf("Barter Matcher: Market density is too low (%d active barter items). Skipping scan.", activeBarters)
		return
	}

	log.Println("Barter Matcher: Density check passed. Scanning dependency graphs...")

	// 1. Fetch active barter participants
	rows, err := db.QueryContext(ctx, `
		SELECT i.id, i.seller_id, i.category, i.want_category, i.price
		FROM items i
		WHERE i.barter_enabled = 1 AND i.status = 'active'
	`)
	if err != nil {
		log.Printf("Barter Matcher: failed to load candidates: %v", err)
		return
	}
	defer rows.Close()

	var nodes []node
	for rows.Next() {
		var n node
		if err := rows.Scan(&n.ItemID, &n.UserID, &n.Category, &n.WantCategory, &n.Price); err == nil {
			nodes = append(nodes, n)
		}
	}

	if len(nodes) < 2 {
		return
	}

	// 2. Run DFS Backtracking Cycle Finder
	cycles := checkGraphCycles(nodes)
	if len(cycles) == 0 {
		log.Println("Barter Matcher: No cyclic loops (size 2 or 3) discovered in current market session.")
		return
	}

	// Pick the first discovered loop
	loop := cycles[0]
	log.Printf("Barter Matcher: Discovered compatible cycle! Size: %d, ItemIDs: %v", len(loop), loop)

	// 3. Balance values using OpenAI
	justification, adjustments, err := a.balancedBarterClearing(ctx, loop, nodes)
	if err != nil {
		log.Printf("Barter Matcher: Failed to balance loop values: %v. Skipping.", err)
		return
	}

	// 4. Record loop in transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Barter Matcher: Failed to start transaction: %v", err)
		return
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, "INSERT INTO barter_loops (status, justification) VALUES ('pending', ?)", justification)
	if err != nil {
		log.Printf("Barter Matcher: Failed to record loop: %v", err)
		return
	}
	loopID, _ := res.LastInsertId()

	for i, itemID := range loop {
		var n node
		for _, nd := range nodes {
			if nd.ItemID == itemID {
				n = nd
				break
			}
		}

		// In a cycle, Receiver is the NEXT member in the loop array (loop wraps around to 0)
		nextIdx := (i + 1) % len(loop)
		receiverItemID := loop[nextIdx]
		var receiverNode node
		for _, nd := range nodes {
			if nd.ItemID == receiverItemID {
				receiverNode = nd
				break
			}
		}

		adj := adjustments[itemID]

		_, err = tx.ExecContext(ctx, `
			INSERT INTO barter_loop_members (loop_id, user_id, item_id, receiver_id, cash_adjustment, shipping_status) 
			VALUES (?, ?, ?, ?, ?, 'pending')`,
			loopID, n.UserID, n.ItemID, receiverNode.UserID, adj,
		)
		if err != nil {
			log.Printf("Barter Matcher: Failed to save loop member: %v", err)
			return
		}

		// Flag item as 'hidden' or 'pending' so it doesn't show in standard searches
		_, _ = tx.ExecContext(ctx, "UPDATE items SET status = 'hidden' WHERE id = ?", n.ItemID)
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Barter Matcher: Failed to commit matched loop: %v", err)
		return
	}

	log.Printf("Barter Matcher: Successfully locked and established Loop ID: #%d!", loopID)
}

func checkGraphCycles(nodes []node) [][]int64 {
	var cycles [][]int64

	// Build Adjacency Matrix: edge from A to B if Seller A's item category matches Buyer B's wanted category,
	// and they are different users.
	adj := make(map[int64][]int64)
	for _, a := range nodes {
		for _, b := range nodes {
			if a.UserID != b.UserID && a.Category == b.WantCategory {
				adj[a.ItemID] = append(adj[a.ItemID], b.ItemID)
			}
		}
	}

	// Solve using Backtracking DFS for size 2 and size 3 cycles
	var dfs func(curr int64, start int64, path []int64, visited map[int64]bool)
	dfs = func(curr int64, start int64, path []int64, visited map[int64]bool) {
		if len(path) > 3 {
			return
		}
		for _, next := range adj[curr] {
			if next == start && len(path) >= 2 {
				// Cycle found!
				cpy := make([]int64, len(path))
				copy(cpy, path)
				cycles = append(cycles, cpy)
				return
			}
			if !visited[next] {
				visited[next] = true
				dfs(next, start, append(path, next), visited)
				visited[next] = false
			}
		}
	}

	for _, n := range nodes {
		vMap := make(map[int64]bool)
		vMap[n.ItemID] = true
		dfs(n.ItemID, n.ItemID, []int64{n.ItemID}, vMap)
		if len(cycles) > 0 {
			break // return the first cycle discovered
		}
	}

	return cycles
}

func (a *app) balancedBarterClearing(ctx context.Context, loop []int64, nodes []node) (string, map[int64]int, error) {
	type itemInfo struct {
		ItemID   int64  `json:"itemId"`
		Title    string `json:"title"`
		Category string `json:"category"`
		Price    int    `json:"price"`
	}

	var targetItems []itemInfo
	for _, id := range loop {
		for _, n := range nodes {
			if n.ItemID == id {
				var title string
				_ = a.dbHandle().QueryRowContext(ctx, "SELECT title FROM items WHERE id = ?", id).Scan(&title)
				targetItems = append(targetItems, itemInfo{
					ItemID:   id,
					Title:    title,
					Category: n.Category,
					Price:    n.Price,
				})
			}
		}
	}

	// 🏆 1. Calculate mathematically perfect zero-sum adjustments programmatically in Go
	// Formula: CashReceived_i = Price(GivenItem_i) - Price(ReceivedItem_i)
	mathAdjustments := make(map[int64]int)
	var explanationParts []string

	for i, id := range loop {
		var givenPrice int
		var title string
		for _, item := range targetItems {
			if item.ItemID == id {
				givenPrice = item.Price
				title = item.Title
				break
			}
		}

		// Received item is the NEXT item in the circular loop array
		nextIdx := (i + 1) % len(loop)
		receivedID := loop[nextIdx]
		var receivedPrice int
		for _, item := range targetItems {
			if item.ItemID == receivedID {
				receivedPrice = item.Price
				break
			}
		}

		// Cash adjustment: positive means receive, negative means pay
		adj := givenPrice - receivedPrice
		mathAdjustments[id] = adj
		explanationParts = append(explanationParts, fmt.Sprintf("・「%s」(価格: ¥%d) の提供者 ➔ 清算金収支: ¥%d", title, givenPrice, adj))
	}

	explanationList := strings.Join(explanationParts, "\n")
	itemsJSON, _ := json.Marshal(targetItems)

	prompt := fmt.Sprintf(`あなたは物々交換の調停AIファイナンシャルアドバイザーです。
以下の%d個の商品を循環して交換する物々交換ループを調停してください。

対象商品リスト:
%s

等価交換（参加者全員の取引完了時の純利益がちょうど「0円」となる状態）を完全に成立させるため、システムは以下の数式モデル（提供商品価格 - 受領商品価格）に基づき、各メンバーの「清算調整金」を正確に算定しました。
算定済みの清算調整金（合計は必ずぴったり0円となります）：
%s

この価格差調整（清算調整金）の意義、全員が等価・公平に循環トレードできる仕組みの妥当性について、プロフェッショナルで親切な「査定・解説理由」を日本語（100文字以上）で作成してください。

必ず以下のJSONフォーマットのみで出力してください（マークダウンのコードブロックで囲まず、生のJSONテキストのみを出力してください）：
{
  "justification": "数式モデルの解説を交えた、公平な取引理由に関するプロフェッショナルな日本語解説（100文字以上）"
}`, len(loop), string(itemsJSON), explanationList)

	var res struct {
		Justification string `json:"justification"`
	}

	err := a.callOpenAIJSON(ctx, prompt, &res)
	if err != nil {
		return "", nil, err
	}

	return res.Justification, mathAdjustments, nil
}
