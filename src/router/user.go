package router

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/twinj/uuid"

	"github.com/CloudHammer/Seeds/src/utils"

	"github.com/CloudHammer/Seeds/src/models"
	"github.com/gin-gonic/gin"
)

type UserRouter struct {
	routerAble
}

var unableToProcessReq = gin.H{
	"ret":   0,
	"error": "Unable to process entity",
}

func getNodeFromQuery(context *gin.Context) (models.SsNode, bool) {
	db := utils.GetMySQLInstance()
	data, hasKey := context.GetQuery("node_id")

	if !hasKey {
		context.JSON(http.StatusUnprocessableEntity, unableToProcessReq)
		return models.SsNode{}, true
	}
	nodeId, err := strconv.ParseInt(data, 0, 64)

	if err != nil {
		context.JSON(http.StatusUnprocessableEntity, unableToProcessReq)
		return models.SsNode{}, true
	}

	var node models.SsNode
	err = db.Database.First(&node, "id = ?", nodeId).Error

	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{
			"ret":     0,
			"message": "Node not found",
		})
		return models.SsNode{}, true
	}
	return node, false
}

func getUserList(context *gin.Context) {
	db := utils.GetMySQLInstance()

	node, notFound := getNodeFromQuery(context)

	if notFound {
		return
	}

	// Heartbeat
	node.NodeHeartbeat = time.Now().Unix()
	db.Database.Save(&node)

	if node.NodeBandwidthLimit != 0 {
		if node.NodeBandwidthLimit < node.NodeBandwidth {
			context.JSON(http.StatusOK, gin.H{
				"ret":  1,
				"data": []gin.H{},
			})
			return
		}
	}
	var rawUsers []models.User
	query := db.Database.Where("class >= ?", node.NodeClass).
		Where("enable = ?", 1).Where("expire_in > ?", time.Now().Unix()).
		Or("is_admin = ?", 1)

	if node.NodeGroup != 0 {
		query = query.Where(models.User{NodeGroup: node.NodeGroup})
	}
	query.Find(&rawUsers)
	var users []gin.H

	for _, user := range rawUsers {
		users = append(users, gin.H{
			"id":              user.Id,
			"uuid":            uuid.NewV3(uuid.NameSpaceDNS, fmt.Sprintf("%d|%s", user.Id, user.Passwd)).String(),
			"method":          user.Method,
			"obfs":            user.Obfs,
			"obfs_param":      user.ObfsParam,
			"protocol":        user.Protocol,
			"protocol_param":  user.ProtocolParam,
			"forbidden_ip":    user.ForbiddenIp,
			"forbidden_port":  user.ForbiddenPort,
			"node_speedlimit": user.NodeSpeedlimit,
			"disconnect_ip":   user.DisconnectIp,
			"is_multi_user":   user.IsMultiUser,
			"port":            user.Port,
			"passwd":          user.Passwd,
			"u":               user.U,
			"d":               user.D,
			"transfer_enable": user.U + user.D,
		})
	}

	context.JSON(http.StatusOK, gin.H{
		"ret":  1,
		"data": users,
	})
}

type TrafficData struct {
	U      int64 `json:"u"`
	D      int64 `json:"d"`
	UserId int64 `json:"user_id"`
}

type TrafficDataJSON struct {
	Data []TrafficData `json:"data"`
}

func addTraffic(context *gin.Context) {
	db := utils.GetMySQLInstance()
	node, notFound := getNodeFromQuery(context)
	if notFound {
		return
	}

	var body TrafficDataJSON
	context.BindJSON(&body)

	var totalBandwidth int64 = 0

	for _, data := range body.Data {
		var user models.User
		err := db.Database.First(&user, "id = ?", data.UserId).Error
		if err != nil {
			continue
		}
		user.T = int(time.Now().Unix())
		user.U += int64(float64(data.U) * node.TrafficRate)
		user.D += int64(float64(data.D) * node.TrafficRate)
		totalBandwidth += data.U + data.D
		db.Database.Save(&user)

		db.Database.Save(&models.UserTrafficLog{
			UserId:  user.Id,
			U:       data.U,
			D:       data.D,
			NodeId:  node.Id,
			Rate:    node.TrafficRate,
			Traffic: strconv.Itoa(int(float64(data.U+data.D) * node.TrafficRate)),
			LogTime: int(time.Now().Unix()),
		})
	}
	node.NodeBandwidth += totalBandwidth
	db.Database.Save(&node)

	db.Database.Save(&models.SsNodeOnlineLog{
		NodeId:     node.Id,
		OnlineUser: len(body.Data),
		LogTime:    int(time.Now().Unix()),
	})

	context.JSON(http.StatusOK, gin.H{
		"ret":  1,
		"data": "ok",
	})
}

type AliveData struct {
	Ip     string `json:"ip"`
	UserId int    `json:"user_id"`
}

type AliveDataJSON struct {
	Data []AliveData `json:"data"`
}

func addAliveIp(context *gin.Context) {
	db := utils.GetMySQLInstance()
	node, notFound := getNodeFromQuery(context)
	if notFound {
		return
	}
	var body AliveDataJSON
	context.BindJSON(&body)

	for _, data := range body.Data {
		db.Database.Save(&models.AliveIp{
			UserId:   data.UserId,
			Ip:       data.Ip,
			NodeId:   node.Id,
			Datetime: time.Now().Unix(),
		})
	}
	context.JSON(http.StatusOK, gin.H{
		"ret":  1,
		"data": "ok",
	})
}

type DetectData struct {
	ListId int64 `json:"list_id"`
	UserId int64 `json:"user_id"`
}

type DetectDataJSON struct {
	Data []DetectData `json:"data"`
}

func addDetectLog(context *gin.Context) {
	db := utils.GetMySQLInstance()
	node, notFound := getNodeFromQuery(context)
	if notFound {
		return
	}
	var body DetectDataJSON
	context.BindJSON(&body)

	for _, data := range body.Data {
		db.Database.Save(&models.DetectLog{
			UserId:   data.UserId,
			ListId:   data.ListId,
			NodeId:   node.Id,
			Datetime: time.Now().Unix(),
		})
	}

	context.JSON(http.StatusOK, gin.H{
		"ret":  1,
		"data": "ok",
	})
}

func (UserRouter) create(engine *gin.RouterGroup) {
	userGroup := engine.Group("/users")
	{
		userGroup.GET("", getUserList)
		userGroup.POST("/traffic", addTraffic)
		userGroup.POST("/aliveip", addAliveIp)
		userGroup.POST("/detectlog", addDetectLog)
	}
}
