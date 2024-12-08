package main

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/mmcdole/gofeed"
	"net/http"
	"time"
)

func feedProcess(feedUrl string) ([]*gofeed.Item, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(feedUrl)
	if err != nil {
		return nil, fmt.Errorf("feed 格式化失败: %v", err)
	}

	return feed.Items, nil
}

var blogFeedCache cacheData
var githubFeedCache cacheData

type blogFeedResponseItem struct {
	Cover      string    `json:"cover"`
	Date       time.Time `json:"date"`
	Title      string    `json:"title"`
	Categories []string  `json:"categories"`
	Link       string    `json:"link"`
}

type githubFeedResponseItem struct {
	Date  time.Time `json:"date"`
	Title string    `json:"title"`
	Link  string    `json:"link"`
}

func FeedBlog(c echo.Context) error {
	if blogFeedCache.data == nil || time.Now().Sub(blogFeedCache.createdAt) > 12*time.Hour {
		// 重新拉取
		feed, err := feedProcess("https://candinya.com/atom.xml")
		if err != nil {
			c.Logger().Error(err)
			return c.JSON(http.StatusInternalServerError, "博客 feed 处理失败")
		}

		// 处理数据
		var itemsSelected []*blogFeedResponseItem
		for (varFeedLimitBlog <= 0 || len(itemsSelected) < varFeedLimitBlog) && len(feed) > 0 {
			if feed[0].Title != "" {
				item := blogFeedResponseItem{
					Date:       *feed[0].PublishedParsed,
					Title:      feed[0].Title,
					Categories: feed[0].Categories,
					Link:       feed[0].Link,
				}
				if len(feed[0].Extensions["media"]["thumbnail"]) > 0 {
					item.Cover = feed[0].Extensions["media"]["thumbnail"][0].Attrs["url"]
				} else {
					item.Cover = "https://candinya.com/images/default.webp" // 默认图片
				}
				itemsSelected = append(itemsSelected, &item)
			}
			feed = feed[1:]
		}

		// 缓存
		dataBytes, err := json.Marshal(itemsSelected)
		if err != nil {
			c.Logger().Error(err)
			return c.String(http.StatusInternalServerError, "响应结果格式化失败")
		}

		blogFeedCache.data = dataBytes
		blogFeedCache.createdAt = time.Now()
	}

	// 直接作为 binary 输出
	return c.Blob(http.StatusOK, "application/json", blogFeedCache.data)
}

func FeedGithub(c echo.Context) error {
	if githubFeedCache.data == nil || time.Now().Sub(githubFeedCache.createdAt) > 1*time.Hour {
		// 重新拉取
		feed, err := feedProcess("https://github.com/Candinya.atom")
		if err != nil {
			c.Logger().Error(err)
			return c.JSON(http.StatusInternalServerError, "GitHub feed 处理失败")
		}

		// 处理数据
		var itemsSelected []*githubFeedResponseItem
		for (varFeedLimitGithub <= 0 || len(itemsSelected) < varFeedLimitGithub) && len(feed) > 0 {
			itemsSelected = append(itemsSelected, &githubFeedResponseItem{
				Date:  *feed[0].PublishedParsed,
				Title: feed[0].Title,
				Link:  feed[0].Link,
			})
			feed = feed[1:]
		}

		// 缓存
		dataBytes, err := json.Marshal(itemsSelected)
		if err != nil {
			c.Logger().Error(err)
			return c.String(http.StatusInternalServerError, "响应结果格式化失败")
		}

		githubFeedCache.data = dataBytes
		githubFeedCache.createdAt = time.Now()
	}

	// 直接作为 binary 输出
	return c.Blob(http.StatusOK, "application/json", githubFeedCache.data)
}
