import axios from "axios";
const apiUrl = import.meta.env.VITE_API_BASE_URL;

// 创建 axios 实例，配置基础 URL 和默认选项
export default axios.create({
  baseURL: apiUrl, // API 基础地址
  withCredentials: true, // ✅ 允许发送 cookies（必须在顶层，不能放在 headers 里）
  headers: {
    "Content-Type": "application/json", // 默认请求内容类型
  },
});
