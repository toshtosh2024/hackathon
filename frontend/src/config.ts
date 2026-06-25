const rawApiBaseUrl = import.meta.env.VITE_API_BASE_URL?.trim();

export const API_BASE = rawApiBaseUrl ? rawApiBaseUrl.replace(/\/$/, "") : "/api";
