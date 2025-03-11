package model

import (
	"gorm.io/gorm"
	"sourcegraph2api/common"
	"time"
)

type Chat struct {
	Id                         string    `json:"id" gorm:"type:varchar(64);not null;primaryKey"`
	Cookie                     string    `json:"cookie" gorm:"type:text"`
	Model                      string    `json:"model" gorm:"type:varchar(255);not null;index:idx_cookie_hash_last_messages,priority:1"`
	CookieHash                 string    `json:"cookie_hash" gorm:"type:varchar(255);not null"`
	HixChatId                  string    `json:"hix_chat_id" gorm:"type:varchar(255);not null"`
	LastMessagesPair           string    `json:"last_messages_pair" gorm:"type:text"`
	LastMessagesPairSha256Hash string    `json:"last_messages_pair_sha256_hash" gorm:"type:varchar(255);not null;indexidx_cookie_hash_last_messages,priority:2"`
	UpdateTime                 time.Time `json:"update_time" gorm:"type:datetime;autoUpdateTime"`
	CreateTime                 time.Time `json:"create_time" gorm:"type:datetime;not null"`
}

func (c *Chat) Create(db *gorm.DB) error {
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

// 根据 HixChatId 更新对话记录的方法
func (c *Chat) UpdateLastMessages(db *gorm.DB) error {
	result := db.Model(&Chat{}).
		Where("hix_chat_id = ?", c.HixChatId).
		Updates(map[string]interface{}{
			"last_messages_pair":             c.LastMessagesPair,
			"last_messages_pair_sha256_hash": c.LastMessagesPairSha256Hash,
		})

	if result.Error != nil {
		return result.Error
	}

	// 可选：检查是否实际更新了记录
	//if result.RowsAffected == 0 {
	//	return gorm.ErrRecordNotFound
	//}

	return nil
}
