package router

import (
	"net/http"
	"strconv"
	"time"

	"github.com/CloudHammer/Seeds/src/models"
	"github.com/CloudHammer/Seeds/src/utils"
	"github.com/gin-gonic/gin"
)

type NodeRouter struct {
	routerAble
}

func getNodeFromParam(context *gin.Context) (models.SsNode, bool) {
	db := utils.GetMySQLInstance()

	nodeId := context.Param("id")
	var node models.SsNode
	err := db.Database.First(&node, "id = ?", nodeId).Error

	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{
			"ret":     0,
			"message": "Node not found",
		})
		return models.SsNode{}, true
	}
	return node, false
}

type InfoBody struct {
	Load   string `json:"load"`
	Uptime string `json:"uptime"`
}

func setInfo(context *gin.Context) {
	db := utils.GetMySQLInstance()
	node, notFound := getNodeFromParam(context)
	if notFound {
		return
	}
	var body InfoBody
	context.BindJSON(&body)
	uptime, err := strconv.ParseFloat(body.Uptime, 64)

	if err != nil {
		context.JSON(http.StatusUnprocessableEntity, gin.H{
			"ret": 0,
		})
		return
	}

	db.Database.Save(&models.SsNodeInfo{
		Load:    body.Load,
		Uptime:  uptime,
		NodeId:  node.Id,
		LogTime: int(time.Now().Unix()),
	})
	context.JSON(http.StatusOK, gin.H{
		"ret":  1,
		"data": "ok",
	})
}

func getInfo(context *gin.Context) {
	node, notFound := getNodeFromParam(context)
	if notFound {
		return
	}

	context.JSON(http.StatusOK, gin.H{
		"ret": 1,
		"data": gin.H{
			"server":          node.Server,
			"node_group":      node.NodeGroup,
			"node_class":      node.NodeClass,
			"node_speedlimit": node.NodeSpeedlimit,
			"traffic_rate":    node.TrafficRate,
			"mu_only":         node.MuOnly,
			"sort":            node.Sort,
		},
	})
}

func getAllInfo(context *gin.Context) {
	db := utils.GetMySQLInstance()
	var rawNodes []models.SsNode
	db.Database.Where("node_ip <> ?", nil).Where("sort = ?", 10).
		Or("sort = ?", 0).Find(&rawNodes)

	var nodes []gin.H
	for _, data := range rawNodes {
		nodes = append(nodes, gin.H{
			"node_ip": data.NodeIp,
		})
	}
	context.JSON(http.StatusOK, gin.H{
		"ret":  1,
		"data": nodes,
	})
}

func (NodeRouter) create(engine *gin.RouterGroup) {
	nodeGroup := engine.Group("/nodes")
	{
		nodeGroup.GET("", getAllInfo)
		nodeGroup.GET("/:id/info", getInfo)
		nodeGroup.POST("/:id/info", setInfo)
	}
}
