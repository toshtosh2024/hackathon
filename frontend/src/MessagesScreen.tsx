/**
 * @file MessagesScreen.tsx
 * @description Next Market - 会話・取引ナビ画面（Stripeエスクロー、3段階配送ナビ、受取評価）
 */

import React, { useState, FormEvent } from "react";
import { MessageCircle, Send, Star } from "lucide-react";
import { User, Conversation, Message, getPublicUrl } from "./types";

// Local helper to format dates beautifully
function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    return `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
  } catch {
    return dateStr;
  }
}

// Local helper to translate status into friendly labels
function statusLabel(status: "active" | "sold" | "hidden") {
  if (status === "sold") return "売却済み";
  if (status === "hidden") return "公開停止";
  return "販売中";
}

interface MessagesScreenProps {
  user: User | null;
  conversations: Conversation[];
  selectedConversation: Conversation | null;
  messages: Message[];
  api: <T>(path: string, options?: RequestInit) => Promise<T>;
  onSelect: (conversationId: number) => void;
  onOpenItem: (itemId: number) => void;
  onRefreshConversations?: () => Promise<void>;
}

export function MessagesScreen({
  user,
  conversations,
  selectedConversation,
  messages,
  api,
  onSelect,
  onOpenItem,
  onRefreshConversations
}: MessagesScreenProps) {
  const [body, setBody] = useState("購入前に状態をもう少し教えてください。");
  const [shippingLoading, setShippingLoading] = useState(false);

  async function send(event: FormEvent) {
    event.preventDefault();
    if (!selectedConversation) return;
    await api(`/conversations/${selectedConversation.id}/messages`, {
      method: "POST",
      body: JSON.stringify({ body })
    });
    setBody("");
    onSelect(selectedConversation.id);
  }

  async function shipItem() {
    if (!selectedConversation) return;
    setShippingLoading(true);
    try {
      await api(`/purchases/${selectedConversation.purchaseId}/ship`, { method: "POST" });
      if (onRefreshConversations) await onRefreshConversations();
      onSelect(selectedConversation.id);
    } catch (err) {
      alert(err instanceof Error ? err.message : "発送通知に失敗しました");
    } finally {
      setShippingLoading(false);
    }
  }

  async function receiveItem() {
    if (!selectedConversation) return;
    setShippingLoading(true);
    try {
      await api(`/purchases/${selectedConversation.purchaseId}/receive`, { method: "POST" });
      if (onRefreshConversations) await onRefreshConversations();
      onSelect(selectedConversation.id);
    } catch (err) {
      alert(err instanceof Error ? err.message : "受取報告に失敗しました");
    } finally {
      setShippingLoading(false);
    }
  }

  return (
    <section className="page-shell">
      <div className="split-heading">
        <div>
          <p className="eyebrow">Messages</p>
          <h2>DM</h2>
        </div>
      </div>

      <section className="message-layout">
        <article className="panel conversation-panel">
          <div className="panel-heading">
            <MessageCircle size={20} />
            <h3>会話一覧</h3>
          </div>
          <div className="conversation-list">
            {conversations.map((conversation) => (
              <button
                key={conversation.id}
                type="button"
                className={selectedConversation?.id === conversation.id ? "conversation active" : "conversation"}
                onClick={() => onSelect(conversation.id)}
              >
                <span className="icon-label">
                  <MessageCircle size={16} />
                  <span>{formatDate(conversation.updatedAt)}</span>
                </span>
                <strong>{conversation.itemTitle}</strong>
              </button>
            ))}
            {conversations.length === 0 && <p className="muted">会話はまだありません。</p>}
          </div>
        </article>

        <article className="panel thread-panel">
          <div className="panel-heading">
            <Send size={20} />
            <h3>{selectedConversation ? selectedConversation.itemTitle : "メッセージ"}</h3>
          </div>
          {selectedConversation && (
            <button type="button" className="conversation-item-card" onClick={() => onOpenItem(selectedConversation.itemId)}>
              <img src={getPublicUrl(selectedConversation.itemImageUrl) || "/placeholder.svg"} alt="" />
              <div className="conversation-item-copy">
                <div className="conversation-counterpart">
                  <img src={getPublicUrl(selectedConversation.counterpartAvatarUrl) || "/placeholder-avatar.svg"} alt="" />
                  <strong>{selectedConversation.counterpartName}</strong>
                </div>
                <strong>{selectedConversation.itemTitle}</strong>
                <span>¥{selectedConversation.itemPrice.toLocaleString()}</span>
                <small>
                  {selectedConversation.itemCategory} / {statusLabel(selectedConversation.itemStatus)}
                </small>
              </div>
            </button>
          )}

          {/* Escrow Transaction Navigator (取引ナビ) */}
          {selectedConversation && selectedConversation.itemStatus === "sold" && selectedConversation.purchaseStatus && (
            <div style={{ background: "#f8fafc", border: "1px solid #cbd5e1", borderRadius: "8px", padding: "16px", margin: "12px", display: "flex", flexDirection: "column", gap: "10px" }}>
              <span style={{ fontSize: "12px", fontWeight: 700, color: "#475569" }}>🤝 Stripe エスクロー取引ナビ</span>
              
              {selectedConversation.purchaseStatus === "paid" && (
                <div style={{ display: "flex", flexDirection: "column", gap: "8px" }}>
                  {user?.id === selectedConversation.sellerId ? (
                    <>
                      <p style={{ margin: 0, fontSize: "13px", color: "#1e293b", lineHeight: "1.4" }}>
                        🎉 <strong>購入者の支払いが完了しました！</strong><br />
                        売上金はStripeエスクロー（一時預かり）に安全に保護されています。商品を発送し、以下の「発送通知」ボタンを押してください。
                      </p>
                      <button className="primary-button" disabled={shippingLoading} onClick={shipItem} style={{ background: "#3b82f6", color: "#fff", border: "none", alignSelf: "start", padding: "8px 16px", fontSize: "13px", cursor: "pointer" }}>
                        📦 商品を発送したので発送通知をする
                      </button>
                    </>
                  ) : (
                    <p style={{ margin: 0, fontSize: "13px", color: "#047857", lineHeight: "1.4" }}>
                      🔒 <strong>決済が完了しました（エスクロー保護中）</strong><br />
                      代金は取引が完了するまで運営に安全に保護されています。出品者による商品の発送をお待ちください。
                    </p>
                  )}
                </div>
              )}

              {selectedConversation.purchaseStatus === "shipped" && (
                <div style={{ display: "flex", flexDirection: "column", gap: "8px" }}>
                  {user?.id === selectedConversation.sellerId ? (
                    <p style={{ margin: 0, fontSize: "13px", color: "#0369a1", lineHeight: "1.4" }}>
                      🚚 <strong>商品の発送を通知しました</strong><br />
                      商品は配送中です。購入者が受け取りを確認し、「受取評価」を行うと自動で売上残高が確定されます。
                    </p>
                  ) : (
                    <>
                      <p style={{ margin: 0, fontSize: "13px", color: "#1e293b", lineHeight: "1.4" }}>
                        🚚 <strong>出品者が商品を発送しました！</strong><br />
                        荷物が届いたら中身を確認し、問題がなければ「受取確認＆取引完了」ボタンを押してください。完了すると出品者へ売上金がリリースされます。
                      </p>
                      <button className="primary-button" disabled={shippingLoading} onClick={receiveItem} style={{ background: "#10b981", color: "#fff", border: "none", alignSelf: "start", padding: "8px 16px", fontSize: "13px", cursor: "pointer" }}>
                        ✅ 商品を受け取ったので取引を完了する
                      </button>
                    </>
                  )}
                </div>
              )}

              {selectedConversation.purchaseStatus === "completed" && (
                <p style={{ margin: 0, fontSize: "13px", color: "#047857", fontWeight: 600 }}>
                  🎉 この取引は完了しました！ありがとうございました。
                </p>
              )}
            </div>
          )}
          <div className="message-list">
            {messages.map((message) => (
              <p key={message.id} className={message.senderId === user?.id ? "message mine" : "message"}>
                {message.body}
              </p>
            ))}
            {selectedConversation && messages.length === 0 && <p className="muted">まだメッセージはありません。</p>}
            {!selectedConversation && <p className="muted">左の会話を選択してください。</p>}
          </div>
          <form className="message-form" onSubmit={send}>
            <input disabled={!selectedConversation} value={body} onChange={(e) => setBody(e.target.value)} />
            <button disabled={!selectedConversation}>
              <Send size={18} />
            </button>
          </form>
          {selectedConversation?.itemStatus === "sold" && (
            <ReviewComposer
              api={api}
              itemId={selectedConversation.itemId}
              counterpartName={selectedConversation.counterpartName}
            />
          )}
        </article>
      </section>
    </section>
  );
}

// ==========================================
// ReviewComposer Component (Moved modularly)
// ==========================================

export function ReviewComposer({
  api,
  itemId,
  counterpartName
}: {
  api: <T>(path: string, options?: RequestInit) => Promise<T>;
  itemId: number;
  counterpartName: string;
}) {
  const [rating, setRating] = useState(5);
  const [comment, setComment] = useState("スムーズで大変気持ちの良いお取引ができました。また機会がありましたらよろしくお願いいたします！");
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");

  const submitReview = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setMessage("");
    try {
      const pData = await api<{ conversations: Conversation[] }>("/conversations");
      const matched = pData.conversations.find((c) => c.itemId === itemId);
      if (!matched || !matched.purchaseId) {
        throw new Error("取引の決済記録が見つかりませんでした");
      }

      await api(`/items/${itemId}/reviews`, {
        method: "POST",
        body: JSON.stringify({ purchaseId: matched.purchaseId, rating, comment })
      });
      setMessage(`🎉 ${counterpartName} さんへの受取評価を投稿しました！`);
      setComment(""); // Clear comment on success to prevent double submission
    } catch (err) {
      setMessage(err instanceof Error ? err.message : "受取評価の投稿に失敗しました");
    } finally {
      setSaving(false);
    }
  };

  return (
    <form className="review-composer-form" onSubmit={submitReview} style={{ borderTop: "1px solid #eadfd3", paddingTop: "16px", marginTop: "16px", display: "flex", flexDirection: "column", gap: "12px" }}>
      <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
        <Star size={18} style={{ color: "#fbbf24" }} />
        <strong>🤝 取引相手（{counterpartName} さん）への受取評価を投稿</strong>
      </div>
      <div style={{ display: "flex", alignItems: "center", gap: "12px" }}>
        <label style={{ fontSize: "13px", fontWeight: 600 }}>評価レーティング:</label>
        <div style={{ display: "flex", gap: "4px" }}>
          {[1, 2, 3, 4, 5].map((num) => (
            <button key={num} type="button" onClick={() => setRating(num)} style={{ background: "none", border: "none", cursor: "pointer", padding: 0 }} disabled={message.startsWith("🎉")}>
              <Star size={20} fill={num <= rating ? "#fbbf24" : "none"} stroke={num <= rating ? "#fbbf24" : "#cbd5e1"} />
            </button>
          ))}
        </div>
      </div>
      <div style={{ display: "flex", flexDirection: "column", gap: "4px" }}>
        <label style={{ fontSize: "13px", fontWeight: 600 }}>評価コメント:</label>
        <textarea value={comment} onChange={(e) => setComment(e.target.value)} style={{ width: "100%", height: "60px", padding: "8px", borderRadius: "6px", border: "1px solid #cbd5e1", fontSize: "13px", fontFamily: "sans-serif" }} required disabled={message.startsWith("🎉")} />
      </div>
      <button className="primary-button" type="submit" disabled={saving || message.startsWith("🎉")} style={{ alignSelf: "start", background: message.startsWith("🎉") ? "#cbd5e1" : "#10b981", color: "#fff", border: "none", padding: "8px 16px", fontSize: "13px", cursor: message.startsWith("🎉") ? "not-allowed" : "pointer", borderRadius: "6px" }}>
        {saving ? "投稿中..." : message.startsWith("🎉") ? "✓ 評価送信済み" : "評価を送信して取引を完了する"}
      </button>
      {message && <p className={message.includes("失敗") ? "error" : "notice inline-notice"} style={{ margin: 0, fontSize: "13px", color: message.includes("失敗") ? "#ef4444" : "#047857" }}>{message}</p>}
    </form>
  );
}
