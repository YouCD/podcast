package handlers

import (
	"podcast/pkg/dgraph"

	"github.com/gin-gonic/gin"
)

func Graph(c *gin.Context) {
	d, err := dgraph.New()
	if err != nil {
		ErrorWithMessage(c, "dgraph.New() error")
		return
	}
	defer func() {
		d.Close()
	}()
	dql := `
# 1. 找到所有“企业”类型的节点
{
  var(func: type("企业")) {
    u as uid
  }
  # 2. 找到所有“技术”类型的节点
  var(func: type("技术")) {
    t as uid
  }
  # 3. 查询这些节点之间的“研发”关系
  q(func: uid(u)) {
    uid
    name
    dgraph.type
    <研发> @filter(uid(t)) {
       uid
       name
       dgraph.type
    }
  }
}
`
	query, err := d.Query(c.Request.Context(), dql)
	if err != nil {
		ErrorWithMessage(c, "d.Query() error")
		return
	}
	Success(c, query)
}
