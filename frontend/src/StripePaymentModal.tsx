import React, { useState, useEffect, useRef } from "react";

export function StripePaymentModal({
  price,
  title,
  onClose,
  onSuccess
}: {
  price: number;
  title: string;
  onClose: () => void;
  onSuccess: () => Promise<void>;
}) {
  const [cardNumber, setCardNumber] = useState("");
  const [expiry, setExpiry] = useState("");
  const [cvc, setCvc] = useState("");
  const [name, setName] = useState("");
  const [paying, setPaying] = useState(false);
  const [success, setSuccess] = useState(false);

  const timerRef1 = useRef<any>(null);
  const timerRef2 = useRef<any>(null);

  useEffect(() => {
    return () => {
      if (timerRef1.current) clearTimeout(timerRef1.current);
      if (timerRef2.current) clearTimeout(timerRef2.current);
    };
  }, []);

  // Format card number with spaces
  const handleCardNumberChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    let value = e.target.value.replace(/\s+/g, "").replace(/[^0-9]/gi, "");
    let matches = value.match(/\d{4,16}/g);
    let match = (matches && matches[0]) || "";
    let parts = [];

    for (let i = 0, len = match.length; i < len; i += 4) {
      parts.push(match.substring(i, i + 4));
    }

    if (parts.length > 0) {
      setCardNumber(parts.join(" "));
    } else {
      setCardNumber(value);
    }
  };

  const handleExpiryChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    let value = e.target.value.replace(/[^0-9]/g, "");
    if (value.length > 2) {
      setExpiry(value.substring(0, 2) + "/" + value.substring(2, 4));
    } else {
      setExpiry(value);
    }
  };

  const handlePay = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!cardNumber || !expiry || !cvc || !name) {
      alert("すべてのクレジットカード情報を入力してください。");
      return;
    }
    setPaying(true);
    // Simulate secure network tokenization roundtrip (1.8 seconds)
    timerRef1.current = setTimeout(() => {
      setPaying(false);
      setSuccess(true);
      // Wait for success checkmark animation (1.2 seconds)
      timerRef2.current = setTimeout(() => {
        onSuccess();
      }, 1200);
    }, 1800);
  };

  const isVisa = cardNumber.startsWith("4");
  const isMaster = cardNumber.startsWith("5");

  return (
    <div style={{ position: "fixed", top: 0, left: 0, right: 0, bottom: 0, background: "rgba(15, 23, 42, 0.65)", backdropFilter: "blur(4px)", display: "flex", justifyContent: "center", alignItems: "center", zIndex: 1100, padding: "20px" }}>
      <div style={{ background: "#ffffff", borderRadius: "16px", border: "1px solid #cbd5e1", padding: "32px", width: "100%", maxWidth: "480px", boxShadow: "0 25px 50px -12px rgba(0,0,0,0.25)", position: "relative", display: "flex", flexDirection: "column", gap: "24px", color: "#1f2937" }}>
        
        {/* Header */}
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", borderBottom: "1px solid #f1f5f9", paddingBottom: "16px" }}>
          <div>
            <h3 style={{ margin: 0, fontSize: "18px", color: "#0f172a", fontWeight: 700 }}>💳 Stripe 安全クレジットカード決済</h3>
            <small style={{ color: "#64748b" }}>購入商品: {title}</small>
          </div>
          <button onClick={onClose} style={{ background: "none", border: "none", fontSize: "20px", color: "#94a3b8", cursor: "pointer" }} disabled={paying || success}>✕</button>
        </div>

        {success ? (
          /* Success Screen */
          <div style={{ display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", padding: "40px 0", gap: "16px", textAlign: "center" }}>
            <div style={{ width: "64px", height: "64px", borderRadius: "50%", background: "#ecfdf5", border: "2px solid #34d399", display: "flex", alignItems: "center", justifyContent: "center", color: "#34d399", fontSize: "32px", animation: "sparkleGlow 1s infinite alternate" }}>✓</div>
            <strong style={{ fontSize: "20px", color: "#065f46" }}>お支払いが完了しました！</strong>
            <p style={{ margin: 0, color: "#047857", fontSize: "14px" }}>
              Stripeセキュリティ保護が適用されました。<br />
              代金は取引完了までエスクローに安全に保護されます。
            </p>
          </div>
        ) : paying ? (
          /* Loading Screen */
          <div style={{ display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", padding: "40px 0", gap: "16px", textAlign: "center" }}>
            <div className="updating-spinner" style={{ fontSize: "40px" }}>⏳</div>
            <strong style={{ fontSize: "16px", color: "#334155" }}>Stripe Secure Gateway で決済を処理中...</strong>
            <p style={{ margin: 0, color: "#64748b", fontSize: "13px" }}>
              カード番号の安全トークン化および与信審査を行っています。<br />
              画面を閉じずにそのままお待ちください。
            </p>
          </div>
        ) : (
          /* Input Form */
          <form onSubmit={handlePay} style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
            
            {/* Live Card Preview */}
            <div style={{ 
              background: isVisa ? "linear-gradient(135deg, #1e3a8a, #3b82f6)" : isMaster ? "linear-gradient(135deg, #374151, #111827)" : "linear-gradient(135deg, #475569, #1e293b)",
              borderRadius: "12px", 
              padding: "20px", 
              color: "#ffffff", 
              display: "flex", 
              flexDirection: "column", 
              justifyContent: "space-between", 
              height: "160px",
              boxShadow: "0 8px 16px rgba(0,0,0,0.15)",
              transition: "all 0.3s ease"
            }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "start" }}>
                <span style={{ fontSize: "12px", fontWeight: 700, letterSpacing: "1px", opacity: 0.8 }}>SECURE CARD</span>
                {isVisa && <strong style={{ fontSize: "18px", fontStyle: "italic" }}>VISA</strong>}
                {isMaster && <strong style={{ fontSize: "18px", fontStyle: "italic" }}>Mastercard</strong>}
                {!isVisa && !isMaster && <strong style={{ fontSize: "18px", fontStyle: "italic", opacity: 0.5 }}>CARD</strong>}
              </div>
              <strong style={{ fontSize: "20px", fontFamily: "monospace", letterSpacing: "2px", margin: "12px 0" }}>
                {cardNumber || "•••• •••• •••• ••••"}
              </strong>
              <div style={{ display: "flex", justifyContent: "space-between" }}>
                <div>
                  <small style={{ display: "block", fontSize: "9px", opacity: 0.7 }}>CARDHOLDER</small>
                  <span style={{ fontSize: "13px", fontWeight: 600, textTransform: "uppercase" }}>{name || "YOUR NAME"}</span>
                </div>
                <div>
                  <small style={{ display: "block", fontSize: "9px", opacity: 0.7 }}>EXPIRES</small>
                  <span style={{ fontSize: "13px", fontFamily: "monospace" }}>{expiry || "MM/YY"}</span>
                </div>
              </div>
            </div>

            {/* Inputs */}
            <div style={{ display: "flex", flexDirection: "column", gap: "12px" }}>
              <div style={{ display: "flex", flexDirection: "column", gap: "4px" }}>
                <label style={{ fontSize: "12px", fontWeight: 600, color: "#475569" }}>カード番号</label>
                <input
                  type="text"
                  maxLength={19}
                  placeholder="4000 1234 5678 9010"
                  value={cardNumber}
                  onChange={handleCardNumberChange}
                  style={{ width: "100%", padding: "10px", borderRadius: "8px", border: "1px solid #cbd5e1" }}
                  required
                />
              </div>

              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "12px" }}>
                <div style={{ display: "flex", flexDirection: "column", gap: "4px" }}>
                  <label style={{ fontSize: "12px", fontWeight: 600, color: "#475569" }}>有効期限</label>
                  <input
                    type="text"
                    maxLength={5}
                    placeholder="MM/YY"
                    value={expiry}
                    onChange={handleExpiryChange}
                    style={{ width: "100%", padding: "10px", borderRadius: "8px", border: "1px solid #cbd5e1" }}
                    required
                  />
                </div>
                <div style={{ display: "flex", flexDirection: "column", gap: "4px" }}>
                  <label style={{ fontSize: "12px", fontWeight: 600, color: "#475569" }}>CVC (セキュリティコード)</label>
                  <input
                    type="password"
                    maxLength={4}
                    placeholder="•••"
                    value={cvc}
                    onChange={(e) => setCvc(e.target.value.replace(/[^0-9]/g, ""))}
                    style={{ width: "100%", padding: "10px", borderRadius: "8px", border: "1px solid #cbd5e1" }}
                    required
                  />
                </div>
              </div>

              <div style={{ display: "flex", flexDirection: "column", gap: "4px" }}>
                <label style={{ fontSize: "12px", fontWeight: 600, color: "#475569" }}>カード名義人 (ローマ字)</label>
                <input
                  type="text"
                  placeholder="TARO YAMADA"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  style={{ width: "100%", padding: "10px", borderRadius: "8px", border: "1px solid #cbd5e1", textTransform: "uppercase" }}
                  required
                />
              </div>
            </div>

            {/* Total Indicator & Submit */}
            <div style={{ borderTop: "1px solid #f1f5f9", paddingTop: "16px", marginTop: "8px", display: "flex", flexDirection: "column", gap: "12px" }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                <span style={{ fontSize: "14px", color: "#475569" }}>支払合計額 (エスクロー保護)</span>
                <strong style={{ fontSize: "20px", color: "#0f172a" }}>¥{price.toLocaleString()}</strong>
              </div>
              <button
                type="submit"
                className="primary-button"
                style={{ width: "100%", background: "#34d399", color: "#ffffff", border: "none", padding: "14px", fontSize: "16px" }}
              >
                🔐 安全な決済を実行（Stripe認証）
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}
