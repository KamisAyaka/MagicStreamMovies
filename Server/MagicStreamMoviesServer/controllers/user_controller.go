package controllers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/KamisAyaka/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	"github.com/KamisAyaka/MagicStreamMovies/Server/MagicStreamMoviesServer/models"
	"github.com/KamisAyaka/MagicStreamMovies/Server/MagicStreamMoviesServer/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func RegisterUser(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User

		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
			return
		}
		validate := validator.New()
		if err := validate.Struct(user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
			return
		}

		hashedPassword, err := HashPassword(user.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
			return
		}
		var ctx, cancel = context.WithTimeout(c, 100*time.Second)
		defer cancel()
		var userCollection *mongo.Collection = database.OpenCollection("users", client)
		count, err := userCollection.CountDocuments(ctx, bson.M{"email": user.Email})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing user"})
			return
		}
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
			return
		}
		user.UserID = bson.NewObjectID().Hex()
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
		user.Password = hashedPassword

		result, err := userCollection.InsertOne(ctx, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
		c.JSON(http.StatusCreated, result)
	}
}

func LoginUser(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var userLogin models.UserLogin
		if err := c.ShouldBindJSON(&userLogin); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
			return
		}

		var ctx, cancel = context.WithTimeout(c, 100*time.Second)
		defer cancel()
		var foundUser models.User
		var userCollection *mongo.Collection = database.OpenCollection("users", client)
		err := userCollection.FindOne(ctx, bson.M{"email": userLogin.Email}).Decode(&foundUser)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(userLogin.Password))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
			return
		}

		// 生成 JWT 访问令牌和刷新令牌
		token, refreshToken, err := utils.GenerateAllTokens(foundUser.Email, foundUser.FirstName, foundUser.LastName, foundUser.Role, foundUser.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating tokens"})
			return
		}

		// 注意：使用 HttpOnly Cookie 存储 token，不再将 token 保存到数据库
		// 这样更安全，因为：
		// 1. 减少数据库存储负担
		// 2. token 只在 Cookie 中，后端无状态（stateless）
		// 3. 过期后自动失效，无需手动清理数据库

		// 根据环境配置 Cookie 安全设置
		// 开发环境(HTTP): Secure=false, SameSite=Lax
		// 生产环境(HTTPS): Secure=true, SameSite=None (允许跨域)
		isProduction := os.Getenv("ENV") == "production"
		sameSiteMode := http.SameSiteLaxMode
		secureFlag := false

		if isProduction {
			sameSiteMode = http.SameSiteNoneMode
			secureFlag = true
		}

		// 设置访问令牌 Cookie
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "access_token",
			Value:    token,
			HttpOnly: true,         // 防止 XSS 攻击，JavaScript 无法访问
			Secure:   secureFlag,   // 开发环境: false, 生产环境: true
			MaxAge:   86400,        // 24小时
			SameSite: sameSiteMode, // 开发环境: Lax, 生产环境: None
			Path:     "/",
		})

		// 设置刷新令牌 Cookie
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			HttpOnly: true,         // 防止 XSS 攻击
			Secure:   secureFlag,   // 开发环境: false, 生产环境: true
			MaxAge:   604800,       // 7天
			SameSite: sameSiteMode, // 开发环境: Lax, 生产环境: None
			Path:     "/",
		})
		c.JSON(http.StatusOK, models.UserResponse{
			UserID:    foundUser.UserID,
			FirstName: foundUser.FirstName,
			LastName:  foundUser.LastName,
			Email:     foundUser.Email,
			Role:      foundUser.Role,
			// Token:           token,
			// RefreshToken:    refreshToken,
			FavouriteGenres: foundUser.FavouriteGenres,
		})
	}
}

// LogoutHandler 处理用户登出请求
// 通过删除 HttpOnly Cookie 来清除客户端的认证信息
func LogoutHandler(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 注意：使用 Cookie 方案后，登出只需要删除客户端的 Cookie
		// 不需要从数据库删除 token（因为我们已经不在数据库存储 token 了）
		// Token 过期后会自动失效

		// 根据环境配置 Cookie 设置（与登录时保持一致）
		isProduction := os.Getenv("ENV") == "production"
		sameSiteMode := http.SameSiteLaxMode
		secureFlag := false

		if isProduction {
			sameSiteMode = http.SameSiteNoneMode
			secureFlag = true
		}

		// 删除 access_token Cookie
		// MaxAge: -1 表示立即删除
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "access_token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,           // 立即过期
			Secure:   secureFlag,   // 与登录时一致
			HttpOnly: true,         // 保持 HttpOnly
			SameSite: sameSiteMode, // 与登录时一致
		})

		// 删除 refresh_token Cookie
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "refresh_token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			Secure:   secureFlag,
			HttpOnly: true,
			SameSite: sameSiteMode,
		})

		c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
	}
}
func RefreshTokenHandler(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(c, 100*time.Second)
		defer cancel()

		refreshToken, err := c.Cookie("refresh_token")

		if err != nil {
			fmt.Println("error", err.Error())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unable to retrieve refresh token from cookie"})
			return
		}

		claim, err := utils.ValidateRefreshToken(refreshToken)
		if err != nil || claim == nil {
			fmt.Println("error", err.Error())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
			return
		}

		var userCollection *mongo.Collection = database.OpenCollection("users", client)

		var user models.User
		err = userCollection.FindOne(ctx, bson.D{{Key: "user_id", Value: claim.UserID}}).Decode(&user)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}

		newToken, newRefreshToken, _ := utils.GenerateAllTokens(user.Email, user.FirstName, user.LastName, user.Role, user.UserID)
		err = utils.UpdateAllTokens(user.UserID, newToken, newRefreshToken, client)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating tokens"})
			return
		}

		c.SetCookie("access_token", newToken, 86400, "/", "localhost", true, true)          // expires in 24 hours
		c.SetCookie("refresh_token", newRefreshToken, 604800, "/", "localhost", true, true) //expires in 1 week

		c.JSON(http.StatusOK, gin.H{"message": "Tokens refreshed"})
	}
}
