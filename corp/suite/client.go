// @description wechat 是腾讯微信公众平台 api 的 golang 语言封装
// @link        https://github.com/chanxuehong/wechat for the canonical source repository
// @license     https://github.com/chanxuehong/wechat/blob/master/LICENSE
// @authors     chanxuehong(chanxuehong@gmail.com)

// +build !wechatdebug

package suite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"

	"github.com/chanxuehong/wechat/corp"
)

type Client struct {
	SuiteId string
	AccessTokenServer
	HttpClient *http.Client
}

// 创建一个新的 Client.
//  如果 HttpClient == nil 则默认用 http.DefaultClient
func NewClient(SuiteId string, AccessTokenServer AccessTokenServer, HttpClient *http.Client) *Client {
	if SuiteId == "" {
		panic("empty SuiteId")
	}
	if AccessTokenServer == nil {
		panic("AccessTokenServer == nil")
	}
	if HttpClient == nil {
		HttpClient = http.DefaultClient
	}

	return &Client{
		SuiteId:           SuiteId,
		AccessTokenServer: AccessTokenServer,
		HttpClient:        HttpClient,
	}
}

// 用 encoding/json 把 request marshal 为 JSON, 放入 http 请求的 body 中,
// POST 到微信服务器, 然后将微信服务器返回的 JSON 用 encoding/json 解析到 response.
//
//  NOTE:
//  1. 一般不用调用这个方法, 请直接调用高层次的封装方法;
//  2. 最终的 URL == incompleteURL + suite_access_token;
//  3. response 格式有要求, 要么是 *corp.Error, 要么是下面结构体的指针(注意 Error 必须是第一个 Field):
//      struct {
//          corp.Error
//          ...
//      }
func (clt *Client) PostJSON(incompleteURL string, request interface{}, response interface{}) (err error) {
	buf := textBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer textBufferPool.Put(buf)

	if err = json.NewEncoder(buf).Encode(request); err != nil {
		return
	}
	requestBytes := buf.Bytes()

	token, err := clt.Token()
	if err != nil {
		return
	}

	hasRetried := false
RETRY:
	finalURL := incompleteURL + url.QueryEscape(token)

	httpResp, err := clt.HttpClient.Post(finalURL, "application/json; charset=utf-8", bytes.NewReader(requestBytes))
	if err != nil {
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("http.Status: %s", httpResp.Status)
	}

	if err = json.NewDecoder(httpResp.Body).Decode(response); err != nil {
		return
	}

	var ErrorStructValue reflect.Value // corp.Error

	// 下面的代码对 response 有特定要求, 见此函数 NOTE
	responseStructValue := reflect.ValueOf(response).Elem()
	if v := responseStructValue.Field(0); v.Kind() == reflect.Struct {
		ErrorStructValue = v
	} else {
		ErrorStructValue = responseStructValue
	}

	switch ErrCode := ErrorStructValue.Field(0).Int(); ErrCode {
	case corp.ErrCodeOK:
		return
	case corp.ErrCodeSuiteAccessTokenExpired:
		ErrMsg := ErrorStructValue.Field(1).String()
		corp.LogInfoln("[WECHAT_RETRY] err_code:", ErrCode, ", err_msg:", ErrMsg)
		corp.LogInfoln("[WECHAT_RETRY] current token:", token)

		if !hasRetried {
			hasRetried = true

			if token, err = clt.TokenRefresh(); err != nil {
				return
			}
			corp.LogInfoln("[WECHAT_RETRY] new token:", token)

			responseStructValue.Set(reflect.New(responseStructValue.Type()).Elem())
			goto RETRY
		}
		corp.LogInfoln("[WECHAT_RETRY] fallthrough, current token:", token)
		fallthrough
	default:
		return
	}
}

// GET 微信资源, 然后将微信服务器返回的 JSON 用 encoding/json 解析到 response.
//
//  NOTE:
//  1. 一般不用调用这个方法, 请直接调用高层次的封装方法;
//  2. 最终的 URL == incompleteURL + suite_access_token;
//  3. response 格式有要求, 要么是 *corp.Error, 要么是下面结构体的指针(注意 Error 必须是第一个 Field):
//      struct {
//          corp.Error
//          ...
//      }
func (clt *Client) GetJSON(incompleteURL string, response interface{}) (err error) {
	token, err := clt.Token()
	if err != nil {
		return
	}

	hasRetried := false
RETRY:
	finalURL := incompleteURL + url.QueryEscape(token)

	httpResp, err := clt.HttpClient.Get(finalURL)
	if err != nil {
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("http.Status: %s", httpResp.Status)
	}

	if err = json.NewDecoder(httpResp.Body).Decode(response); err != nil {
		return
	}

	var ErrorStructValue reflect.Value // corp.Error

	// 下面的代码对 response 有特定要求, 见此函数 NOTE
	responseStructValue := reflect.ValueOf(response).Elem()
	if v := responseStructValue.Field(0); v.Kind() == reflect.Struct {
		ErrorStructValue = v
	} else {
		ErrorStructValue = responseStructValue
	}

	switch ErrCode := ErrorStructValue.Field(0).Int(); ErrCode {
	case corp.ErrCodeOK:
		return
	case corp.ErrCodeSuiteAccessTokenExpired:
		ErrMsg := ErrorStructValue.Field(1).String()
		corp.LogInfoln("[WECHAT_RETRY] err_code:", ErrCode, ", err_msg:", ErrMsg)
		corp.LogInfoln("[WECHAT_RETRY] current token:", token)

		if !hasRetried {
			hasRetried = true

			if token, err = clt.TokenRefresh(); err != nil {
				return
			}
			corp.LogInfoln("[WECHAT_RETRY] new token:", token)

			responseStructValue.Set(reflect.New(responseStructValue.Type()).Elem())
			goto RETRY
		}
		corp.LogInfoln("[WECHAT_RETRY] fallthrough, current token:", token)
		fallthrough
	default:
		return
	}
}
