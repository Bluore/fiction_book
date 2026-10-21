package handler

import (
	"context"
	"errors"

	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/pkg/response"
	"fiction-book/pkg/transmit"

	"github.com/gin-gonic/gin"
)

// GetOrderByID 获取用户订单
// @Summary 获取用户订单
// @Description 根据订单ID获取用户订单详细信息；订单待付款时使用长轮询，服务端最长保持连接 30 秒，期间每秒检查一次，订单状态变化或超时后返回最新订单
// @Tags 订单管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "订单ID"
// @Success 200 {object} response.Response{data=dto.OrderResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 4020 {object} response.Response "缺失对应权限"
// @Failure 4021 {object} response.Response "缺失对应权限"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/orders/{id} [get]
func (h *Handler) GetOrderByID(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetOrderRequest](c)
	userId := transmit.GetStr(c, "userID")
	if userId == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	out, err := h.orderService.GetOrder(c.Request.Context(), userId, req.ID)
	if err != nil {
		// 客户端断开长连接时无需返回错误响应
		if errors.Is(err, context.Canceled) {
			return
		}
		response.PushError(c, err)
		return
	}
	response.Success(c, out)
}

// GetOrderList 获取用户订单列表
// @Summary 获取用户订单列表
// @Description 获取当前用户的订单列表（分页）
// @Tags 订单管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} response.Response{data=dto.GetOrderListResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/orders [get]
func (h *Handler) GetOrderList(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetOrderListRequest](c)
	req.ToStandard()
	userId := transmit.GetStr(c, "userID")
	if userId == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	out, err := h.orderService.GetOrderListByUserID(c.Request.Context(), userId, req)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, out)
}

// CreateOrder 创建用户充值订单
// @Summary 创建用户充值订单
// @Description 创建一个金币充值订单（1元=100金币）
// @Tags 订单管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body dto.CreateOrderRequest true "订单信息"
// @Success 200 {object} response.Response{data=dto.OrderResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Failure 4017 {object} response.Response "价格计算校对失败"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/orders/create/gold [post]
func (h *Handler) CreateOrder(c *gin.Context) {
	req := middleware.MustGetRequest[dto.CreateOrderRequest](c)
	userId := transmit.GetStr(c, "userID")
	if userId == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	out, err := h.orderService.CreateOrder(c.Request.Context(), userId, req)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, out)
}
