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

/*
{
  "@context": "http://schema.org/",
  "@graph":[
    {
      "@type": "Organization",
      "logo": "https://www.ijd8.com/logo.jpg",    //LOGO图片地址，必须是112x112，.jpg,.png,.gif
      "url": "https://www.ijd8.com"               //与Logo相关联的url
    },
    {
      "@type": "BreadcrumbList",
       "itemListElement": [{
          "@type": "ListItem",
          "position": 1,
          "name": "图书",
          "item": "https://www.ijd8.com/图书"
       },{
          "@type": "ListItem",
          "position": 2,
          "name": "小说",
          "item": "https://www.ijd8.com/图书/小说"
       }]
    },
    {
      "@type": "Article",
      "dateModified": "2015-02-05T08:00:00+08:00",
      "datePublished": "2015-02-05T08:00:00+08:00",
      "headline": "标题，不超过110个字符",
      "image": [                 //提供三张不同比例的高清图片， 长x宽>=300 000
        "https://example.com/photos/1x1/photo.jpg",  //至少：600*600 = 360 000
        "https://example.com/photos/4x3/photo.jpg",  //至少：800*600 = 480 000
        "https://example.com/photos/16x9/photo.jpg"  //至少：960*540 = 518 400
      ],
      "author": {
        "@type": "Person",
        "name": "李XX"    //作者名称
      },
      "publisher": {
         "@type": "Organization",
         "name": "ijd8博客",                          //发布机构名称
         "logo": {
           "@type": "ImageObject",
           "url": "https://www.ijd8.com/logo.jpg"    //发布机构Logo,遵循
         }
      },
      "description": "内容描述",
      "mainEntityOfPage": "canonical URL of the article page",   //网页权威链接，无重复网页就设置成当前页地址
      "speakable": {
        "@type": "SpeakableSpecification",
        "xpath": [
          "/html/head/title",                              //指向head中的title
          "/html/head/meta[@name='description']/@content"  //指向head中的description
         ]
      }
    }
  ]
 }
*/
