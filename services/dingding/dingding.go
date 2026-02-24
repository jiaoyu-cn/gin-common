package dingding

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DingDing 钉钉机器人客户端
type DingDing struct {
	Token  string
	Secret string
	URL    string
	Client *http.Client
}

// NewDingDing 创建钉钉机器人实例
func NewDingDing(token, secret string) *DingDing {
	return &DingDing{
		Token:  token,
		Secret: secret,
		URL:    "https://oapi.dingtalk.com/robot/send?access_token=" + token,
		Client: &http.Client{Timeout: 5 * time.Second},
	}
}

type Option func(*MessageConfig)

type MessageConfig struct {
	IsAtAll        bool
	AtMobiles      []string
	AtUserIds      []string
	PicURL         string
	BtnOrientation string
	HideAvatar     string
}

func WithAt(mobiles []string, userIds []string) Option {
	return func(c *MessageConfig) {
		c.AtMobiles = mobiles
		c.AtUserIds = userIds
	}
}

func WithAtAll() Option {
	return func(c *MessageConfig) {
		c.IsAtAll = true
	}
}

func WithPicURL(picURL string) Option {
	return func(c *MessageConfig) {
		c.PicURL = picURL
	}
}

func WithBtnOrientation(orientation string) Option {
	return func(c *MessageConfig) {
		c.BtnOrientation = orientation
	}
}

func WithHideAvatar(hideAvatar string) Option {
	return func(c *MessageConfig) {
		c.HideAvatar = hideAvatar
	}
}

type TextMessage struct {
	MsgType string `json:"msgtype"`
	Text    Text   `json:"text"`
	At      *At    `json:"at,omitempty"`
}

type Text struct {
	Content string `json:"content"`
}

type At struct {
	IsAtAll   bool     `json:"isAtAll"`
	AtMobiles []string `json:"atMobiles"`
	AtUserIds []string `json:"atUserIds"`
}

type LinkMessage struct {
	MsgType string `json:"msgtype"`
	Link    Link   `json:"link"`
}

type Link struct {
	MessageURL string `json:"messageUrl"`
	Title      string `json:"title"`
	PicURL     string `json:"picUrl"`
	Text       string `json:"text"`
}

type MarkdownMessage struct {
	MsgType  string   `json:"msgtype"`
	Markdown Markdown `json:"markdown"`
	At       *At      `json:"at,omitempty"`
}

type Markdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type ActionCardMessage struct {
	MsgType    string     `json:"msgtype"`
	ActionCard ActionCard `json:"actionCard"`
	At         *At        `json:"at,omitempty"`
}

type ActionCard struct {
	HideAvatar     string          `json:"hideAvatar"`
	BtnOrientation string          `json:"btnOrientation"`
	SingleTitle    string          `json:"singleTitle,omitempty"`
	SingleURL      string          `json:"singleURL,omitempty"`
	Text           string          `json:"text"`
	Title          string          `json:"title"`
	Btns           []ActionCardBtn `json:"btns,omitempty"`
}

type ActionCardBtn struct {
	Title     string `json:"title"`
	ActionURL string `json:"actionURL"`
}

type FeedCardMessage struct {
	MsgType  string   `json:"msgtype"`
	FeedCard FeedCard `json:"feedCard"`
}

type FeedCard struct {
	Links []FeedCardLink `json:"links"`
}

type FeedCardLink struct {
	MessageURL string `json:"messageURL"`
	Title      string `json:"title"`
	PicURL     string `json:"picURL"`
}

type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SendText 发送文本消息
func (d *DingDing) SendText(message string, opts ...Option) Response {
	if message == "" {
		return d.message("2000", "文本消息的内容不能为空")
	}
	config := &MessageConfig{}
	for _, opt := range opts {
		opt(config)
	}
	msg := TextMessage{
		MsgType: "text",
		Text: Text{
			Content: message,
		},
	}
	if config.IsAtAll || len(config.AtMobiles) > 0 || len(config.AtUserIds) > 0 {
		msg.At = &At{
			IsAtAll:   config.IsAtAll,
			AtMobiles: config.AtMobiles,
			AtUserIds: config.AtUserIds,
		}
	}
	return d.send(msg)
}

// SendLink 发送链接消息
func (d *DingDing) SendLink(messageURL, title, text string, opts ...Option) Response {
	if messageURL == "" {
		return d.message("2000", "链接消息的URL不能为空")
	}
	if title == "" {
		return d.message("2000", "链接消息的标题不能为空")
	}
	config := &MessageConfig{}
	for _, opt := range opts {
		opt(config)
	}
	msg := LinkMessage{
		MsgType: "link",
		Link: Link{
			MessageURL: messageURL,
			Title:      title,
			Text:       text,
		},
	}
	msg.Link.PicURL = config.PicURL
	return d.send(msg)
}

// SendMarkdown 发送Markdown消息
func (d *DingDing) SendMarkdown(title, text string, opts ...Option) Response {
	if title == "" {
		return d.message("2000", "Markdown消息的标题不能为空")
	}
	if text == "" {
		return d.message("2000", "Markdown消息的文本不能为空")
	}
	config := new(MessageConfig)
	for _, opt := range opts {
		opt(config)
	}
	msg := MarkdownMessage{
		MsgType: "markdown",
		Markdown: Markdown{
			Title: title,
			Text:  text,
		},
	}
	if config.IsAtAll || len(config.AtMobiles) > 0 || len(config.AtUserIds) > 0 {
		msg.At = &At{
			AtMobiles: config.AtMobiles,
			AtUserIds: config.AtUserIds,
			IsAtAll:   config.IsAtAll,
		}
	}
	return d.send(msg)
}

// SendActionCard 发送ActionCard消息
func (d *DingDing) SendActionCard(title, text string, btns []ActionCardBtn, opts ...Option) Response {
	// 参数验证
	for _, btn := range btns {
		if btn.Title == "" {
			return d.message("2000", "ActionCard消息按钮title不能为空")
		}
		if btn.ActionURL == "" {
			return d.message("2000", "ActionCard消息按钮actionURL不能为空")
		}
	}
	config := &MessageConfig{}
	for _, opt := range opts {
		opt(config)
	}
	if config.BtnOrientation == "" {
		config.BtnOrientation = "0"
	}
	if config.HideAvatar == "" {
		config.HideAvatar = "0"
	}
	msg := ActionCardMessage{
		MsgType: "actionCard",
		ActionCard: ActionCard{
			Title:          title,
			Text:           text,
			BtnOrientation: config.BtnOrientation,
			HideAvatar:     config.HideAvatar,
		},
	}
	if len(btns) == 1 {
		msg.ActionCard.SingleTitle = btns[0].Title
		msg.ActionCard.SingleURL = btns[0].ActionURL
	}
	if len(btns) > 1 {
		msg.ActionCard.Btns = btns
	}

	if config.IsAtAll || len(config.AtMobiles) > 0 || len(config.AtUserIds) > 0 {
		msg.At = &At{
			AtMobiles: config.AtMobiles,
			AtUserIds: config.AtUserIds,
			IsAtAll:   config.IsAtAll,
		}
	}
	return d.send(msg)
}

// SendFeedCard 发送FeedCard消息
func (d *DingDing) SendFeedCard(links []FeedCardLink) Response {
	// 参数验证
	for _, link := range links {
		if link.MessageURL == "" {
			return d.message("2000", "FeedCard消息的URL不能为空")
		}
		if link.Title == "" {
			return d.message("2000", "FeedCard消息的标题不能为空")
		}
	}
	msg := FeedCardMessage{
		MsgType: "feedCard",
		FeedCard: FeedCard{
			Links: links,
		},
	}
	return d.send(msg)
}

// send 发送消息
func (d *DingDing) send(message interface{}) Response {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return d.message("2000", fmt.Sprintf("序列化消息失败: %s", err.Error()))
	}
	url := d.URL
	if d.Secret != "" {
		timestamp := time.Now().UnixNano() / 1e6
		sign := fmt.Sprintf("%d\n%s", timestamp, d.Secret)

		h := hmac.New(sha256.New, []byte(d.Secret))
		h.Write([]byte(sign))
		signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

		url = fmt.Sprintf("%s&timestamp=%d&sign=%s", url, timestamp, signature)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return d.message("2000", fmt.Sprintf("创建请求失败: %s", err.Error()))
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	resp, err := d.Client.Do(req)
	if err != nil {
		return d.message("2000", fmt.Sprintf("请求机器人失败: %s", err.Error()))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return d.message("2000", fmt.Sprintf("读取响应失败: %s", err.Error()))
	}
	var dingResp struct {
		Code    int    `json:"errcode"`
		Message string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &dingResp); err != nil {
		return d.message("2000", fmt.Sprintf("解析响应失败: %s", err.Error()))
	}
	if dingResp.Code != 0 {
		return d.message("2000", dingResp.Message)
	}

	return d.message("0000", "发送成功")
}

func (d *DingDing) message(code, message string) Response {
	return Response{
		Code:    code,
		Message: message,
	}
}
