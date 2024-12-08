package main

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

var githubLikeCache cacheData

type likeGithubResponse []struct {
	FullName string `json:"full_name"`
	// 忽略其他用不到的字段
}

func LikeGithub(c echo.Context) error {
	if githubLikeCache.data == nil || time.Now().Sub(githubLikeCache.createdAt) > 1*time.Hour {
		// 重新请求数据
		req, err := http.NewRequest("GET", "https://api.github.com/users/Candinya/starred", nil)
		if err != nil {
			c.Logger().Error(fmt.Errorf("请求创建失败: %v", err))
			return c.String(http.StatusInternalServerError, "GitHub Stars 请求创建失败")
		}

		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Authorization", "Bearer "+varGithubToken)
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			c.Logger().Error(fmt.Errorf("请求失败: %v", err))
			return c.String(http.StatusInternalServerError, "GitHub Stars 请求发送失败")
		}

		defer res.Body.Close()

		var result likeGithubResponse
		err = json.NewDecoder(res.Body).Decode(&result)
		if err != nil {
			c.Logger().Error(fmt.Errorf("响应解析失败: %v", err))
			return c.String(http.StatusInternalServerError, "GitHub Stars 响应解析失败")
		}

		var processedResult []string // 只返回一个列表
		for _, project := range result {
			processedResult = append(processedResult, project.FullName)
		}

		// 截取结果
		if varLikeLimitGithub > 0 && len(processedResult) > varLikeLimitGithub {
			processedResult = processedResult[:varLikeLimitGithub]
		}

		dataBytes, err := json.Marshal(processedResult)
		if err != nil {
			c.Logger().Error(fmt.Errorf("结果编码失败: %v", err))
			return c.String(http.StatusInternalServerError, "响应结果格式化失败")
		}

		githubLikeCache.data = dataBytes
		githubLikeCache.createdAt = time.Now()
	}

	// 直接作为 binary 输出
	return c.Blob(http.StatusOK, "application/json", githubLikeCache.data)
}
