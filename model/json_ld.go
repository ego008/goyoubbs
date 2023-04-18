package model

type (
	JsonLd struct {
		Context string        `json:"@context"`
		Graph   []interface{} `json:"@graph"`
	}

	JsOrganization struct {
		Type string `json:"@type"`
		Logo string `json:"logo"`
		URL  string `json:"url"`
	}

	JsItemListElement struct {
		Type     string `json:"@type"`
		Position int    `json:"position"`
		Name     string `json:"name"`
		Item     string `json:"item"`
	}
	JsBreadcrumbList struct {
		Type            string              `json:"@type"`
		ItemListElement []JsItemListElement `json:"itemListElement"`
	}

	JsAuthor struct {
		Type string `json:"@type"`
		Name string `json:"name"`
		URL  string `json:"url"`
	}

	JsLogo struct {
		Type string `json:"@type"`
		URL  string `json:"url"`
	}

	JsPublisher struct {
		Type string `json:"@type"`
		Name string `json:"name"`
		Logo JsLogo `json:"logo"`
	}

	JsSpeakable struct {
		Type  string   `json:"@type"`
		Xpath []string `json:"xpath"`
	}

	JsCommentAuthor struct {
		Type string `json:"@type"` // Person
		Name string `json:"name"`
		Url  string `json:"url"`
	}

	JsComment struct {
		Type        string          `json:"@type"` // Comment
		Url         string          `json:"url"`   // https://youbbs.org/t/3272#3
		Text        string          `json:"text"`
		DateCreated string          `json:"dateCreated"`
		Name        string          `json:"name"` // 1 // #id 楼层
		Author      JsCommentAuthor `json:"author"`
		Publisher   JsCommentAuthor `json:"publisher"`
	}

	JsArticle struct {
		Type             string      `json:"@type"`
		DateModified     string      `json:"dateModified"`
		DatePublished    string      `json:"datePublished"`
		Headline         string      `json:"headline"`
		Image            []string    `json:"image"`
		Author           JsAuthor    `json:"author"`
		Publisher        JsPublisher `json:"publisher"`
		Description      string      `json:"description"`
		MainEntityOfPage string      `json:"mainEntityOfPage"`
		Speakable        JsSpeakable `json:"speakable"`
		// comment
		CommentCount int         `json:"commentCount"`
		Comment      []JsComment `json:"comment"`
	}
)
