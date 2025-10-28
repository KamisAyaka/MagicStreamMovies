package utils

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/KamisAyaka/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// SignedDetails 结构体定义了 JWT Token 中包含的用户信息
// JWT (JSON Web Token) 是一种用于安全传输信息的开放标准
// 它包含三部分：Header.Payload.Signature
type SignedDetails struct {
	Email                string // 用户邮箱
	FirstName            string // 用户名字
	LastName             string // 用户姓氏
	Role                 string // 用户角色 (ADMIN/USER)
	UserID               string // 用户唯一标识符
	jwt.RegisteredClaims        // JWT 标准声明，包含过期时间、签发者等信息
}

// 从环境变量获取 JWT 签名密钥
// 这些密钥用于签名和验证 JWT Token，确保 Token 的安全性
var SECRET_KEY string = os.Getenv("SECRET_KEY")                 // 访问令牌签名密钥
var SECRET_REFRESH_KEY string = os.Getenv("SECRET_REFRESH_KEY") // 刷新令牌签名密钥

// GenerateAllTokens 生成访问令牌和刷新令牌
// 访问令牌：用于 API 请求的身份验证，有效期较短
// 刷新令牌：用于获取新的访问令牌，有效期较长
func GenerateAllTokens(email, firstName, lastName, role, userId string) (signedToken, signedRefreshToken string, err error) {
	// 创建访问令牌的声明 (Claims)
	// 声明包含用户信息和标准 JWT 字段
	claims := &SignedDetails{
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Role:      role,
		UserID:    userId,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "MagicStream",                                      // 签发者
			IssuedAt:  jwt.NewNumericDate(time.Now()),                     // 签发时间
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)), // 过期时间：24小时后
		},
	}

	// 使用 HS256 算法创建 JWT Token
	// HS256 是一种对称加密算法，使用密钥签名
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 使用密钥签名 Token，生成最终的访问令牌
	signedToken, err = token.SignedString([]byte(SECRET_KEY))
	if err != nil {
		return "", "", err
	}

	// 创建刷新令牌的声明
	// 刷新令牌通常包含相同信息但有不同的过期时间
	refreshClaims := &SignedDetails{
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Role:      role,
		UserID:    userId,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "MagicStream",                                          // 签发者
			IssuedAt:  jwt.NewNumericDate(time.Now()),                         // 签发时间
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 7)), // 过期时间：7天后
		},
	}

	// 创建刷新令牌
	refreshtoken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)

	// 使用密钥签名刷新令牌
	signedRefreshToken, err = refreshtoken.SignedString([]byte(SECRET_REFRESH_KEY))
	if err != nil {
		return "", "", err
	}

	// 返回生成的访问令牌和刷新令牌
	return signedToken, signedRefreshToken, nil
}

// UpdateAllTokens 更新用户数据库中的令牌信息
// 当用户登录或刷新令牌时，需要将新的令牌保存到数据库中
func UpdateAllTokens(userId, token, refreshToken string, client *mongo.Client) (err error) {
	// 创建带超时的上下文，防止数据库操作超时
	var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel() // 确保资源被正确释放

	// 格式化当前时间，用于更新 updated_at 字段
	updateAt, _ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

	// 准备更新数据
	// $set 操作符用于更新指定字段
	updateData := bson.M{"$set": bson.M{
		"token":         token,        // 新的访问令牌
		"refresh_token": refreshToken, // 新的刷新令牌
		"updated_at":    updateAt,     // 更新时间
	}}
	var userCollection *mongo.Collection = database.OpenCollection("users", client) // 用户集合

	// 根据用户ID更新用户文档中的令牌信息
	_, err = userCollection.UpdateOne(ctx, bson.M{"user_id": userId}, updateData)
	if err != nil {
		return err
	}
	return nil
}

// GetAccessToken 从 HTTP 请求头中提取 JWT 访问令牌
// 这个函数用于从标准的 Authorization 头中安全地提取 Bearer Token
// 格式：Authorization: Bearer <JWT_TOKEN>
func GetAccessToken(c *gin.Context) (string, error) {
	// 从请求头中获取 Authorization 字段
	// 这是 JWT 令牌的标准传输方式
	// authHeader := c.Request.Header.Get("Authorization")

	// // 检查是否存在 Authorization 头
	// // 如果没有，说明请求未携带认证信息
	// if authHeader == "" {
	// 	return "", errors.New("authorization header is required")
	// }

	// // 从 "Bearer <token>" 格式中提取实际的 JWT 令牌
	// // 去掉 "Bearer " 前缀（长度为 7 个字符）
	// tokenString := authHeader[len("Bearer "):]

	// // 验证提取的令牌是否为空
	// // 防止 "Bearer " 后面没有实际令牌的情况
	// if tokenString == "" {
	// 	return "", errors.New("bearer token is required")
	// }
	tokenString, err := c.Cookie("access_token")
	if err != nil {
		return "", err
	}

	// 返回提取的 JWT 令牌字符串
	return tokenString, nil
}

// ValidateToken 验证 JWT 令牌的有效性
// 这个函数用于验证从请求中提取的 JWT 令牌是否有效、未过期且未被篡改
// 返回解析后的用户声明信息，如果验证失败则返回错误
func ValidateToken(tokenString string) (*SignedDetails, error) {
	// 创建一个空的 SignedDetails 结构体用于存储解析后的声明信息
	claims := &SignedDetails{}

	// 解析 JWT 令牌并验证签名
	// ParseWithClaims 会验证令牌的格式、签名和有效性
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// 返回用于验证签名的密钥
		// 这个密钥必须与生成令牌时使用的密钥相同
		return []byte(SECRET_KEY), nil
	})

	// 检查解析过程中是否出现错误
	// 可能的错误：令牌格式错误、签名验证失败等
	if err != nil {
		return nil, err
	}

	// 验证令牌使用的签名算法是否为 HMAC
	// 这确保令牌使用的是我们期望的签名方法（HS256）
	// 防止算法替换攻击（Algorithm Confusion Attack）
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, err
	}

	// 检查令牌是否已过期
	// 比较令牌的过期时间与当前时间
	// 如果令牌已过期，则拒绝访问
	if claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("token expired")
	}

	// 如果所有验证都通过，返回解析后的用户声明信息
	// 这些信息包含用户ID、邮箱、角色等，可用于后续的授权判断
	return claims, nil
}

func GetUserIdFromContext(c *gin.Context) (string, error) {
	userId, exists := c.Get("userID")
	if !exists {
		return "", errors.New("user ID not found in context")
	}
	id, ok := userId.(string)
	if !ok {
		return "", errors.New("user ID is not a string")
	}
	return id, nil
}

func GetRoleFromContext(c *gin.Context) (string, error) {
	role, exists := c.Get("role")
	if !exists {
		return "", errors.New("role ID not found in context")
	}
	memberRole, ok := role.(string)
	if !ok {
		return "", errors.New("role ID is not a string")
	}
	return memberRole, nil
}
func ValidateRefreshToken(tokenString string) (*SignedDetails, error) {
	claims := &SignedDetails{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {

		return []byte(SECRET_REFRESH_KEY), nil
	})

	if err != nil {
		return nil, err
	}

	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, err
	}

	if claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("refresh token has expired")
	}

	return claims, nil
}
