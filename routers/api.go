package routers

import (
	"database/sql"
	"log"
	"os"
	"time"

	"budgetingapi/controllers"
	"budgetingapi/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

func Route() *gin.Engine {
	router := gin.Default()
	router.Use(CORS())
	api := controllers.NewAPI()

	api.Db = newDB(nil)
	api.Db.SetConnMaxLifetime(5 * time.Minute)
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")

	api.Redis = redis.NewClient(&redis.Options{
		Addr: redisHost + ":" + redisPort,
		DB:   0,
	})

	router.POST("/api/login", api.Authenticate)
	router.GET("/api/check-session", middlewares.Auth(api.Redis), api.CheckSession)
	router.GET("/api/refresh-session", middlewares.Auth(api.Redis), api.RefreshSession)
	router.GET("/api/logout", middlewares.Auth(api.Redis), api.Logout)
	router.POST("/api/forgot-password", api.ForgotPassword)
	router.GET("/api/verify-token/:token", api.VerifyTokenReset)
	router.POST("/api/reset-password/:token", api.UpdateUserReset)

	product := router.Group("/api/products")
	product.Use(middlewares.Auth(api.Redis))
	{
		product.GET("", api.GetProducts)
		// batch upsert/delete
		product.POST("", api.UpsertProducts)
		product.DELETE("", api.DeleteProducts)
	}

	categories := router.Group("/api/categories")
	categories.Use(middlewares.Auth(api.Redis))
	{
		categories.GET("", api.GetCategories)
		// batch upsert/delete
		categories.POST("", api.UpsertCategories)
		categories.DELETE("", api.DeleteCategories)
	}

	expenses := router.Group("/api/expenses")
	expenses.Use(middlewares.Auth(api.Redis))
	{
		expenses.GET("", api.GetExpenses)
		expenses.GET("/report", api.GetExpensesReport)
		// batch upsert/delete
		expenses.POST("", api.UpsertExpenses)
		expenses.DELETE("", api.DeleteExpenses)
	}

	incomes := router.Group("/api/incomes")
	incomes.Use(middlewares.Auth(api.Redis))
	{
		incomes.GET("", api.GetIncomes)
		incomes.GET("/report", api.GetIncomesReport)
		// batch upsert/delete
		incomes.POST("", api.UpsertIncomes)
		incomes.DELETE("", api.DeleteIncomes)
	}
	return router
}

// CORS Cross Origin Resource Sharing
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, "+
			"Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func newDB(indb *sql.DB) *sql.DB {
	if indb != nil {
		return indb
	}
	connString := os.Getenv("DB_CONNECTION_STRING")
	if connString == "" {
		log.Fatal("Please provide DB_CONNECTION_STRING environment variable")
	}

	log.Println(connString)

	var err error
	conn, err := sql.Open("postgres", connString)
	if err != nil {
		log.Fatalf("Cannot connect to db with connection %s: %v", connString, err)
	}

	err = conn.Ping()
	if err != nil {
		log.Fatal(err)
	}

	return conn
}
