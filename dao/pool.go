package dao

import (
	"fmt"

	"gorm.io/gorm"
)

//------------------------------------------------------------------------------
//	Pool
//------------------------------------------------------------------------------

/**
 *	Task pool
 */
type Pool struct {
	PoolId      string `gorm:"primaryKey;column:pool_id;type:varchar(30)" json:"pool_id"`         //pool ID
	Engine      string `gorm:"column:engine;type:varchar(30)" json:"engine"`                      //task execution engine
	Description string `gorm:"column:description;type:varchar(255)" json:"description,omitempty"` //pool description showing key information for user understanding
	Config      string `gorm:"column:config;type:text" json:"config,omitempty"`                   //pool configuration for various task engines
	Running     int    `gorm:"column:running;type:int" json:"running"`                            //maximum concurrent tasks
	Waiting     int    `gorm:"column:waiting;type:int" json:"waiting"`                            //maximum queued tasks
}

/**
 * Maps Pool struct to database pool table
 */
func (Pool) TableName() string {
	return "pool"
}

/**
 * Stores Pool record to database
 */
func (p *Pool) Store(tx *gorm.DB) error {
	return tx.Create(p).Error
}

/**
 * Updates Pool record in database
 */
func (p *Pool) Update(tx *gorm.DB) error {
	return tx.Updates(p).Error
}

/**
 * Checks if Pool exists in database
 */
func (p *Pool) Exists(tx *gorm.DB) (bool, error) {
	var pools []Pool
	result := tx.Model(p).Find(&pools, "pool_id = ?", p.PoolId)
	if result.Error != nil {
		return false, result.Error
	}
	if len(pools) == 0 {
		return false, nil
	}
	return true, nil
}

/**
 *	Retrieves all task pools
 */
func ListPools() ([]Pool, error) {
	if DB.Error != nil {
		return []Pool{}, DB.Error
	}
	var pools []Pool
	err := DB.Model(&Pool{}).Find(&pools).Error
	if err != nil {
		return []Pool{}, err
	}
	return pools, nil
}

/**
 *	Loads task pool
 */
func LoadPool(poolId string) (*Pool, error) {
	if poolId == "" {
		return nil, fmt.Errorf("poolId is not exist")
	}
	var td Pool
	err := DB.Model(&td).Where("pool_id = ?", poolId).First(&td).Error
	return &td, err
}

//------------------------------------------------------------------------------
//	PoolResource
//------------------------------------------------------------------------------

type PoolResource struct {
	Id      int    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`              //record ID
	PoolId  string `gorm:"column:pool_id;type:varchar(30);not null" json:"pool_id"`   //pool ID
	ResName string `gorm:"column:res_name;type:varchar(30);not null" json:"res_name"` //resource name
	ResNum  string `gorm:"column:res_num;type:varchar(50);not null" json:"res_num"`   //resource quantity
}

/**
 * Maps PoolResource struct to database pool_resource table
 */
func (PoolResource) TableName() string {
	return "pool_resource"
}

/**
 * Stores PoolResource record to database
 */
func (r *PoolResource) Store(tx *gorm.DB) error {
	return tx.Create(r).Error
}

/**
 * Updates PoolResource record in database
 */
func (r *PoolResource) Update(tx *gorm.DB) error {
	return tx.Updates(r).Error
}

/**
 *	Lists all resource records for poolId
 */
func ListPoolResources(poolId string) ([]PoolResource, error) {
	if DB.Error != nil {
		return []PoolResource{}, DB.Error
	}

	var resources []PoolResource
	result := DB.Where("pool_id = ?", poolId).Find(&resources)
	if result.Error != nil {
		return []PoolResource{}, result.Error
	}

	return resources, nil
}

/**
 * Deletes Pool and related PoolResources
 */
func DeletePool(poolId string) error {
	if poolId == "" {
		return fmt.Errorf("poolId is empty")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		// Delete associated PoolResources
		if err := tx.Where("pool_id = ?", poolId).Delete(&PoolResource{}).Error; err != nil {
			return err
		}
		// Delete Pool
		if err := tx.Where("pool_id = ?", poolId).Delete(&Pool{}).Error; err != nil {
			return err
		}
		return nil
	})
}
