package model

// 页面基本信息
type (
	BasePage struct {
		SiteCf         *SiteConf
		CurrentUser    User
		Title          string
		Breadcrumbs    string
		Keywords       string
		Description    string
		Canonical      string
		Authorized     bool   // 合法的，已登录
		PageName       string // index/post_add/post_detail/...
		HasMsg         bool   // 有站内信息
		HasTopicReview bool   // 有帖子要审核
		HasReplyReview bool   // 有评论要审核
		ShowAutoAd     bool

		ShowPostTopAd bool
		ShowPostBotAd bool
		ShowSideAd    bool
		//TopRate       []model.ArticleSimple
		//RecentLst     []model.ArticleSimple
		CloseSidebar  bool          // 关闭边栏
		TagCloud      []TagFontSize // 边栏 tag cloud
		JsonLd        string
		NodeLst       []Node       // 边栏 分类
		RangeTopicLst []TopicLi    // 边栏显示最近被浏览的文章
		RecentComment []CommentFmt // 边栏最近评论内容
		LinkLst       []Link       // 边栏 链接
		SiteInfo      SiteInfo     // 边栏 站点信息
		DefaultNode   Node         // 默认发帖节点，当前文章所属的分类
	}

	// 通用响应信息
	NormalRsp struct {
		Code int
		Msg  string
	}

	// 首页、节点、tag、搜索 的文章列表
	TopicLstPage struct {
		BasePage
		Q             string
		Tag           string
		TopicPageInfo TopicPageInfo
	}

	// 文章详情页
	TopicDetailPage struct {
		BasePage
		TopicFmt   TopicFmt
		NewTopic   TopicLi       // 新一篇文章
		OldTopic   TopicLi       // 旧一篇文章
		TagLst     []TagFontSize // tags
		CommentLst []CommentFmt  // 评论列表
	}

	// admin
	AdminBasePage struct {
		BasePage
	}
)
