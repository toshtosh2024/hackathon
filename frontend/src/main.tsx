import { FormEvent, useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import {
  Bot,
  Heart,
  LogIn,
  MessageCircle,
  PackagePlus,
  Send,
  ShoppingBag,
  Sparkles,
  WalletCards
} from "lucide-react";
import "./styles.css";

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080/api";

type User = {
  id: number;
  name: string;
  email: string;
  role: string;
};

type Item = {
  id: number;
  sellerId: number;
  sellerName: string;
  title: string;
  description: string;
  category: string;
  price: number;
  status: "active" | "sold" | "hidden";
  imageUrl: string;
  likeCount: number;
  createdAt: string;
};

type Conversation = {
  id: number;
  itemId: number;
  itemTitle: string;
  buyerId: number;
  sellerId: number;
  updatedAt: string;
};

type Message = {
  id: number;
  conversationId: number;
  senderId: number;
  body: string;
  createdAt: string;
};

function App() {
  const [token, setToken] = useState(localStorage.getItem("token") ?? "");
  const [user, setUser] = useState<User | null>(loadUser());
  const [items, setItems] = useState<Item[]>([]);
  const [selectedItem, setSelectedItem] = useState<Item | null>(null);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [selectedConversation, setSelectedConversation] = useState<Conversation | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [notice, setNotice] = useState("");

  useEffect(() => {
    void loadItems();
  }, []);

  useEffect(() => {
    if (token) {
      void loadConversations();
    }
  }, [token]);

  async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
    const headers = new Headers(options.headers);
    headers.set("Content-Type", "application/json");
    if (token) headers.set("Authorization", `Bearer ${token}`);
    const response = await fetch(`${API_BASE}${path}`, { ...options, headers });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error ?? "API error");
    return data;
  }

  async function loadItems() {
    const data = await api<{ items: Item[] }>("/items");
    setItems(data.items);
    setSelectedItem((current) => current ?? data.items[0] ?? null);
  }

  async function loadConversations() {
    const data = await api<{ conversations: Conversation[] }>("/conversations");
    setConversations(data.conversations);
  }

  async function loadMessages(conversation: Conversation) {
    setSelectedConversation(conversation);
    const data = await api<{ messages: Message[] }>(`/conversations/${conversation.id}/messages`);
    setMessages(data.messages);
  }

  function saveSession(nextToken: string, nextUser: User) {
    setToken(nextToken);
    setUser(nextUser);
    localStorage.setItem("token", nextToken);
    localStorage.setItem("user", JSON.stringify(nextUser));
  }

  return (
    <main className="app-shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">Next Market</p>
          <h1>AIが出品と購入を支えるフリマ</h1>
        </div>
        <div className="session">
          <span>{user ? `${user.name} でログイン中` : "未ログイン"}</span>
          {user && (
            <button
              className="ghost-button"
              onClick={() => {
                localStorage.clear();
                setToken("");
                setUser(null);
                setNotice("ログアウトしました");
              }}
            >
              ログアウト
            </button>
          )}
        </div>
      </header>

      {notice && <p className="notice">{notice}</p>}

      <section className="workspace">
        <AuthPanel onAuth={saveSession} api={api} />
        <CreateItemPanel
          disabled={!user}
          api={api}
          onCreated={(item) => {
            setItems([item, ...items]);
            setSelectedItem(item);
            setNotice("商品を出品しました");
          }}
        />
        <ItemList items={items} selectedItem={selectedItem} onSelect={setSelectedItem} />
        <ItemDetail
          item={selectedItem}
          user={user}
          api={api}
          onChanged={loadItems}
          onNotice={setNotice}
          onConversationCreated={loadConversations}
        />
        <MessagesPanel
          user={user}
          conversations={conversations}
          selectedConversation={selectedConversation}
          messages={messages}
          api={api}
          onSelect={loadMessages}
          onSent={(conversation) => void loadMessages(conversation)}
        />
      </section>
    </main>
  );
}

function AuthPanel({
  api,
  onAuth
}: {
  api: <T>(path: string, options?: RequestInit) => Promise<T>;
  onAuth: (token: string, user: User) => void;
}) {
  const [mode, setMode] = useState<"login" | "register">("register");
  const [name, setName] = useState("Toshi");
  const [email, setEmail] = useState("toshi@example.com");
  const [password, setPassword] = useState("password");
  const [error, setError] = useState("");

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    try {
      const data = await api<{ token: string; user: User }>(`/auth/${mode}`, {
        method: "POST",
        body: JSON.stringify(mode === "register" ? { name, email, password } : { email, password })
      });
      onAuth(data.token, data.user);
    } catch (err) {
      setError(err instanceof Error ? err.message : "認証に失敗しました");
    }
  }

  return (
    <section className="panel auth-panel">
      <div className="panel-heading">
        <LogIn size={20} />
        <h2>認証</h2>
      </div>
      <div className="segmented">
        <button className={mode === "register" ? "active" : ""} onClick={() => setMode("register")}>
          新規登録
        </button>
        <button className={mode === "login" ? "active" : ""} onClick={() => setMode("login")}>
          ログイン
        </button>
      </div>
      <form onSubmit={submit}>
        {mode === "register" && <input value={name} onChange={(e) => setName(e.target.value)} placeholder="名前" />}
        <input value={email} onChange={(e) => setEmail(e.target.value)} placeholder="メール" />
        <input value={password} onChange={(e) => setPassword(e.target.value)} placeholder="パスワード" type="password" />
        <button className="primary-button" type="submit">
          <LogIn size={18} />
          {mode === "register" ? "登録" : "ログイン"}
        </button>
      </form>
      {error && <p className="error">{error}</p>}
    </section>
  );
}

function CreateItemPanel({
  disabled,
  api,
  onCreated
}: {
  disabled: boolean;
  api: <T>(path: string, options?: RequestInit) => Promise<T>;
  onCreated: (item: Item) => void;
}) {
  const [title, setTitle] = useState("撥水ミニショルダーバッグ");
  const [category, setCategory] = useState("fashion");
  const [condition, setCondition] = useState("数回使用、美品");
  const [notes, setNotes] = useState("軽い。内ポケットあり。通勤にも旅行にも使える。");
  const [description, setDescription] = useState("");
  const [price, setPrice] = useState(4800);
  const [imageUrl, setImageUrl] = useState("https://images.unsplash.com/photo-1594223274512-ad4803739b7c?auto=format&fit=crop&w=900&q=80");
  const [loadingAI, setLoadingAI] = useState(false);

  async function generateDescription() {
    setLoadingAI(true);
    try {
      const data = await api<{ description: string }>("/ai/generate-description", {
        method: "POST",
        body: JSON.stringify({ title, category, condition, notes })
      });
      setDescription(data.description);
    } finally {
      setLoadingAI(false);
    }
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    const data = await api<{ item: Item }>("/items", {
      method: "POST",
      body: JSON.stringify({ title, category, description, price, imageUrl })
    });
    onCreated(data.item);
  }

  return (
    <section className="panel create-panel">
      <div className="panel-heading">
        <PackagePlus size={20} />
        <h2>出品</h2>
      </div>
      <form onSubmit={submit}>
        <input disabled={disabled} value={title} onChange={(e) => setTitle(e.target.value)} placeholder="商品名" />
        <div className="two-col">
          <input disabled={disabled} value={category} onChange={(e) => setCategory(e.target.value)} placeholder="カテゴリ" />
          <input disabled={disabled} value={price} onChange={(e) => setPrice(Number(e.target.value))} type="number" placeholder="価格" />
        </div>
        <input disabled={disabled} value={condition} onChange={(e) => setCondition(e.target.value)} placeholder="状態" />
        <textarea disabled={disabled} value={notes} onChange={(e) => setNotes(e.target.value)} placeholder="AIに渡すメモ" />
        <button className="ai-button" disabled={disabled || loadingAI} type="button" onClick={generateDescription}>
          <Sparkles size={18} />
          {loadingAI ? "生成中" : "Geminiで説明生成"}
        </button>
        <textarea disabled={disabled} value={description} onChange={(e) => setDescription(e.target.value)} placeholder="商品説明" />
        <input disabled={disabled} value={imageUrl} onChange={(e) => setImageUrl(e.target.value)} placeholder="画像URL" />
        <button className="primary-button" disabled={disabled || !description} type="submit">
          <ShoppingBag size={18} />
          出品する
        </button>
      </form>
    </section>
  );
}

function ItemList({
  items,
  selectedItem,
  onSelect
}: {
  items: Item[];
  selectedItem: Item | null;
  onSelect: (item: Item) => void;
}) {
  return (
    <section className="panel list-panel">
      <div className="panel-heading">
        <ShoppingBag size={20} />
        <h2>商品一覧</h2>
      </div>
      <div className="item-list">
        {items.map((item) => (
          <button key={item.id} className={selectedItem?.id === item.id ? "item-row active" : "item-row"} onClick={() => onSelect(item)}>
            <img src={item.imageUrl || "/placeholder.svg"} alt="" />
            <span>
              <strong>{item.title}</strong>
              <small>
                ¥{item.price.toLocaleString()} / {item.status}
              </small>
            </span>
          </button>
        ))}
        {items.length === 0 && <p className="muted">まだ商品がありません。</p>}
      </div>
    </section>
  );
}

function ItemDetail({
  item,
  user,
  api,
  onChanged,
  onNotice,
  onConversationCreated
}: {
  item: Item | null;
  user: User | null;
  api: <T>(path: string, options?: RequestInit) => Promise<T>;
  onChanged: () => Promise<void>;
  onNotice: (message: string) => void;
  onConversationCreated: () => Promise<void>;
}) {
  const [question, setQuestion] = useState("通勤用として雨の日にも使えそう？");
  const [answer, setAnswer] = useState("");

  if (!item) {
    return (
      <section className="panel detail-panel">
        <p className="muted">商品を選択してください。</p>
      </section>
    );
  }
  const currentItem = item;

  async function askAI() {
    const data = await api<{ answer: string }>("/ai/ask", {
      method: "POST",
      body: JSON.stringify({ itemId: currentItem.id, question })
    });
    setAnswer(data.answer);
  }

  async function like() {
    await api(`/items/${currentItem.id}/like`, { method: "POST" });
    await onChanged();
  }

  async function purchase() {
    await api(`/items/${currentItem.id}/purchase`, { method: "POST" });
    await onChanged();
    onNotice("購入が完了しました");
  }

  async function messageSeller() {
    const data = await api<{ conversationId: number }>("/conversations", {
      method: "POST",
      body: JSON.stringify({ itemId: currentItem.id, sellerId: currentItem.sellerId })
    });
    await onConversationCreated();
    onNotice(`会話を作成しました: #${data.conversationId}`);
  }

  return (
    <section className="panel detail-panel">
      <img className="hero-image" src={currentItem.imageUrl || "/placeholder.svg"} alt="" />
      <div className="detail-title">
        <div>
          <p className="eyebrow">{currentItem.category}</p>
          <h2>{currentItem.title}</h2>
        </div>
        <strong>¥{currentItem.price.toLocaleString()}</strong>
      </div>
      <p>{currentItem.description}</p>
      <div className="detail-actions">
        <button disabled={!user} onClick={like}>
          <Heart size={18} />
          {currentItem.likeCount}
        </button>
        <button disabled={!user || currentItem.status !== "active"} onClick={messageSeller}>
          <MessageCircle size={18} />
          DM
        </button>
        <button disabled={!user || currentItem.status !== "active"} onClick={purchase}>
          <WalletCards size={18} />
          購入
        </button>
      </div>
      <div className="ai-ask">
        <textarea disabled={!user} value={question} onChange={(e) => setQuestion(e.target.value)} />
        <button className="ai-button" disabled={!user} onClick={askAI}>
          <Bot size={18} />
          AIに質問
        </button>
        {answer && <p className="ai-answer">{answer}</p>}
      </div>
    </section>
  );
}

function MessagesPanel({
  user,
  conversations,
  selectedConversation,
  messages,
  api,
  onSelect,
  onSent
}: {
  user: User | null;
  conversations: Conversation[];
  selectedConversation: Conversation | null;
  messages: Message[];
  api: <T>(path: string, options?: RequestInit) => Promise<T>;
  onSelect: (conversation: Conversation) => Promise<void>;
  onSent: (conversation: Conversation) => void;
}) {
  const [body, setBody] = useState("購入前に状態をもう少し教えてください。");

  async function send(event: FormEvent) {
    event.preventDefault();
    if (!selectedConversation) return;
    await api(`/conversations/${selectedConversation.id}/messages`, {
      method: "POST",
      body: JSON.stringify({ body })
    });
    setBody("");
    onSent(selectedConversation);
  }

  return (
    <section className="panel messages-panel">
      <div className="panel-heading">
        <MessageCircle size={20} />
        <h2>DM</h2>
      </div>
      <div className="conversation-list">
        {conversations.map((conversation) => (
          <button
            key={conversation.id}
            className={selectedConversation?.id === conversation.id ? "conversation active" : "conversation"}
            onClick={() => void onSelect(conversation)}
          >
            {conversation.itemTitle}
          </button>
        ))}
        {conversations.length === 0 && <p className="muted">会話はまだありません。</p>}
      </div>
      <div className="message-list">
        {messages.map((message) => (
          <p key={message.id} className={message.senderId === user?.id ? "message mine" : "message"}>
            {message.body}
          </p>
        ))}
      </div>
      <form className="message-form" onSubmit={send}>
        <input disabled={!selectedConversation} value={body} onChange={(e) => setBody(e.target.value)} />
        <button disabled={!selectedConversation}>
          <Send size={18} />
        </button>
      </form>
    </section>
  );
}

function loadUser() {
  const raw = localStorage.getItem("user");
  if (!raw) return null;
  try {
    return JSON.parse(raw) as User;
  } catch {
    return null;
  }
}

createRoot(document.getElementById("root")!).render(<App />);
