import React, { useState } from "react";
import { CreditCard, X, Lock } from "lucide-react";

interface StripePaymentModalProps {
  price: number;
  title: string;
  onClose: () => void;
  onSuccess: () => Promise<void>;
}

export function StripePaymentModal({ price, title, onClose, onSuccess }: StripePaymentModalProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handlePay = async () => {
    setLoading(true);
    setError("");
    try {
      await onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : "決済処理に失敗しました");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{
      position: "fixed", top: 0, left: 0, right: 0, bottom: 0,
      background: "rgba(15, 23, 42, 0.6)", backdropFilter: "blur(8px)",
      display: "flex", justifyContent: "center", alignItems: "center",
      zIndex: 2000, padding: "20px"
    }}>
      <div style={{
        background: "#ffffff", borderRadius: "24px", padding: "28px",
        width: "100%", maxWidth: "420px", boxShadow: "0 24px 48px rgba(0,0,0,0.2)",
        display: "flex", flexDirection: "column", gap: "20px"
      }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
            <CreditCard size={22} style={{ color: "#4F46E5" }} />
            <h3 style={{ margin: 0, fontSize: "18px", fontWeight: 800 }}>Stripeで安全に決済</h3>
          </div>
          <button type="button" onClick={onClose} style={{ background: "rgba(79,70,229,0.08)", borderRadius: "10px", padding: "6px", minHeight: "auto", color: "#475569" }}>
            <X size={18} />
          </button>
        </div>

        <div style={{ background: "rgba(79,70,229,0.06)", borderRadius: "16px", padding: "16px" }}>
          <p style={{ margin: "0 0 4px", fontSize: "13px", color: "#64748B" }}>購入商品</p>
          <p style={{ margin: "0 0 8px", fontWeight: 700, fontSize: "16px" }}>{title}</p>
          <p style={{ margin: 0, fontSize: "24px", fontWeight: 900, color: "#4F46E5", fontFamily: "Outfit, sans-serif" }}>
            ¥{price.toLocaleString()}
          </p>
        </div>

        <div style={{ display: "flex", alignItems: "center", gap: "6px", color: "#64748B", fontSize: "12px" }}>
          <Lock size={13} />
          <span>Stripeエスクロー保護付き — 受取確認後に出品者へ送金されます</span>
        </div>

        {error && <p style={{ margin: 0, color: "#DC2626", fontSize: "13px" }}>{error}</p>}

        <div style={{ display: "flex", gap: "10px" }}>
          <button type="button" onClick={onClose} style={{
            flex: 1, background: "rgba(79,70,229,0.08)", color: "#475569",
            borderRadius: "14px", padding: "14px", fontWeight: 600, minHeight: "48px"
          }}>
            キャンセル
          </button>
          <button type="button" onClick={handlePay} disabled={loading} style={{
            flex: 2, background: "linear-gradient(135deg, #4F46E5, #3730A3)",
            color: "#ffffff", borderRadius: "14px", padding: "14px",
            fontWeight: 700, minHeight: "48px",
            boxShadow: "0 8px 20px rgba(79,70,229,0.3)"
          }}>
            {loading ? "処理中..." : `¥${price.toLocaleString()} を支払う`}
          </button>
        </div>
      </div>
    </div>
  );
}
