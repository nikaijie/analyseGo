// 修正：使用 process.env 代替 import.meta.env
export const API_BASE = (import.meta.env.VITE_API_URL as string) || 'http://localhost:8099'
