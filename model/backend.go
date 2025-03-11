package model

type AuthVerifyReq struct {
	AccessKey string `json:"accessKey"`
}

type ApiKeySaveReq struct {
	// ApiKey apiKey
	ApiKey string `json:"apiKey"`
	// Remark 备注
	Remark string `json:"remark"`
}

type ApiKeyUpdateReq struct {
	// Id id
	Id string `json:"id"`
	// ApiKey apiKey
	ApiKey string `json:"apiKey"`
	// Remark 备注
	Remark string `json:"remark"`
}

type ApiKeyResp struct {
	// Id Id
	Id string `json:"id"`
	// ApiKey apiKey
	ApiKey string `json:"apiKey"`
	// Remark 备注
	Remark string `json:"remark"`
	// CreateTime 创建时间
	CreateTime string `json:"createTime"`
}

type CookieSaveReq struct {
	// Cookie cookie
	Cookie string `json:"cookie"`
	// Remark 备注
	Remark string `json:"remark"`
}

type CookieUpdateReq struct {
	// Id Id
	Id string `json:"id"`
	// Cookie cookie
	Cookie string `json:"cookie"`
	// Remark 备注
	Remark string `json:"remark"`
}

type CookieResp struct {
	// Id Id
	Id string `json:"id"`
	// Cookie cookie
	Cookie string `json:"cookie"`
	// Credit 额度
	Credit int `json:"credit"`
	// Remark 备注
	Remark string `json:"remark"`
	// CreateTime 创建时间
	CreateTime string `json:"createTime"`
}
