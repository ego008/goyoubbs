package model

// TopicRemotePostForm 远程发布帖子、评论
type TopicRemotePostForm struct {
	TopicId  uint64
	NodeId   uint64
	UserName string
	Title    string
	Content  string
}
