package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

// GitHub 的数据拉取参照 https://medium.com/@yuichkun/how-to-retrieve-contribution-graph-data-from-the-github-api-dc3a151b4af

const countGithubQueryBody = `
query($userName:String!) {
  user(login: $userName){
    contributionsCollection {
      contributionCalendar {
        totalContributions
        weeks {
          contributionDays {
            contributionCount
            date
          }
        }
      }
    }
  }
}
`

const countGithubVariablesTemplate = `
{
  "userName": "Candinya"
}
`

type countGithubResponse struct {
	Data struct {
		User struct {
			ContributionsCollection struct {
				ContributionCalendar struct {
					TotalContributions int `json:"totalContributions"`
					Weeks              []struct {
						ContributionDays []struct {
							ContributionCount int    `json:"contributionCount"`
							Date              string `json:"date"` // 非规范时间戳格式， Go 没法直接格式化，会报错，所以需要手动处理
						} `json:"contributionDays"`
					} `json:"weeks"`
				} `json:"contributionCalendar"`
			} `json:"contributionsCollection"`
		} `json:"user"`
	} `json:"data"`
}

type activityCountResult struct {
	Total int   `json:"total"`
	Max   int   `json:"max"`
	Day   []int `json:"day"` // 倒序，从当前天开始
}

func countGithubActivity() (*activityCountResult, error) {
	// 使用 GraphQL 查询拉取 GitHub 活动记录

	// 准备请求体
	requestBody := map[string]string{
		"query":     countGithubQueryBody,
		"variables": countGithubVariablesTemplate,
	}
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("请求体准备失败: %v", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", "https://api.github.com/graphql", bytes.NewReader(requestBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("请求构建失败: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+varGithubToken)

	// 发出请求
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求发送失败: %v", err)
	}

	defer res.Body.Close()

	var result countGithubResponse

	// 解析结果
	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("无法解析响应结果: %v", err)
	}

	processedResult := activityCountResult{
		Total: 0,
		Max:   -1,
		Day:   make([]int, varCountDays), // 初始化定长数组
	}

	// 处理结果
	nowSec := time.Now().Unix()
	for _, week := range result.Data.User.ContributionsCollection.ContributionCalendar.Weeks {
		for _, day := range week.ContributionDays {
			if day.ContributionCount > 0 {
				date, _ := time.Parse("2006-01-02", day.Date)   // 2023-12-03
				diffDays := int((nowSec - date.Unix()) / 86400) // 一天有 24 * 3600 = 86400 秒，通过时间差来计算有多少天
				if diffDays < varCountDays {
					// 属于计算日里面，累加
					processedResult.Day[diffDays] = day.ContributionCount
					processedResult.Total += day.ContributionCount
					if day.ContributionCount > processedResult.Max {
						processedResult.Max = day.ContributionCount
					}
				} // else 不属于计算日，忽略
			} // else 没有贡献，没必要累加
		}
	}

	// 返回结果
	return &processedResult, nil
}

type countMisskeyResponse struct {
	Total []int `json:"total"`
	Inc   []int `json:"inc"`
	Dec   []int `json:"dec"`
	Diffs struct {
		Normal   []int `json:"normal"`
		Reply    []int `json:"reply"`
		Renote   []int `json:"renote"`
		WithFile []int `json:"withFile"`
	} `json:"diffs"`
}

func countMisskeyActivity() (*activityCountResult, error) {
	res, err := http.Get(fmt.Sprintf("https://nya.one/api/charts/user/notes?userId=8837yxdz1d&limit=%d&span=day", varCountDays))
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}

	var result countMisskeyResponse

	// 解析结果
	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("无法解析响应结果: %v", err)
	}

	processedResult := activityCountResult{
		Total: 0,
		Max:   -1,
		Day:   result.Inc,
	}

	for _, r := range result.Inc {
		processedResult.Total += r
		if r > processedResult.Max {
			processedResult.Max = r
		}
	}

	return &processedResult, nil
}

var countCache cacheData

func CountActivity(c echo.Context) error {
	if countCache.data == nil || time.Now().Sub(countCache.createdAt) > 12*time.Hour {
		// 请求新的数据

		resGithub, err := countGithubActivity()
		if err != nil {
			c.Logger().Error(err)
			return c.String(http.StatusInternalServerError, "GitHub 统计失败")
		}

		resMisskey, err := countMisskeyActivity()
		if err != nil {
			c.Logger().Error(err)
			return c.String(http.StatusInternalServerError, "Misskey 统计失败")
		}

		dataJson := map[string]interface{}{
			"days":    varCountDays,
			"github":  resGithub,
			"misskey": resMisskey,
		}
		dataBytes, err := json.Marshal(dataJson)
		if err != nil {
			c.Logger().Error(err)
			return c.String(http.StatusInternalServerError, "响应结果格式化失败")
		}

		countCache.data = dataBytes
		countCache.createdAt = time.Now()
	}

	// 直接作为 binary 输出
	return c.Blob(http.StatusOK, "application/json", countCache.data)
}
