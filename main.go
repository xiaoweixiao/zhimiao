package main

import (
	"fmt"
	"github.com/robertkrimen/otto"
	"github.com/roseboy/httpcase/requests"
	"github.com/roseboy/httpcase/util"
	"log"
	url2 "net/url"
	"strings"
	"time"
)

var (
	http                = requests.NewHttpSession()
	Cookie              = "ASP.NET_SessionId=mmzuzank0nifzzw0wiwkikkj"
	City                = "[\"广西壮族自治区\",\"桂林市\",\"\"]"
	CityCode            = 450300
	CustomerName        = "桂林市疾病预防控制中心"
	CustomerProductName = "23价"
	Month               = 202103

	user = struct {
		birthday string
		tel      string
		sex      int //1男 2女
		cname    string
		Ftime    int //1针
		idcard   string
	}{
		birthday: "1993-12-11",
		tel:      "1885668989",
		sex:      2,
		cname:    "王二",
		Ftime:    1,
		idcard:   "610523198305134774",
	}
)

func main() {
	var (
		CustomerId        = 245
		CustomerProductId = 7
		Dates             = make([]string, 0)
		MxId              = "" //时间段id
		Guid              = "" //验证码id
	)

	//查询地点
	for CustomerId == 0 {
		url := fmt.Sprintf("https://cloud.cn2030.com/sc/wx/HandlerSubscribe.ashx?act=CustomerList&city=%s&id=0&cityCode=%d&product=2",
			url2.PathEscape(City), CityCode)
		json, err := http.Get(url).Headers(header()).Timeout(2000).Send().ReadToText()
		if err != nil {
			log.Println("CustomerList error,retry...")
			continue
		}
		jsonObj := util.NewJsonObject(json)
		jsonObj.GetArray("list").ForEach(func(i int, object *util.JsonObject) {
			if strings.Contains(object.Get("cname").(string), CustomerName) {
				CustomerId = int(object.Get("id").(float64))
				fmt.Println(jsonObj.GetArray("list").GetObject(i))
				return
			}
		})
	}
	fmt.Printf("CustomerId:%d\n", CustomerId)

	//查询疫苗以及预约时间
	for CustomerProductId == 0 {
		url := fmt.Sprintf("https://cloud.cn2030.com/sc/wx/HandlerSubscribe.ashx?act=CustomerProduct&id=%d", CustomerId)
		json, err := http.Get(url).Headers(header()).Send().ReadToText()
		if err != nil {
			log.Println("CustomerProduct error,retry...")
			continue
		}
		jsonObj := util.NewJsonObject(json)
		jsonObj.GetArray("list").ForEach(func(i int, object *util.JsonObject) {
			if strings.Contains(object.Get("text").(string), CustomerProductName) {
				CustomerProductId = int(object.Get("id").(float64))
				fmt.Println(jsonObj.GetArray("list").GetObject(i))
				return
			}
		})
	}
	fmt.Printf("CustomerProductId:%d\n", CustomerProductId)

	//查询可预约的日期
	for len(Dates) == 0 {
		url := fmt.Sprintf("https://cloud.cn2030.com/sc/wx/HandlerSubscribe.ashx?act=GetCustSubscribeDateAll&pid=%d&id=%d&month=%d",
			CustomerProductId, CustomerId, Month)
		json, err := http.Get(url).Headers(header()).Send().ReadToText()
		if err != nil {
			log.Println("GetCustSubscribeDateAll error,retry...")
			continue
		}
		jsonObj := util.NewJsonObject(json)
		jsonObj.GetArray("list").ForEach(func(i int, object *util.JsonObject) {
			if object.Get("enable").(bool) {
				Dates = append(Dates, object.Get("date").(string))
			}
		})
	}
	fmt.Printf("Dates:%v\n", Dates)

	//查询预约时间段
	ScDate := Dates[0]
	for MxId == "" {
		url := fmt.Sprintf("https://cloud.cn2030.com/sc/wx/HandlerSubscribe.ashx?act=GetCustSubscribeDateDetail&pid=%d&id=%d&scdate=%s",
			CustomerProductId, CustomerId, ScDate)
		json, err := http.Get(url).Headers(header()).Send().ReadToText()
		if err != nil {
			log.Println("GetCustSubscribeDateDetail error,retry...")
			continue
		}
		jsonObj := util.NewJsonObject(json)
		jsonObj.GetArray("list").ForEach(func(i int, object *util.JsonObject) {
			if object.Get("qty").(float64) > 0 { //库存
				MxId = object.Get("mxid").(string)
				return
			}
		})
	}
	fmt.Printf("MxId:%v\n", MxId)

	for Guid == "" {
		//获取验证吗
		url := "https://cloud.cn2030.com/sc/wx/HandlerSubscribe.ashx?act=GetCaptcha"
		json, err := http.Get(url).Headers(header()).Send().ReadToText()
		if err != nil {
			log.Println("GetCaptcha error,retry...")
			time.Sleep(2 * time.Second)
			continue
		}
		jsonObj := util.NewJsonObject(json)
		if fmt.Sprintf("%v", jsonObj.Get("status")) != "0" {
			log.Println(fmt.Sprintf("CaptchaVerify GetCaptcha:%s,retry...", jsonObj.Get("msg")))
			time.Sleep(2 * time.Second)
			continue
		}
		//fmt.Println(jsonObj)

		//识别验证码
		url = "http://127.0.0.1:8080/captcha"
		json, err = http.Post(url).Body(json).Send().ReadToText()
		if err != nil {
			log.Println("IdentifyVerify error,retry...")
			time.Sleep(2 * time.Second)
			continue
		}
		jsonObj = util.NewJsonObject(json)
		x := jsonObj.Get("x")

		//提交验证码
		url = fmt.Sprintf("https://cloud.cn2030.com/sc/wx/HandlerSubscribe.ashx?act=CaptchaVerify&token=&x=%v&y=%d", x, 5)
		json, err = http.Get(url).Headers(header()).Send().ReadToText()
		if err != nil {
			log.Println("CaptchaVerify error,retry...")
			time.Sleep(2 * time.Second)
			continue
		}
		jsonObj = util.NewJsonObject(json)
		if fmt.Sprintf("%v", jsonObj.Get("status")) == "408" {
			log.Println("Cookie 失效...")
			return
		} else if fmt.Sprintf("%v", jsonObj.Get("status")) != "200" {
			log.Println(fmt.Sprintf("CaptchaVerify error:(%v)%s,retry...", jsonObj.Get("status"), jsonObj.Get("msg")))
			time.Sleep(2 * time.Second)
			continue
		}
		Guid = jsonObj.Get("guid").(string)

		//提交预约
		url = fmt.Sprintf("https://cloud.cn2030.com/sc/wx/HandlerSubscribe.ashx?act=Save20"+
			"&birthday=%s&tel=%s&sex=%d&cname=%s&doctype=1&idcard=%s&mxid=%s&date=%s&pid=7&Ftime=%d&guid=%s",
			user.birthday, user.tel, user.sex, url2.QueryEscape(user.cname), user.idcard, MxId, ScDate, user.Ftime, Guid)
		json, err = http.Get(url).Headers(header()).Send().ReadToText()
		if err != nil {
			log.Println("Save20 error,retry...")
			time.Sleep(1 * time.Second)
			continue
		}
		jsonObj = util.NewJsonObject(json)
		if fmt.Sprintf("%v", jsonObj.Get("status")) != "200" {
			log.Println(fmt.Sprintf("Save20 error:%s,retry...", jsonObj.Get("msg")))
			time.Sleep(1 * time.Second)
			continue
		}
		fmt.Println(json)

		//预约状态
		url = "https://cloud.cn2030.com/sc/wx/HandlerSubscribe.ashx?act=GetOrderStatus"
		json, err = http.Get(url).Headers(header()).Send().ReadToText()
		if err != nil {
			log.Println("GetOrderStatus error,retry...")
			time.Sleep(1 * time.Second)
			continue
		}

		fmt.Println(json)

	}
	fmt.Println("Congratulations!!!")
}

func header() map[string]string {
	headers := make(map[string]string)
	headers["Host"] = "cloud.cn2030.com"
	headers["Content-Type"] = "application/json"
	headers["Accept"] = "*/*"
	headers["Connection"] = "keep-alive"
	headers["Cookie"] = Cookie
	headers["User-Agent"] = "Mozilla/5.0 (iPhone; CPU iPhone OS 11_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E217 MicroMessenger/6.8.0(0x16080000) NetType/WIFI Language/en Branch/Br_trunk MiniProgramEnv/Mac"
	headers["Referer"] = "https://servicewechat.com/wx2c7f0f3c30d99445/72/page-frame.html"
	headers["zftsl"], _ = GetZFTSL()
	headers["Accept-Language"] = "zh-cn"
	headers["Accept-Encoding"] = "gzip,deflate,br"
	return headers
}

func GetZFTSL() (string, error) {
	vm := otto.New()
	_, err := vm.Run(appJs)
	enc, err := vm.Call("zftsl", nil)
	if err != nil {
		fmt.Printf("GetZFTSL err : %v\n", err)
		return "", err
	}
	//fmt.Printf("zftsl : %s\n", enc.String())
	return enc.String(), nil
}

var appJs = `
function zftsl(){
    var a= 0
    var e = (Date.parse(new Date()) + a) / 1e3 + "";
    return s("zfsw_" + e.substring(0, e.length - 1));
}
function s(n){
    return l(v(n))
}
function r(n, e) {
    var r = (65535 & n) + (65535 & e);
    return (n >> 16) + (e >> 16) + (r >> 16) << 16 | 65535 & r;
}
function t(n, e, t, u, o, f) {
    return r((i = r(r(e, n), r(u, f))) << (c = o) | i >>> 32 - c, t);
    var i, c;
}
function u(n, e, r, u, o, f, i) {
    return t(e & r | ~e & u, n, e, o, f, i);
}
function o(n, e, r, u, o, f, i) {
    return t(e & u | r & ~u, n, e, o, f, i);
}
function f(n, e, r, u, o, f, i) {
    return t(e ^ r ^ u, n, e, o, f, i);
}
function i(n, e, r, u, o, f, i) {
    return t(r ^ (e | ~u), n, e, o, f, i);
}
function c(n, e) {
    var t, c, a, d;
    n[e >> 5] |= 128 << e % 32, n[14 + (e + 64 >>> 9 << 4)] = e;
    for (var l = 1732584193, h = -271733879, v = -1732584194, g = 271733878, m = 0; m < n.length; m += 16) {
        l = u(t = l, c = h, a = v, d = g, n[m], 7, -680876936), g = u(g, l, h, v, n[m + 1], 12, -389564586), v = u(v, g, l, h, n[m + 2], 17, 606105819), h = u(h, v, g, l, n[m + 3], 22, -1044525330), l = u(l, h, v, g, n[m + 4], 7, -176418897), g = u(g, l, h, v, n[m + 5], 12, 1200080426), v = u(v, g, l, h, n[m + 6], 17, -1473231341), h = u(h, v, g, l, n[m + 7], 22, -45705983), l = u(l, h, v, g, n[m + 8], 7, 1770035416), g = u(g, l, h, v, n[m + 9], 12, -1958414417), v = u(v, g, l, h, n[m + 10], 17, -42063), h = u(h, v, g, l, n[m + 11], 22, -1990404162), l = u(l, h, v, g, n[m + 12], 7, 1804603682), g = u(g, l, h, v, n[m + 13], 12, -40341101), v = u(v, g, l, h, n[m + 14], 17, -1502002290), l = o(l, h = u(h, v, g, l, n[m + 15], 22, 1236535329), v, g, n[m + 1], 5, -165796510), g = o(g, l, h, v, n[m + 6], 9, -1069501632), v = o(v, g, l, h, n[m + 11], 14, 643717713), h = o(h, v, g, l, n[m], 20, -373897302), l = o(l, h, v, g, n[m + 5], 5, -701558691), g = o(g, l, h, v, n[m + 10], 9, 38016083), v = o(v, g, l, h, n[m + 15], 14, -660478335), h = o(h, v, g, l, n[m + 4], 20, -405537848), l = o(l, h, v, g, n[m + 9], 5, 568446438), g = o(g, l, h, v, n[m + 14], 9, -1019803690), v = o(v, g, l, h, n[m + 3], 14, -187363961), h = o(h, v, g, l, n[m + 8], 20, 1163531501), l = o(l, h, v, g, n[m + 13], 5, -1444681467), g = o(g, l, h, v, n[m + 2], 9, -51403784), v = o(v, g, l, h, n[m + 7], 14, 1735328473), l = f(l, h = o(h, v, g, l, n[m + 12], 20, -1926607734), v, g, n[m + 5], 4, -378558), g = f(g, l, h, v, n[m + 8], 11, -2022574463), v = f(v, g, l, h, n[m + 11], 16, 1839030562), h = f(h, v, g, l, n[m + 14], 23, -35309556), l = f(l, h, v, g, n[m + 1], 4, -1530992060), g = f(g, l, h, v, n[m + 4], 11, 1272893353), v = f(v, g, l, h, n[m + 7], 16, -155497632), h = f(h, v, g, l, n[m + 10], 23, -1094730640), l = f(l, h, v, g, n[m + 13], 4, 681279174), g = f(g, l, h, v, n[m], 11, -358537222), v = f(v, g, l, h, n[m + 3], 16, -722521979), h = f(h, v, g, l, n[m + 6], 23, 76029189), l = f(l, h, v, g, n[m + 9], 4, -640364487), g = f(g, l, h, v, n[m + 12], 11, -421815835), v = f(v, g, l, h, n[m + 15], 16, 530742520), l = i(l, h = f(h, v, g, l, n[m + 2], 23, -995338651), v, g, n[m], 6, -198630844), g = i(g, l, h, v, n[m + 7], 10, 1126891415), v = i(v, g, l, h, n[m + 14], 15, -1416354905), h = i(h, v, g, l, n[m + 5], 21, -57434055), l = i(l, h, v, g, n[m + 12], 6, 1700485571), g = i(g, l, h, v, n[m + 3], 10, -1894986606), v = i(v, g, l, h, n[m + 10], 15, -1051523), h = i(h, v, g, l, n[m + 1], 21, -2054922799), l = i(l, h, v, g, n[m + 8], 6, 1873313359), g = i(g, l, h, v, n[m + 15], 10, -30611744), v = i(v, g, l, h, n[m + 6], 15, -1560198380), h = i(h, v, g, l, n[m + 13], 21, 1309151649), l = i(l, h, v, g, n[m + 4], 6, -145523070), g = i(g, l, h, v, n[m + 11], 10, -1120210379), v = i(v, g, l, h, n[m + 2], 15, 718787259), h = i(h, v, g, l, n[m + 9], 21, -343485551), l = r(l, t), h = r(h, c), v = r(v, a), g = r(g, d);
    }return [l, h, v, g];
}
function a(n) {
    for (var e = "", r = 32 * n.length, t = 0; t < r; t += 8) {
        e += String.fromCharCode(n[t >> 5] >>> t % 32 & 255);
    }return e;
}
function d(n) {
    var e = [];
    for (e[(n.length >> 2) - 1] = void 0, t = 0; t < e.length; t += 1) {
        e[t] = 0;
    }for (var r = 8 * n.length, t = 0; t < r; t += 8) {
        e[t >> 5] |= (255 & n.charCodeAt(t / 8)) << t % 32;
    }return e;
}
function l(n) {
    for (var e, r = "0123456789abcdef", t = "", u = 0; u < n.length; u += 1) {
        e = n.charCodeAt(u), t += r.charAt(e >>> 4 & 15) + r.charAt(15 & e);
    }return t;
}
function h(n) {
    return unescape(encodeURIComponent(n));
}
function v(n) {
    return a(c(d(e = h(n)), 8 * e.length));
    var e;
}
function g(n, e) {
    return function (n, e) {
        var r,
            t,
            u = d(n),
            o = [],
            f = [];
        for (o[15] = f[15] = void 0, 16 < u.length && (u = c(u, 8 * n.length)), r = 0; r < 16; r += 1) {
            o[r] = 909522486 ^ u[r], f[r] = 1549556828 ^ u[r];
        }return t = c(o.concat(d(e)), 512 + 8 * e.length), a(c(f.concat(t), 640));
    }(h(n), h(e));
}
function m(n, e, r) {
    return e ? r ? g(e, n) : l(g(e, n)) : r ? v(n) : l(v(n));
}

`