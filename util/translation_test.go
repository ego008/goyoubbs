package util

import (
	"goyoubbs/model"
	"testing"
	"time"
)

func TestTranslateTopic(t *testing.T) {
	originalTopic := model.Topic{
		ID:         123,
		NodeId:     1,
		UserId:     10,
		Title:      "你好世界", // "Hello World" in Chinese
		Content:    "这是一个测试帖子。", // "This is a test post." in Chinese
		Language:   "zh",
		AddTime:    time.Now().Unix() - 3600,
		EditTime:   time.Now().Unix() - 3600,
		ClientIp:   "127.0.0.1",
		Tags:       "测试,翻译",
		ReadAuthed: false,
		ReadReply:  true,
		Comments:   0,
	}

	targetLang := "en"
	translatedTopic, err := TranslateTopic(originalTopic, targetLang)

	// 1. Assert that no error is returned
	if err != nil {
		t.Errorf("TranslateTopic returned an unexpected error: %v", err)
	}

	// 2. Assert that the returned Topic object has its Language field set to "en"
	if translatedTopic.Language != targetLang {
		t.Errorf("Expected translated topic language to be '%s', but got '%s'", targetLang, translatedTopic.Language)
	}

	// 3. Assert that the Title is not empty and is different from the original Chinese title
	if translatedTopic.Title == "" {
		t.Errorf("Expected translated title to be non-empty, but it was empty")
	}
	if translatedTopic.Title == originalTopic.Title {
		t.Errorf("Expected translated title ('%s') to be different from original title ('%s')", translatedTopic.Title, originalTopic.Title)
	}

	// 4. Assert that the Content is not empty and is different from the original Chinese content
	if translatedTopic.Content == "" {
		t.Errorf("Expected translated content to be non-empty, but it was empty")
	}
	if translatedTopic.Content == originalTopic.Content {
		t.Errorf("Expected translated content to be different from original content")
	}

	// 5. Assert that other fields like UserId, NodeId are correctly copied
	if translatedTopic.UserId != originalTopic.UserId {
		t.Errorf("Expected UserId to be %d, but got %d", originalTopic.UserId, translatedTopic.UserId)
	}
	if translatedTopic.NodeId != originalTopic.NodeId {
		t.Errorf("Expected NodeId to be %d, but got %d", originalTopic.NodeId, translatedTopic.NodeId)
	}
	// Check a few other copied fields to be reasonably sure
	if translatedTopic.ClientIp != originalTopic.ClientIp {
		t.Errorf("Expected ClientIp to be '%s', but got '%s'", originalTopic.ClientIp, translatedTopic.ClientIp)
	}
	if translatedTopic.Tags != originalTopic.Tags {
		t.Errorf("Expected Tags to be '%s', but got '%s'", originalTopic.Tags, translatedTopic.Tags)
	}
	if translatedTopic.ReadAuthed != originalTopic.ReadAuthed {
		t.Errorf("Expected ReadAuthed to be %t, but got %t", originalTopic.ReadAuthed, translatedTopic.ReadAuthed)
	}
	// AddTime should be inherited as per current TranslateTopic implementation
	if translatedTopic.AddTime != originalTopic.AddTime {
		t.Errorf("Expected AddTime to be %d, but got %d", originalTopic.AddTime, translatedTopic.AddTime)
	}


	// 6. Assert that the ID of the translated topic is 0
	if translatedTopic.ID != 0 {
		t.Errorf("Expected translated topic ID to be 0, but got %d", translatedTopic.ID)
	}

	// Check that EditTime was updated (should be different from original's EditTime and closer to now)
	if translatedTopic.EditTime == originalTopic.EditTime {
		t.Logf("Warning: Translated topic EditTime (%d) is the same as original EditTime (%d). This might be an issue if translation took negligible time or system clock has low resolution.", translatedTopic.EditTime, originalTopic.EditTime)
	}
	if translatedTopic.EditTime < originalTopic.EditTime {
		t.Errorf("Expected translated topic EditTime (%d) to be greater than or equal to original EditTime (%d)", translatedTopic.EditTime, originalTopic.EditTime)
	}

	// Optional: Log translated results for manual inspection if needed
	t.Logf("Original Title: %s", originalTopic.Title)
	t.Logf("Translated Title: %s", translatedTopic.Title)
	t.Logf("Original Content: %s", originalTopic.Content)
	t.Logf("Translated Content: %s", translatedTopic.Content)
}
