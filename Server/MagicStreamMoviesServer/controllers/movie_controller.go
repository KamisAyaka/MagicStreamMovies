// Package controllers 包含电影相关的HTTP处理器函数
package controllers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/KamisAyaka/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	"github.com/KamisAyaka/MagicStreamMovies/Server/MagicStreamMoviesServer/models"
	"github.com/KamisAyaka/MagicStreamMovies/Server/MagicStreamMoviesServer/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms/openai"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// 全局变量定义
var validate = validator.New() // 数据验证器实例

// GetMovies 获取所有电影的处理器函数
// 返回所有存储在数据库中的电影列表
func GetMovies(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 创建带超时的上下文，防止数据库操作超时
		ctx, cancel := context.WithTimeout(c, 100*time.Second)
		defer cancel()

		var movieCollection *mongo.Collection = database.OpenCollection("movies", client) // 电影集合

		var movies []models.Movie

		// 查询所有电影记录
		cursor, err := movieCollection.Find(ctx, bson.M{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching movies"})
			return
		}
		defer cursor.Close(ctx)

		// 将查询结果解码到movies切片中
		if err = cursor.All(ctx, &movies); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching movies"})
			return
		}
		// 返回成功响应和电影列表
		c.JSON(http.StatusOK, movies)
	}
}

// GetMovie 根据IMDB ID获取单个电影的处理器函数
// 通过URL参数中的imdb_id来查找特定电影
func GetMovie(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(c, 100*time.Second)
		defer cancel()

		// 从URL参数中获取电影ID
		movieID := c.Param("imdb_id")

		// 验证电影ID是否为空
		if movieID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Movie ID is required"})
			return
		}
		var movie models.Movie

		var movieCollection *mongo.Collection = database.OpenCollection("movies", client)

		// 根据IMDB ID查找电影
		err := movieCollection.FindOne(ctx, bson.M{"imdb_id": movieID}).Decode(&movie)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Movie not found"})
			return
		}
		// 返回找到的电影信息
		c.JSON(http.StatusOK, movie)
	}
}

// AddMovie 添加新电影的处理器函数
// 接收JSON格式的电影数据并存储到数据库中
func AddMovie(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(c, 100*time.Second)
		defer cancel()

		var movie models.Movie
		// 将请求体中的JSON数据绑定到movie结构体
		if err := c.ShouldBindJSON(&movie); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
			return
		}
		// 验证电影数据的有效性
		if err := validate.Struct(movie); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
			return
		}
		var movieCollection *mongo.Collection = database.OpenCollection("movies", client)

		// 将电影数据插入到数据库中
		result, err := movieCollection.InsertOne(ctx, movie)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding movie"})
			return
		}
		// 返回创建成功的结果
		c.JSON(http.StatusCreated, result)

	}
}

// AdminReviewUpdate 管理员更新电影评论的处理器函数
// 使用AI分析评论内容并自动分配排名等级
func AdminReviewUpdate(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, err := utils.GetRoleFromContext(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Role not found in context"})
			return
		}
		if role != "ADMIN" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
			return
		}
		// 从URL参数获取电影ID
		movieId := c.Param("imdb_id")
		if movieId == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Movie Id required"})
			return
		}

		// 定义请求和响应结构体
		var req struct {
			AdminReview string `json:"admin_review"`
		}
		var resp struct {
			RankingName string `json:"ranking_name"`
			AdminReview string `json:"admin_review"`
		}

		// 绑定请求数据
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
			return
		}

		// 使用AI分析评论并获取排名
		sentiment, rankVal, err := GetReviewRanking(req.AdminReview, client, c)
		if err != nil {
			log.Printf("Error getting review ranking: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting review ranking", "details": err.Error()})
			return
		}

		// 构建数据库更新操作
		filter := bson.M{"imdb_id": movieId}
		update := bson.M{
			"$set": bson.M{
				"admin_review": req.AdminReview,
				"ranking": bson.M{
					"ranking_value": rankVal,
					"ranking_name":  sentiment,
				},
			},
		}

		// 创建数据库操作上下文
		var ctx, cancel = context.WithTimeout(c, 100*time.Second)
		defer cancel()
		var movieCollection *mongo.Collection = database.OpenCollection("movies", client)

		// 执行数据库更新操作
		result, err := movieCollection.UpdateOne(ctx, filter, update)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating movie"})
			return
		}

		// 检查是否找到要更新的电影
		if result.MatchedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Movie not found"})
			return
		}

		// 构建响应数据
		resp.RankingName = sentiment
		resp.AdminReview = req.AdminReview

		// 返回更新结果
		c.JSON(http.StatusOK, resp)
	}
}

// GetReviewRanking 使用AI分析评论内容并返回相应的排名等级
// 参数: admin_review - 管理员评论内容
// 返回: 排名名称, 排名数值, 错误信息
func GetReviewRanking(admin_review string, client *mongo.Client, c *gin.Context) (string, int, error) {
	// 获取所有可用的排名等级
	rankings, err := GetRankings(client, c)
	if err != nil {
		log.Printf("Error getting rankings: %v", err)
		return "", 0, err
	}

	// 构建排名名称的逗号分隔字符串，用于AI提示
	sentimentDelimited := ""
	for _, ranking := range rankings {
		if ranking.RankingValue != 999 { // 排除特殊值999
			sentimentDelimited += ranking.RankingName + ","
		}
	}
	sentimentDelimited = strings.Trim(sentimentDelimited, ",")

	// 加载环境变量文件
	err = godotenv.Load(".env")
	if err != nil {
		log.Println("Warning: Error loading .env file")
	}

	// 获取DeepSeek API密钥
	deepseekApiKey := os.Getenv("DEEPSEEK_API_KEY")
	if deepseekApiKey == "" {
		log.Println("Error: DEEPSEEK_API_KEY is not set in .env file")
		return "", 0, errors.New("DEEPSEEK_API_KEY is not set")
	}

	// 创建DeepSeek LLM实例（使用OpenAI兼容接口）
	llm, err := openai.New(
		openai.WithToken(deepseekApiKey),
		openai.WithBaseURL("https://api.deepseek.com"),
		openai.WithModel("deepseek-chat"),
	)
	if err != nil {
		log.Printf("Error creating DeepSeek LLM: %v", err)
		return "", 0, err
	}

	// 构建AI提示模板
	base_prompt_template := os.Getenv("BASE_PROMPT_TEMPLATE")
	base_prompt := strings.Replace(base_prompt_template, "{rankings}", sentimentDelimited, 1)

	// 调用AI分析评论内容
	response, err := llm.Call(context.Background(), base_prompt+admin_review)
	if err != nil {
		log.Printf("Error calling DeepSeek API: %v", err)
		return "", 0, err
	}

	// 根据AI返回的排名名称查找对应的数值
	rankVal := 0
	for _, ranking := range rankings {
		if ranking.RankingName == response {
			rankVal = ranking.RankingValue
			break
		}
	}

	return response, rankVal, nil
}

// GetRankings 获取所有排名等级的辅助函数
// 从数据库中查询所有可用的排名等级信息
func GetRankings(client *mongo.Client, c *gin.Context) ([]models.Ranking, error) {
	var rankings []models.Ranking

	// 创建带超时的上下文
	var ctx, cancel = context.WithTimeout(c, 100*time.Second)
	defer cancel()
	var rankingCollection *mongo.Collection = database.OpenCollection("rankings", client)

	// 查询所有排名记录
	cursor, err := rankingCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// 将查询结果解码到rankings切片中
	if err := cursor.All(ctx, &rankings); err != nil {
		return nil, err
	}

	return rankings, nil
}

// GetRecommendedMovies 获取用户推荐电影的处理器函数
// 根据用户喜欢的电影类型，返回评分最高的推荐电影列表
func GetRecommendedMovies(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文中获取用户ID
		userId, err := utils.GetUserIdFromContext(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User ID not found in context"})
			return
		}

		// 获取用户喜欢的电影类型列表
		favourite_genres, err := GetUserFavouriteGenres(userId, client, c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting favourite genres"})
			return
		}

		// 加载环境变量文件
		err = godotenv.Load(".env")
		if err != nil {
			log.Println("Warning: Error loading .env file")
		}

		// 从环境变量获取推荐电影数量限制，默认为5部
		var recommendedMoviesLimitVal int64 = 5
		recommendedMoviesLimitStr := os.Getenv("RECOMMENDED_MOVIES_LIMIT")
		if recommendedMoviesLimitStr != "" {
			recommendedMoviesLimitVal, _ = strconv.ParseInt(recommendedMoviesLimitStr, 10, 64)
		}

		// 设置查询选项：按排名值升序排序（值越小排名越高），限制返回数量
		findOptions := options.Find()
		findOptions.SetSort(bson.D{{Key: "ranking.ranking_value", Value: 1}})
		findOptions.SetLimit(recommendedMoviesLimitVal)

		// 构建过滤条件：电影类型在用户喜欢的类型列表中
		filter := bson.M{"genre.genre_name": bson.M{"$in": favourite_genres}}

		// 创建数据库操作上下文
		var ctx, cancel = context.WithTimeout(c, 100*time.Second)
		defer cancel()
		var movieCollection *mongo.Collection = database.OpenCollection("movies", client)

		// 执行数据库查询
		cursor, err := movieCollection.Find(ctx, filter, findOptions)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching recommended movies"})
			return
		}
		defer cursor.Close(ctx)

		// 将查询结果解码到推荐电影列表中
		var recommendedMovies []models.Movie
		if err := cursor.All(ctx, &recommendedMovies); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding recommended movies"})
			return
		}

		// 返回推荐电影列表
		c.JSON(http.StatusOK, recommendedMovies)

	}
}

// GetUserFavouriteGenres 获取用户喜欢的电影类型列表
// 参数: userId - 用户ID
// 返回: 类型名称字符串切片, 错误信息
func GetUserFavouriteGenres(userId string, client *mongo.Client, c *gin.Context) ([]string, error) {
	// 创建带超时的数据库操作上下文
	var ctx, cancel = context.WithTimeout(c, 100*time.Second)
	defer cancel()

	// 构建查询条件和投影
	filter := bson.M{"user_id": userId}
	projection := bson.M{
		"favourite_genres.genre_name": 1, // 只返回喜欢的类型名称
		"_id":                         0, // 不返回_id字段
	}
	opts := options.FindOne().SetProjection(projection)

	// 执行数据库查询
	var result bson.M
	var userCollection *mongo.Collection = database.OpenCollection("users", client)
	err := userCollection.FindOne(ctx, filter, opts).Decode(&result)
	if err != nil {
		// 如果找不到用户文档，返回空切片
		if err == mongo.ErrNoDocuments {
			return []string{}, nil
		}
		return nil, err
	}

	// 将favourite_genres字段转换为BSON数组
	favGenresArray, ok := result["favourite_genres"].(bson.A)
	if !ok {
		return []string{}, errors.New("favourite_genres is not an array")
	}

	// 遍历数组提取所有类型名称
	var genreName []string
	for _, item := range favGenresArray {
		// 将数组项转换为BSON文档
		if genreMap, ok := item.(bson.D); ok {
			// 遍历文档中的所有字段
			for _, elem := range genreMap {
				// 查找genre_name字段
				if elem.Key == "genre_name" {
					if name, ok := elem.Value.(string); ok {
						genreName = append(genreName, name)
					}
				}
			}
		}
	}

	return genreName, nil
}

func GetGenre(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(c, 100*time.Second)
		defer cancel()
		var genres []models.Genre
		var genreCollection *mongo.Collection = database.OpenCollection("genres", client)
		cursor, err := genreCollection.Find(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching genres"})
			return
		}
		defer cursor.Close(ctx)
		if err := cursor.All(ctx, &genres); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding genres"})
			return
		}
		c.JSON(http.StatusOK, genres)
	}
}
