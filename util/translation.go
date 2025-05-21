package util

import (
	"github.com/bregydoc/gtranslate"
	"goyoubbs/model"
	"time"
)

// TranslateTopic translates the title and content of a topic to the target language.
func TranslateTopic(topic model.Topic, targetLang string) (model.Topic, error) {
	translatedTitle, err := gtranslate.TranslateWithParams(
		topic.Title,
		gtranslate.TranslationParams{
			From: "zh",
			To:   targetLang,
		},
	)
	if err != nil {
		return model.Topic{}, err
	}

	translatedContent, err := gtranslate.TranslateWithParams(
		topic.Content,
		gtranslate.TranslationParams{
			From: "zh",
			To:   targetLang,
		},
	)
	if err != nil {
		return model.Topic{}, err
	}

	newTopic := topic // Create a copy of the original topic

	newTopic.ID = 0 // Reset ID for the new topic
	newTopic.Title = translatedTitle
	newTopic.Content = translatedContent
	newTopic.Language = targetLang
	newTopic.EditTime = time.Now().Unix() // Set EditTime to translation time

	return newTopic, nil
}
