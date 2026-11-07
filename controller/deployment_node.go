package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// GetDeploymentNodes 返回内部（私有化部署）模型的部署节点列表。
// 仅管理员可访问（路由层 middleware.AdminAuth() 已拦截）。
// tag 默认 "私有化部署"，对应 model_meta.tags 的过滤标签。
func GetDeploymentNodes(c *gin.Context) {
	tag := c.DefaultQuery("tag", "私有化部署")
	nodes, err := model.GetDeploymentNodes(tag)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "获取部署节点失败：" + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    nodes,
	})
}
