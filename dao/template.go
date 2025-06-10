package dao

import (
	"fmt"
	"time"
)

type TemplateRec struct {
	Name       string    `gorm:"column:name;type:varchar(255);unique;comment:Task template name" json:"name,omitempty"`
	Title      string    `gorm:"column:title;type:varchar(255);comment:Task template title" json:"title,omitempty"`
	Engine     string    `gorm:"column:engine;type:varchar(255);comment:Task engine" json:"engine,omitempty"`
	Schema     string    `gorm:"column:schema;type:text;comment:Task template metadata" json:"schema,omitempty"`
	Extra      string    `gorm:"column:extra;type:text;comment:Additional parameters for task template" json:"extra,omitempty"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime;comment:Create Time" json:"create_time,omitempty"`
}

/**
 * Get database table name
 * @return string Table name
 */
func (TemplateRec) TableName() string {
	return "template"
}

/**
 * Store template record
 * @return error Error object
 */
func (td *TemplateRec) Store() error {
	return DB.Create(td).Error
}

/**
 * Update template record
 * @return error Error object
 */
func (td *TemplateRec) Update() error {
	return DB.Model(td).Updates(td).Error
}

/**
 * Delete template record
 * @return error Error object
 */
func (td *TemplateRec) Delete() error {
	return DB.Where("name = ?", td.Name).Delete(td).Error
}

/**
 * Load task definition (Job) by task name
 */
func LoadTemplate(templateName string) (*TemplateRec, error) {
	if templateName == "" {
		return nil, fmt.Errorf("template is emptied")
	}
	var td TemplateRec
	err := DB.Model(&td).Where("name = ?", templateName).First(&td).Error
	return &td, err
}

/**
 * List all task classes
 */
func ListTemplates(verbose bool) ([]TemplateRec, error) {
	var tasks []TemplateRec
	if err := DB.Model(&TemplateRec{}).Find(&tasks).Error; err != nil {
		return []TemplateRec{}, err
	}
	if !verbose {
		for i, _ := range tasks {
			tasks[i].Schema = ""
		}
	}
	return tasks, nil
}
