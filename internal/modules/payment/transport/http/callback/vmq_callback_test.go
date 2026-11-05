package paymentcallbackhttp

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/dujiao-next/internal/constants"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	"github.com/gin-gonic/gin"
)

type vmqCallbackStub struct {
	payment paymentdomain.Payment
	channel paymentdomain.PaymentChannel
	err     error
	calls   int
}

func (s *vmqCallbackStub) GetByGatewayOrderNo(no string) (*paymentdomain.Payment, error) {
	if no != s.payment.GatewayOrderNo {
		return nil, nil
	}
	return &s.payment, nil
}
func (s *vmqCallbackStub) GetLatestByProviderRef(string) (*paymentdomain.Payment, error) {
	return nil, nil
}
func (s *vmqCallbackStub) GetByID(id uint) (*paymentdomain.PaymentChannel, error) {
	if id != s.channel.ID {
		return nil, nil
	}
	return &s.channel, nil
}
func (s *vmqCallbackStub) HandleSyncCallback(*paymentdomain.PaymentChannel, map[string][]string, []byte) (*paymentdomain.Payment, error) {
	s.calls++
	return &s.payment, s.err
}
func (s *vmqCallbackStub) HandleWechatWebhook(WechatWebhookInput) (*paymentdomain.Payment, string, error) {
	return nil, "", errors.New("unexpected WeChat callback")
}

func TestVmqCallbackRoutesGetAndPostAndAcknowledgesOnlyAfterProcessing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		for _, failure := range []string{"", "verification", "other provider", "wrong payment type", "wrong payment id", "unknown order"} {
			t.Run(method+"/"+failure, func(t *testing.T) {
				stub := &vmqCallbackStub{
					payment: paymentdomain.Payment{ID: 42, ChannelID: 3, GatewayOrderNo: "DJP-42"},
					channel: paymentdomain.PaymentChannel{ID: 3, ProviderType: constants.PaymentProviderVmq, ChannelType: "alipay"},
				}
				values := url.Values{"payId": {"DJP-42"}, "param": {"42"}, "type": {"2"}, "price": {"10.00"}, "reallyPrice": {"9.99"}, "sign": {"signed"}}
				switch failure {
				case "verification":
					stub.err = errors.New("invalid signature")
				case "other provider":
					stub.channel.ProviderType = constants.PaymentProviderEpay
				case "wrong payment type":
					values.Set("type", "1")
				case "wrong payment id":
					values.Set("param", "43")
				case "unknown order":
					values.Set("payId", "OTHER")
				}
				target := "/api/v1/payments/callback"
				body := ""
				if method == http.MethodGet {
					target += "?" + values.Encode()
				} else {
					body = values.Encode()
				}
				req := httptest.NewRequest(method, target, strings.NewReader(body))
				if method == http.MethodPost {
					req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				}
				w := httptest.NewRecorder()
				r := gin.New()
				RegisterRoutes(r.Group("/api/v1"), NewHandler(stub, stub, stub, nil))
				r.ServeHTTP(w, req)
				want := "fail"
				if failure == "" {
					want = "success"
				}
				if w.Code != http.StatusOK || w.Body.String() != want {
					t.Fatalf("got %d %q, want 200 %s", w.Code, w.Body.String(), want)
				}
				if failure != "" && failure != "verification" && stub.calls != 0 {
					t.Fatal("invalid callback reached payment processing")
				}
			})
		}
	}
}
