package paymentcallbackhttp

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/constants"
	ginutil "github.com/dujiao-next/internal/platform/http/ginutil"
	"github.com/gin-gonic/gin"
)

func (h *Handler) handleVmqCallback(c *gin.Context) bool {
	form, err := parseCallbackForm(c)
	if err != nil || getFirstValue(form, "payId") == "" {
		return false
	}
	fail := func() bool {
		c.String(http.StatusOK, "fail")
		return true
	}
	payment, err := h.payments.GetByGatewayOrderNo(getFirstValue(form, "payId"))
	if err != nil || payment == nil {
		return fail()
	}
	channel, err := h.channels.GetByID(payment.ChannelID)
	if err != nil || channel == nil || channel.ProviderType != constants.PaymentProviderVmq {
		return fail()
	}
	expectedType := ""
	switch strings.ToLower(strings.TrimSpace(channel.ChannelType)) {
	case constants.PaymentChannelTypeWechat:
		expectedType = "1"
	case constants.PaymentChannelTypeAlipay:
		expectedType = "2"
	}
	if expectedType == "" || getFirstValue(form, "type") != expectedType ||
		getFirstValue(form, "param") != strconv.FormatUint(uint64(payment.ID), 10) {
		return fail()
	}
	if _, err := h.service.HandleSyncCallback(channel, form, nil); err != nil {
		ginutil.RequestLog(c).Warnw("vmq_callback_failed", "payment_id", payment.ID, "channel_id", channel.ID, "error", err)
		return fail()
	}
	c.String(http.StatusOK, "success")
	return true
}
