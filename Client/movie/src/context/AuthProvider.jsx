import { createContext, useState, useEffect } from "react";

// 创建认证上下文，用于在整个应用中共享用户认证状态
const AuthContext = createContext({});

/**
 * 认证提供者组件
 * 负责管理用户认证状态，包括登录、登出和持久化存储
 * @param {Object} props - 组件属性
 * @param {React.ReactNode} props.children - 子组件
 */
export const AuthProvider = ({ children }) => {
  // auth: 存储当前登录用户的信息（如用户名、token等）
  const [auth, setAuth] = useState();

  // loading: 标记是否正在从 localStorage 加载用户信息
  const [loading, setLoading] = useState(true);

  // 组件挂载时执行：从 localStorage 恢复用户登录状态
  useEffect(() => {
    try {
      // 尝试从本地存储获取用户信息
      const storedUser = localStorage.getItem("user");

      if (storedUser) {
        // 解析 JSON 字符串并设置到 auth 状态
        const parsedUser = JSON.parse(storedUser);
        setAuth(parsedUser);
      }
    } catch (error) {
      // 如果解析失败（比如数据损坏），打印错误信息
      console.error("Failed to parse user from localStorage", error);
    } finally {
      // 无论成功或失败，都标记加载完成
      setLoading(false);
    }
  }, []); // 空依赖数组表示只在组件挂载时执行一次

  // 监听 auth 状态变化：自动同步到 localStorage
  useEffect(() => {
    if (auth) {
      // 用户已登录：保存用户信息到本地存储
      localStorage.setItem("user", JSON.stringify(auth));
    } else {
      // 用户已登出：清除本地存储中的用户信息
      localStorage.removeItem("user");
    }
  }, [auth]); // 当 auth 改变时执行

  // 提供认证上下文给子组件
  return (
    <AuthContext.Provider value={{ auth, setAuth, loading }}>
      {children}
    </AuthContext.Provider>
  );
};

// 导出上下文对象，供其他组件使用（通过 useContext 或自定义 hook）
export default AuthContext;
