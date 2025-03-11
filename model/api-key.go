package model

import (
	"gorm.io/gorm"
	"sourcegraph2api/common"
	"time"
)

type ApiKey struct {
	Id         string    `json:"id" gorm:"type:varchar(64);not null;primaryKey"`
	ApiKey     string    `json:"apiKey" gorm:"type:varchar(255);not null;index"`
	Remark     string    `json:"remark" gorm:"type:varchar(900)"`
	UpdateTime time.Time `json:"update_time" gorm:"type:datetime;autoUpdateTime"`
	CreateTime time.Time `json:"create_time" gorm:"type:datetime;not null"`
}

func (c *ApiKey) Create(db *gorm.DB) error {
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

func (c *ApiKey) CountByKey(db *gorm.DB) (int64, error) {
	var count int64
	result := db.Model(&ApiKey{}).Where("api_key = ?", c.ApiKey).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (c *ApiKey) Exist(db *gorm.DB) (bool, error) {
	var count int64
	result := db.Model(&ApiKey{}).Where("`api_key` = ? ", c.ApiKey).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

func (c *ApiKey) ExistsNotMe(db *gorm.DB) (bool, error) {
	var count int64
	result := db.Model(&ApiKey{}).Where("api_key = ? and id != ?", c.ApiKey, c.Id).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

func (c *ApiKey) DeleteById(db *gorm.DB) error {
	result := db.Delete(&ApiKey{}, c.Id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (c *ApiKey) UpdateKeyById(db *gorm.DB) error {
	result := db.Model(&ApiKey{}).Where("id = ?", c.Id).
		Update("api_key", c.ApiKey).Update("remark", c.Remark)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (c *ApiKey) GetAll(db *gorm.DB) ([]ApiKey, error) {
	var apiKeys []ApiKey
	result := db.Find(&apiKeys)
	if result.Error != nil {
		return nil, result.Error
	}
	return apiKeys, nil
}
