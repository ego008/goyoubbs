package admin

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

	CommentEdit struct {
		BasePage
		ReadMoreBreak  string
		DefaultTopic   model.Topic // 编辑/添加
		DefaultUser    model.User  // 默认作者
		DefaultComment model.CommentFmt
		GoBack         bool // 返回到编辑前页面
	}

	Link struct {
		BasePage
		Act  string // 行为名称，添加/编辑
		Link model.Link
	}

	Node struct {
		BasePage
		Act  string     // 行为名称，添加/编辑
		Node model.Node // 分区
	}

	SiteConfig struct {
		BasePage
		SiteConf model.SiteConf
	}

	SiteRouter struct {
		BasePage
		TypeLst []string
		ObjLst  []model.CustomRouter
		Obj     model.CustomRouter
	}

	TopicAdd struct {
		BasePage
		ReadMoreBreak string
		DefaultTopic  model.Topic  // 编辑/添加
		DefaultUser   model.User   // 默认作者
		UserLst       []model.User // 可选发表用户列表，管理员
		GoBack        bool         // 返回到编辑前页面
	}

	User struct {
		BasePage
		Act     string // 行为名称，添加/编辑
		User    model.User
		UserLst []model.User
		FlagLst []model.Flag
	}

	IpLookup struct {
		BasePage
		Limit    int
		KeyStart string
		ShowNext bool
		Items    []model.KvStr
	}

	RateLimitSetting struct {
		BasePage
		MyIp       string
		SettingLst []model.SettingKv
	}
)
