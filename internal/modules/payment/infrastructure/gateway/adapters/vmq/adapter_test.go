package vmqadapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/dujiao-next/internal/constants"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/shopspring/decimal"
)

func testConfig(gateway string) jsonmap.JSON {
	return jsonmap.JSON{"gateway_url": gateway, "merchant_key": "testkey123456",
		"notify_url": "https://shop.example.com/notify", "return_url": "https://shop.example.com/return"}
}

func TestFoxGoldenSignatures(t *testing.T) {
	values := url.Values{"payId": {"TEST20260314001"}, "param": {""}, "type": {"1"}, "price": {"1.00"},
		"reallyPrice": {"0.99"}, "notifyUrl": {"https://shop.example.com/notify"}, "returnUrl": {"https://shop.example.com/return"}}
	if got := createSign("testkey123456", values); got != "729a6c529b4a2ffed215d124a7e4244ed5d4981ba1982d4cd6f9a53b28de9263" {
		t.Fatalf("create signature differs from Fox/OpenSSL vector: %s", got)
	}
	if got := callbackSign("testkey123456", values); got != "0f21b9366a12396a71437d336a21fd7c5fe20b292e9b56d25b6b612a144daa60" {
		t.Fatalf("callback signature differs from Fox/OpenSSL vector: %s", got)
	}
}

func TestCreatePaymentUsesFoxAPIAndCashier(t *testing.T) {
	for channel, expectedType := range map[string]string{"wechat": "1", "alipay": "2"} {
		t.Run(channel, func(t *testing.T) {
			var received url.Values
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/order/create" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if err := r.ParseForm(); err != nil {
					t.Error(err)
				}
				received = r.PostForm
				if received.Get("sign") != createSign("testkey123456", received) {
					t.Error("invalid create signature")
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": map[string]any{
					"payId": received.Get("payId"), "orderId": "FOX-1", "publicToken": strings.Repeat("a", 64),
					"redirectUrl": "https://untrusted.example/redirect",
				}})
			}))
			defer server.Close()
			cfg := testConfig(server.URL + "/")
			cfg["notify_url"] = "https://shop.example.com/notify?source=卡密&mode=1"
			result, err := NewVmqAdapter().CreatePayment(context.Background(), cfg, paymentcontract.GatewayCreateInput{
				PaymentID: 42, OrderNo: "DJP-42", ChannelType: channel, Currency: "CNY",
				Amount:         money.FromDecimal(decimal.RequireFromString("10")),
				ReturnURLQuery: map[string]string{"order_no": "商城-42", "guest": "1"},
			})
			if err != nil {
				t.Fatal(err)
			}
			if received.Get("type") != expectedType || received.Get("price") != "10.00" || received.Get("param") != "42" {
				t.Fatalf("unexpected payment fields: %v", received)
			}
			if received.Get("notifyUrl") != cfg["notify_url"] || !strings.Contains(received.Get("returnUrl"), "guest=1") {
				t.Fatal("callback and return URLs were not preserved in signed request")
			}
			if result.RedirectURL != server.URL+"/#/payment/"+strings.Repeat("a", 64) || result.QRCodeURL != "" || result.ProviderRef != "FOX-1" {
				t.Fatalf("unexpected checkout result: %+v", result)
			}
		})
	}
}

func TestCreatePaymentRejectsInvalidResponsesAndRedirects(t *testing.T) {
	for name, body := range map[string]string{
		"legacy response":     `{"code":200,"data":{"orderId":"OLD"}}`,
		"legacy success code": `{"code":1,"data":{}}`,
		"gateway failure":     `{"code":400,"msg":"monitor offline"}`,
		"invalid JSON":        `<html>404</html>`,
		"invalid token":       `{"code":200,"data":{"payId":"DJP-1","orderId":"1","publicToken":"bad"}}`,
		"other order":         `{"code":200,"data":{"payId":"OTHER","orderId":"1","publicToken":"` + strings.Repeat("a", 64) + `"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer s.Close()
			_, err := NewVmqAdapter().CreatePayment(context.Background(), testConfig(s.URL), paymentcontract.GatewayCreateInput{
				PaymentID: 1, OrderNo: "DJP-1", ChannelType: "alipay", Currency: "CNY", Amount: money.FromDecimal(decimal.NewFromInt(1)),
			})
			if err == nil {
				t.Fatal("accepted invalid gateway response")
			}
		})
	}
	var forwarded bool
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { forwarded = true }))
	defer destination.Close()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer s.Close()
	_, err := NewVmqAdapter().CreatePayment(context.Background(), testConfig(s.URL), paymentcontract.GatewayCreateInput{
		OrderNo: "DJP-1", ChannelType: "alipay", Currency: "CNY", Amount: money.FromDecimal(decimal.NewFromInt(1)),
	})
	if err == nil || forwarded {
		t.Fatal("signed order followed a gateway redirect")
	}
}

func TestCallbackVerifiesAllFactsAndSettlesOriginalPrice(t *testing.T) {
	verifier := NewVmqAdapter().(paymentcontract.GatewayCallbackVerifier)
	valid := url.Values{"payId": {"DJP-42"}, "param": {"42"}, "type": {"2"}, "price": {"10.00"}, "reallyPrice": {"9.99"}}
	valid.Set("sign", callbackSign("testkey123456", valid))
	result, err := verifier.VerifyCallback(testConfig("https://vmq.example.com"), valid, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != constants.PaymentStatusSuccess || result.Amount.String() != "10.00" || result.Currency != "CNY" {
		t.Fatalf("incorrect settlement: %+v", result)
	}
	for field, value := range map[string]string{"payId": "OTHER", "param": "43", "type": "1", "price": "1.00", "reallyPrice": "10.00", "sign": strings.Repeat("0", 32)} {
		t.Run(field, func(t *testing.T) {
			changed := url.Values{}
			for k, v := range valid {
				changed[k] = append([]string(nil), v...)
			}
			changed.Set(field, value)
			if _, err := verifier.VerifyCallback(testConfig("https://vmq.example.com"), changed, nil); !errors.Is(err, paymentcontract.ErrGatewaySignatureInvalid) {
				t.Fatalf("tampering was accepted: %v", err)
			}
		})
	}
	valid["price"] = []string{"10.00", "0.01"}
	if _, err := verifier.VerifyCallback(testConfig("https://vmq.example.com"), valid, nil); err == nil {
		t.Fatal("duplicate signed field accepted")
	}
}

func TestVmqConfigRejectsInvalidURLsAndChannels(t *testing.T) {
	a := NewVmqAdapter()
	for _, field := range []string{"gateway_url", "notify_url", "return_url", "merchant_key"} {
		cfg := testConfig("https://vmq.example.com")
		cfg[field] = ""
		if err := a.ValidateConfig(cfg, "alipay"); err == nil {
			t.Fatalf("missing %s accepted", field)
		}
	}
	cfg := testConfig("https://vmq.example.com")
	if err := a.ValidateConfig(cfg, "qqpay"); err == nil {
		t.Fatal("unsupported payment type accepted")
	}
	cfg["gateway_url"] = "https://user:password@vmq.example.com"
	if err := a.ValidateConfig(cfg, "alipay"); err == nil {
		t.Fatal("embedded URL credentials accepted")
	}
}
