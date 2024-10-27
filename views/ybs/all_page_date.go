package ybs

import "goyoubbs/model"

type (
	//BasePage 页面基本信息
	BasePage struct {
		SiteCf         *model.SiteConf
		CurrentUser    model.User
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
		CloseSidebar  bool                // 关闭边栏
		TagCloud      []model.TagFontSize // 边栏 tag cloud
		JsonLd        string
		NodeLst       []model.Node       // 边栏 分类
		RangeTopicLst []model.TopicLi    // 边栏显示最近被浏览的文章
		RecentComment []model.CommentFmt // 边栏最近评论内容
		LinkLst       []model.Link       // 边栏 链接
		SiteInfo      model.SiteInfo     // 边栏 站点信息
		DefaultNode   model.Node         // 默认发帖节点，当前文章所属的分类
	}

	//NormalRsp 通用响应信息
	NormalRsp struct {
		Code int
		Msg  string
	}

	//TopicLstPage 首页、节点、tag、搜索 的文章列表
	TopicLstPage struct {
		BasePage
		Q             string
		Tag           string
		TopicPageInfo model.TopicPageInfo
	}

	//TopicDetailPage 文章详情页
	TopicDetailPage struct {
		BasePage
		ReadMoreBreak string
		TopicFmt      model.TopicFmt
		NewTopic      model.TopicLi       // 新一篇文章
		OldTopic      model.TopicLi       // 旧一篇文章
		TagLst        []model.TagFontSize // tags
		CommentLst    []model.CommentFmt  // 评论列表
	}

	HomePage struct {
		TopicLstPage
	}

	MemberPage struct {
		TopicLstPage
		UserFmt          model.UserFmt
		LstType          string
		TitleText        string
		CommentReviewLst []model.CommentReview // 待评论信息
		TopicLst         []model.TopicRecForm  // 待审核帖子列表
	}

	MyMsg struct {
		BasePage
		TopicPageInfo model.TopicPageInfoMsg
	}

	NodePage struct {
		TopicLstPage
	}

	SearchPage struct {
		TopicLstPage
	}

	TagPage struct {
		TopicLstPage
	}

	UserTopicAdd struct {
		BasePage
		ReadMoreBreak string
		DefaultTopic  model.Topic  // 编辑/添加
		DefaultUser   model.User   // 默认作者
		UserLst       []model.User // 可选发表用户列表，管理员
	}

	UserLogin struct {
		BasePage
		Act          string
		Token        string
		CaptchaId    string
		HasOtherAuth bool
		DefaultName  string
	}

	UserSetting struct {
		BasePage
		User model.User
	}
)
