package wechat

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type user struct {
	Name     string `json:"name"`
	Username string `json:"username"`
}

type project struct {
	ID   int32  `json:"id"`
	Name string `json:"path_with_namespace"`
	URL  string `json:"web_url"`
}

type assignee struct {
	Name     string `json:"name"`
	Username string `json:"username"`
}

type mergeRequestObjectAttributes struct {
	URL            string `json:"url"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	State          string `json:"state"`
	Action         string `json:"action"`
	MergeRequestID int32  `json:"iid"`
}

type mergeRequestPayload struct {
	User             user                         `json:"user"`
	Project          project                      `json:"project"`
	ObjectAttributes mergeRequestObjectAttributes `json:"object_attributes"`
	Assignees        []assignee                   `json:"assignees"`
}

type markdown struct {
	Content string `json:"content"`
}

type pipelineObjectAttributes struct {
	ID     int32  `json:"id"`
	Status string `json:"status"`
	Ref    string `json:"ref"`
}

type commitAuthor struct {
	Name  string `json:name`
	Email string `json:email`
}

type commit struct {
	URL     string       `json:"url"`
	Message string       `json:"message"`
	Author  commitAuthor `json:"author"`
}

type pipelinePayload struct {
	Project          project                  `json:"project"`
	ObjectAttributes pipelineObjectAttributes `json:"object_attributes"`
	Commit           commit                   `json:"commit"`
}

type piplineContentData struct {
	Text, ClassName string
}

var piplineContentDataMap = map[string]piplineContentData{
	"success": piplineContentData{"成功🎉", "info"},
	"failed":  piplineContentData{"失败🤔", "warning"},
}

// 处理合并请求 hook
/**
```markdown
# candyabc/ios 有新的合并请求
标题: 微信登录
描述: 无
提交: @张三
审核: @林国锋
操作: [查看]
`
*/
func handleMergeRequestHook(c echo.Context) error {
	key := c.Param("key")

	payload := new(mergeRequestPayload)
	if err := c.Bind(payload); err != nil {
		return err
	}

	// 创建
	if payload.ObjectAttributes.Action == "open" {
		description := payload.ObjectAttributes.Description
		if description == "" {
			description = "无"
		}
		assignees := make([]string, len(payload.Assignees))
		for i, assigne := range payload.Assignees {
			assignees[i] = fmt.Sprint(assigne.Name, "(", assigne.Username, ")")
		}
		content := fmt.Sprint(
			"### [", payload.Project.Name, "](", payload.Project.URL, ") 有新的合并请求 [!", payload.ObjectAttributes.MergeRequestID, "](", payload.ObjectAttributes.URL, ")\n",
			"> 标题: ", payload.ObjectAttributes.Title, "\n",
			"> 描述: ", description, "\n",
			"> 提交: ", payload.User.Name, "(", payload.User.Username, ")\n",
			"> 审核: ", strings.Join(assignees[:], " "), "\n",
			"> 操作: [[查看](", payload.ObjectAttributes.URL, ")]",
		)

		err := send(key, content)
		if err != nil {
			c.Logger().Error(err)
			return err
		}
	}

	// 合并
	if payload.ObjectAttributes.Action == "merge" {
		content := fmt.Sprint(
			"### [", payload.Project.Name, "](", payload.Project.URL, ") 合并请求 [!", payload.ObjectAttributes.MergeRequestID, "](", payload.ObjectAttributes.URL, ") 已合并\n",
			"> 合并: ", payload.User.Name, "(", payload.User.Username, ")\n",
			"> 操作: [[查看](", payload.ObjectAttributes.URL, ")]",
		)

		err := send(key, content)
		if err != nil {
			c.Logger().Error(err)
			return err
		}
	}

	return c.String(http.StatusOK, "OK")
}

// 处理流水线 hook
/**
```markdown
仓库pay的release/test/v2.6分支部署成功🎉/失败🤔\n
> <font color=\"{{#success build.status}}info{{else}}warning{{/success}}\">{{修复了XXX问题}}</font>\n
> <font color=\"comment\">zhangsan@example.com</font>\n
> 点击进入 [git提交详情页面](http://www.example.com)\n
> 点击进入 [drone构建详情页面](http://www.example.com})
`
*/
func handlePipelineHook(c echo.Context) error {
	key := c.Param("key")

	payload := new(pipelinePayload)
	if err := c.Bind(payload); err != nil {
		return err
	}

	pipelineStatus := payload.ObjectAttributes.Status

	if pipelineStatus == "success" || pipelineStatus == "failed" {
		contentData := piplineContentDataMap[pipelineStatus]
		content := fmt.Sprint(
			"仓库", payload.Project.Name, "的", payload.ObjectAttributes.Ref, "分支部署", contentData.Text, "\n",
			"> <font color=\"", contentData.ClassName, "\">", payload.Commit.Message, "</font>\n",
			"> <font color=\"comment\">", payload.Commit.Author.Email, "</font>\n",
			"> 点击进入 [git提交详情页面](", payload.Commit.URL, ")\n",
			"> 点击进入 [ci构建详情页面](", payload.Project.URL, "/pipelines/", payload.ObjectAttributes.ID, ")",
		)

		err := send(key, content)
		if err != nil {
			c.Logger().Error(err)
			return err
		}
	}

	return c.String(http.StatusOK, "OK")
}

// GitlabHandler 处理 Gitlab
// https://docs.gitlab.com/ee/user/project/integrations/webhooks.html#merge-request-events
func GitlabHandler(c echo.Context) error {
	event := c.Request().Header.Get("X-Gitlab-Event")
	if event == "Merge Request Hook" {
		return handleMergeRequestHook(c)
	} else if event == "Pipeline Hook" {
		return handlePipelineHook(c)
	}
	return c.String(http.StatusOK, "OK")
}
