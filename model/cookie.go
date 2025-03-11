package model

import (
	"gorm.io/gorm"
	"sourcegraph2api/common"
	"time"
)

type Cookie struct {
	Id             string    `json:"id" gorm:"type:varchar(64);not null;primaryKey"`
	Cookie         string    `json:"cookie" gorm:"type:text"`
	CookieHash     string    `json:"cookie_hash" gorm:"type:varchar(255);not null;index"`
	Credit         int       `json:"credit" gorm:"type:bigint(20);not null"`
	AdvancedCredit int       `json:"advanced_credit" gorm:"type:bigint(20);not null"`
	IsActiveSub    bool      `json:"is_active_sub" gorm:"type:tinyint(1);not null;default:0"`
	Remark         string    `json:"remark" gorm:"type:varchar(900)"`
	UpdateTime     time.Time `json:"update_time" gorm:"type:datetime;autoUpdateTime"`
	CreateTime     time.Time `json:"create_time" gorm:"type:datetime;not null"`
}

func (c *Cookie) Create(db *gorm.DB) error {
	if c.Id == "" {
		id, err := common.NextID()
		if err != nil {
			return err
		}
		c.Id = id
		c.CreateTime = time.Now()
	}
	result := db.Create(c)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (c *Cookie) FindAllCookies(db *gorm.DB) ([]Cookie, error) {
	var cookies []Cookie
	result := db.Find(&cookies)
	if result.Error != nil {
		return nil, result.Error
	}
	return cookies, nil
}

func (c *Cookie) FindByMinimumCredit(db *gorm.DB) ([]Cookie, error) {
	var cookies []Cookie
	result := db.Where("credit >= ?", c.Credit).Find(&cookies)
	if result.Error != nil {
		return nil, result.Error
	}
	return cookies, nil
}

// 定义用于接收查询结果的结构体
type ChatResult struct {
	Cookie    string
	HixChatId string
}

func QueryCookiesByChatHashAndModelAndCredit(db *gorm.DB, lastMessagesPairSha256Hash, modelName string, creditLimit int) (string, string, error) {
	var result ChatResult

	// 从Chat表出发进行查询，关联Cookie表
	err := db.Model(&Chat{}).
		Select("chats.cookie, chats.hix_chat_id").
		Joins("JOIN cookies ON chats.cookie_hash = cookies.cookie_hash").
		Where("chats.last_messages_pair_sha256_hash = ?  AND chats.model = ? AND cookies.credit >= ?",
					lastMessagesPairSha256Hash, modelName, creditLimit).
		First(&result).Error // 使用First获取第一条记录

	if err != nil {
		return "", "", err
	}

	return result.Cookie, result.HixChatId, nil
}

func (c *Cookie) UpdateCreditByCookieHash(db *gorm.DB) error {
	// 使用 GORM 的 Model 方法指定模型，并使用 Where 方法指定条件
	result := db.Model(&Cookie{}).Where("cookie_hash = ?", c.CookieHash).Update("credit", c.Credit)
	if result.Error != nil {
		return result.Error
	}

	// 检查是否有记录被更新
	//if result.RowsAffected == 0 {
	//	return fmt.Errorf("no record found with cookie_hash: %s", cookieHash)
	//}

	return nil
}

func (c *Cookie) Exist(db *gorm.DB) (bool, error) {
	var count int64
	result := db.Model(&Cookie{}).Where("`cookie` = ? ", c.Cookie).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

func (c *Cookie) ExistsNotMe(db *gorm.DB) (bool, error) {
	var count int64
	result := db.Model(&Cookie{}).Where("cookie = ? and id != ?", c.Cookie, c.Id).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

func (c *Cookie) DeleteById(db *gorm.DB) error {
	result := db.Delete(&Cookie{}, c.Id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (c *Cookie) UpdateKeyById(db *gorm.DB) error {
	result := db.Model(&Cookie{}).Where("id = ?", c.Id).
		Update("cookie", c.Cookie).
		Update("cookie_hash", c.CookieHash).
		Update("remark", c.Remark)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (c *Cookie) GetAll(db *gorm.DB) ([]Cookie, error) {
	var cookies []Cookie
	result := db.Find(&cookies)
	if result.Error != nil {
		return nil, result.Error
	}
	return cookies, nil
}

func (c *Cookie) FindByMinimumCreditAdvanced(db *gorm.DB) ([]Cookie, error) {
	var cookies []Cookie
	result := db.Where("advanced_credit >= ? and is_active_sub = 1", c.Credit).Find(&cookies)
	if result.Error != nil {
		return nil, result.Error
	}
	return cookies, nil
}
