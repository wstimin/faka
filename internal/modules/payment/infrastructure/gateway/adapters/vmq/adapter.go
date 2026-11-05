package vmqadapter

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/constants"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	gatewaycommon "github.com/dujiao-next/internal/modules/payment/infrastructure/gateway/common"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/shopspring/decimal"
)

type adapter struct{}

func NewVmqAdapter() paymentcontract.GatewayProvider { return &adapter{} }

var (
	_ paymentcontract.GatewayProvider         = (*adapter)(nil)
	_ paymentcontract.GatewayCallbackVerifier = (*adapter)(nil)
)

func (a *adapter) Type() string { return constants.PaymentProviderVmq + ":" }

type config struct {
	gatewayURL string
	key        string
	notifyURL  string
	returnURL  string
}

func parseConfig(raw jsonmap.JSON) (config, error) {
	cfg := config{
		gatewayURL: strings.TrimRight(gatewaycommon.ReadString(raw, "gateway_url"), "/"),
		key:        gatewaycommon.ReadString(raw, "merchant_key"),
		notifyURL:  gatewaycommon.ReadString(raw, "notify_url"),
		returnURL:  gatewaycommon.ReadString(raw, "return_url"),
	}
	if cfg.key == "" {
		return cfg, fmt.Errorf("%w: merchant_key is required", paymentcontract.ErrGatewayConfigInvalid)
	}
	for name, value := range map[string]string{"gateway_url": cfg.gatewayURL, "notify_url": cfg.notifyURL, "return_url": cfg.returnURL} {
		u, err := url.Parse(value)
		if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Fragment != "" {
			return cfg, fmt.Errorf("%w: invalid %s", paymentcontract.ErrGatewayConfigInvalid, name)
		}
		if name == "gateway_url" && u.RawQuery != "" {
			return cfg, fmt.Errorf("%w: gateway_url must not contain a query", paymentcontract.ErrGatewayConfigInvalid)
		}
	}
	return cfg, nil
}

func paymentType(channel string) string {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case constants.PaymentChannelTypeWechat:
		return "1"
	case constants.PaymentChannelTypeAlipay:
		return "2"
	default:
		return ""
	}
}

func (a *adapter) ValidateConfig(raw jsonmap.JSON, channelType string) error {
	if paymentType(channelType) == "" {
		return fmt.Errorf("%w: unsupported VMQ payment type", paymentcontract.ErrGatewayConfigInvalid)
	}
	_, err := parseConfig(raw)
	return err
}

func sign(key, text string) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(text))
	return hex.EncodeToString(mac.Sum(nil))
}

func createSign(key string, values url.Values) string {
	return sign(key, "payId="+values.Get("payId")+"&param="+values.Get("param")+
		"&type="+values.Get("type")+"&price="+values.Get("price")+
		"&notifyUrl="+values.Get("notifyUrl")+"&returnUrl="+values.Get("returnUrl"))
}

func callbackSign(key string, values url.Values) string {
	return sign(key, "payId="+values.Get("payId")+"&param="+values.Get("param")+
		"&type="+values.Get("type")+"&price="+values.Get("price")+"&reallyPrice="+values.Get("reallyPrice"))
}

func (a *adapter) CreatePayment(ctx context.Context, raw jsonmap.JSON, input paymentcontract.GatewayCreateInput) (*paymentcontract.GatewayCreateResult, error) {
	cfg, err := parseConfig(raw)
	if err != nil {
		return nil, err
	}
	typeText := paymentType(input.ChannelType)
	if typeText == "" || !strings.EqualFold(input.Currency, "CNY") || strings.TrimSpace(input.OrderNo) == "" || !input.Amount.Decimal.IsPositive() {
		return nil, fmt.Errorf("%w: VMQ requires a CNY order and WeChat/Alipay", paymentcontract.ErrGatewayConfigInvalid)
	}
	if mode := gatewaycommon.ReadString(input.Extra, "interaction_mode"); mode != "" && mode != constants.PaymentInteractionRedirect {
		return nil, fmt.Errorf("%w: VMQ requires redirect mode", paymentcontract.ErrGatewayConfigInvalid)
	}
	returnURL := gatewaycommon.PickFirstNonEmpty(input.ReturnURL, cfg.returnURL)
	values := url.Values{
		"payId": {input.OrderNo}, "param": {strconv.FormatUint(uint64(input.PaymentID), 10)},
		"type": {typeText}, "price": {input.Amount.Decimal.StringFixed(2)},
		"notifyUrl": {gatewaycommon.PickFirstNonEmpty(input.NotifyURL, cfg.notifyURL)},
		"returnUrl": {gatewaycommon.AppendQueryParams(returnURL, input.ReturnURLQuery)},
		"isHtml":    {"0"},
	}
	values.Set("sign", createSign(cfg.key, values))
	ctx, cancel := gatewaycommon.WithDefaultTimeout(ctx)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.gatewayURL+"/api/order/create", strings.NewReader(values.Encode()))
	if err != nil {
		return nil, paymentcontract.ErrGatewayRequestFailed
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// A redirect must never forward a signed order to another host or silently turn POST into GET.
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: VMQ request failed", paymentcontract.ErrGatewayRequestFailed)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: VMQ HTTP %d", paymentcontract.ErrGatewayRequestFailed, response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return nil, paymentcontract.ErrGatewayResponseInvalid
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			PayID       string `json:"payId"`
			OrderID     string `json:"orderId"`
			PublicToken string `json:"publicToken"`
		} `json:"data"`
	}
	if json.Unmarshal(data, &result) != nil {
		return nil, paymentcontract.ErrGatewayResponseInvalid
	}
	if result.Code != 200 {
		return nil, fmt.Errorf("%w: VMQ code %d: %.200s", paymentcontract.ErrGatewayRequestFailed, result.Code, result.Msg)
	}
	tokenBytes, tokenErr := hex.DecodeString(result.Data.PublicToken)
	if tokenErr != nil || len(tokenBytes) != 32 || result.Data.PayID != input.OrderNo || result.Data.OrderID == "" {
		return nil, paymentcontract.ErrGatewayResponseInvalid
	}
	// Use the gateway's cashier to display reallyPrice; it can differ from the original order amount.
	return &paymentcontract.GatewayCreateResult{
		ProviderRef: result.Data.OrderID,
		RedirectURL: cfg.gatewayURL + "/#/payment/" + result.Data.PublicToken,
		AmountSent:  values.Get("price"), CurrencySent: "CNY",
		Payload: jsonmap.JSON{"payId": input.OrderNo, "orderId": result.Data.OrderID},
	}, nil
}

func (a *adapter) VerifyCallback(raw jsonmap.JSON, form map[string][]string, _ []byte) (*paymentcontract.GatewayCallbackResult, error) {
	cfg, err := parseConfig(raw)
	if err != nil {
		return nil, err
	}
	values := url.Values(form)
	for _, field := range []string{"payId", "param", "type", "price", "reallyPrice", "sign"} {
		if len(values[field]) != 1 {
			return nil, paymentcontract.ErrGatewaySignatureInvalid
		}
	}
	if values.Get("payId") == "" || (values.Get("type") != "1" && values.Get("type") != "2") {
		return nil, paymentcontract.ErrGatewaySignatureInvalid
	}
	if !hmac.Equal([]byte(values.Get("sign")), []byte(callbackSign(cfg.key, values))) {
		return nil, paymentcontract.ErrGatewaySignatureInvalid
	}
	price, err := decimal.NewFromString(values.Get("price"))
	if err != nil || !price.IsPositive() || price.StringFixed(2) != values.Get("price") {
		return nil, paymentcontract.ErrGatewayResponseInvalid
	}
	reallyPrice, err := decimal.NewFromString(values.Get("reallyPrice"))
	if err != nil || !reallyPrice.IsPositive() || reallyPrice.StringFixed(2) != values.Get("reallyPrice") {
		return nil, paymentcontract.ErrGatewayResponseInvalid
	}
	// Fox signs price (original order amount) and reallyPrice (amount allocated to identify the receipt).
	// Business settlement must use price; treating a 0.01 adjustment as underpayment would block delivery.
	return &paymentcontract.GatewayCallbackResult{
		OrderNo: values.Get("payId"), Status: constants.PaymentStatusSuccess,
		Amount: money.FromDecimal(price), Currency: "CNY",
		Payload: jsonmap.JSON{"payId": values.Get("payId"), "param": values.Get("param"),
			"type": values.Get("type"), "price": values.Get("price"), "reallyPrice": values.Get("reallyPrice")},
	}, nil
}
