package main

import (
	"net/http"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func init() {
	// 初始化配置（环境变量）
	if listen, exist := os.LookupEnv("LISTEN"); exist {
		varListen = listen
	}
	if blogFeedLimit, exist := os.LookupEnv("FEED_LIMIT_BLOG"); exist {
		if blogFeedLimitInt, err := strconv.Atoi(blogFeedLimit); err == nil {
			varFeedLimitBlog = blogFeedLimitInt
		} // 忽略错误
	}
	if githubFeedLimit, exist := os.LookupEnv("FEED_LIMIT_GITHUB"); exist {
		if githubFeedLimitInt, err := strconv.Atoi(githubFeedLimit); err == nil {
			varFeedLimitGithub = githubFeedLimitInt
		} // 忽略错误
	}
	if countDats, exist := os.LookupEnv("COUNT_DAYS"); exist {
		if countDatsInt, err := strconv.Atoi(countDats); err == nil {
			varCountDays = countDatsInt
		} // 忽略错误
	}
	if githubToken, exist := os.LookupEnv("GITHUB_TOKEN"); exist {
		varGithubToken = githubToken
	} else {
		panic("GITHUB_TOKEN 未定义")
	}
}

func main() {

	// 创建服务器
	e := echo.New()

	e.Use(middleware.CORS())

	// 健康状态检查
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// Feed
	e.GET("/feed/blog", FeedBlog)
	e.GET("/feed/github", FeedGithub)

	// GitHub 活动 与 社交活动 数量统计
	e.GET("/count/social-activity", CountSocialActivity)

	// 启动服务器
	e.Logger.Fatal(e.Start(varListen))
}
