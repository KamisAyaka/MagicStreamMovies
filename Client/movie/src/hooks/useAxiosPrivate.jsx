import { useEffect } from "react";
import axios from "axios";

import useAuth from "./useAuth";

const apiUrl = import.meta.env.VITE_API_BASE_URL;

/**
 * useAxiosPrivate - 带有自动 token 刷新功能的 axios 实例
 *
 * 这个 Hook 用于发送需要认证的 API 请求，主要功能：
 * 1. 自动携带 HttpOnly Cookie（包含 access_token 和 refresh_token）
 * 2. 自动检测 401 错误（token 过期）
 * 3. 自动刷新 token 并重试失败的请求
 * 4. 处理并发请求时的 token 刷新（避免重复刷新）
 *
 * 使用场景：
 * - 获取需要登录才能访问的数据（如推荐电影、个人信息）
 * - 执行需要权限的操作（如发表评论、修改资料）
 *
 *
 * 工作流程：
 * 1. 用户登录 → 后端设置 HttpOnly Cookie（access_token 24小时，refresh_token 7天）
 * 2. 发送请求 → 自动携带 Cookie
 * 3. Token 过期 → 拦截器检测到 401 错误
 * 4. 自动调用 /refresh 接口 → 后端验证 refresh_token 并颁发新的 access_token
 * 5. 重试原始请求 → 用户无感知，请求成功
 * 6. Refresh token 也过期 → 清除登录状态，跳转到登录页
 */
const useAxiosPrivate = () => {
  // 创建 axios 实例，配置基础 URL 和凭证
  const axiosAuth = axios.create({
    baseURL: apiUrl,
    withCredentials: true, // ⭐ 关键：自动发送 HttpOnly Cookie
  });

  const { auth, setAuth } = useAuth();

  // 刷新状态标志：防止多个请求同时刷新 token
  let isRefreshing = false;

  // 失败请求队列：当正在刷新 token 时，暂存其他失败的请求
  let failedQueue = [];

  /**
   * 处理队列中的所有请求
   * @param {Error} error - 如果刷新失败，传入错误对象
   * @param {Object} response - 如果刷新成功，传入响应对象
   */
  const processQueue = (error, response = null) => {
    failedQueue.forEach((prom) => {
      if (error) {
        prom.reject(error); // 刷新失败 → 拒绝所有排队的请求
      } else {
        prom.resolve(response); // 刷新成功 → 解决所有排队的请求
      }
    });

    failedQueue = []; // 清空队列
  };

  useEffect(() => {
    // 添加响应拦截器：自动处理 token 过期的情况
    axiosAuth.interceptors.response.use(
      // 成功响应：直接返回
      (response) => response,

      // 错误响应：检查是否需要刷新 token
      async (error) => {
        console.log("⚠ 拦截器捕获到错误:", error);
        const originalRequest = error.config; // 保存原始请求配置

        // === 特殊情况 1：刷新 token 的请求本身失败 ===
        // 如果刷新接口返回 401，说明 refresh_token 也过期了
        if (
          originalRequest.url.includes("/refresh") &&
          error.response.status === 401
        ) {
          console.error("❌ Refresh token 已过期或无效，需要重新登录");
          return Promise.reject(error); // 直接失败，不重试
        }

        // === 主要逻辑：处理 401 未授权错误 ===
        if (
          error.response &&
          error.response.status === 401 &&
          !originalRequest._retry // 防止无限循环重试
        ) {
          // 情况 A：已经有其他请求在刷新 token
          if (isRefreshing) {
            // 将当前请求加入队列，等待刷新完成
            return new Promise((resolve, reject) => {
              failedQueue.push({ resolve, reject });
            })
              .then(() => axiosAuth(originalRequest)) // 刷新成功后重试原请求
              .catch((err) => Promise.reject(err)); // 刷新失败则拒绝
          }

          // 情况 B：第一个检测到 token 过期的请求
          originalRequest._retry = true; // 标记为已重试，避免死循环
          isRefreshing = true; // 设置刷新状态

          return new Promise((resolve, reject) => {
            // 调用后端的 /refresh 接口刷新 token
            axiosAuth
              .post("/refresh")
              .then(() => {
                // ✅ 刷新成功
                console.log("✅ Token 刷新成功");
                processQueue(null); // 处理队列中的所有请求（让它们重试）

                // 重试当前的原始请求
                axiosAuth(originalRequest).then(resolve).catch(reject);
              })
              .catch((refreshError) => {
                // ❌ 刷新失败（refresh_token 也过期了）
                console.error("❌ Token 刷新失败，清除认证状态");
                processQueue(refreshError, null); // 拒绝队列中的所有请求

                // 清除本地认证状态，用户需要重新登录
                localStorage.removeItem("user");
                setAuth(null);
                reject(refreshError); // 拒绝当前请求
              })
              .finally(() => {
                isRefreshing = false; // 重置刷新状态
              });
          });
        }

        // 其他错误：直接返回
        return Promise.reject(error);
      }
    );
  }, [auth]); // 当 auth 状态变化时重新设置拦截器

  return axiosAuth; // 返回配置好的 axios 实例
};

export default useAxiosPrivate;
