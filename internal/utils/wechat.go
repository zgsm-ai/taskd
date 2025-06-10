package utils

import (
	"encoding/json"
	"fmt"
	"log"
)

var weChatProxy string
var weChatEnable bool
var robotUrl string

type MessageBody struct {
	Content       string   `json:"content"`
	MentionedList []string `json:"mentioned_list"`
}

type WeChatMessage struct {
	MsgType string      `json:"msgtype"`
	Text    MessageBody `json:"text"`
}

/**
 * Set network proxy
 */
func SetProxyUrl(enable bool, proxy string, url string) {
	weChatEnable = enable
	weChatProxy = proxy
	robotUrl = url
}

/**
 * Handle service errors (without relying on logs)
 */
func ReportError(message ...any) {
	r := "Task Management Service Alert:\n"
	for _, v := range message {
		r += fmt.Sprintf("%v", v)
	}
	SendWeChatMessage(robotUrl, r, nil)
}

/**
 * Send WeChat message
 */
func SendWeChatMessage(webhookUrl, message string, employeeNumbers []string) error {
	if !weChatEnable {
		log.Printf("ignore, webhook=%s, message=%s, employee=%v\n",
			webhookUrl, message, employeeNumbers)
		return nil
	}
	ss := NewProxySession(webhookUrl, weChatProxy)

	var msg WeChatMessage
	msg.MsgType = "text"
	msg.Text.Content = message
	msg.Text.MentionedList = employeeNumbers

	data, err := json.Marshal(&msg)
	if err != nil {
		return err
	}
	_, err = ss.Post("", data)
	if err != nil {
		return err
	}
	return nil
}
