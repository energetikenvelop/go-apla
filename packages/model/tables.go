package model

import (
	"strconv"

	"github.com/EGaaS/go-egaas-mvp/packages/converter"

	"github.com/jinzhu/gorm"
)

type Table struct {
	tableName             string
	Name                  string `gorm:"primary_key;not null;size:100"`
	ColumnsAndPermissions string `gorm:"not null"`
	Conditions            string `gorm:"not null"`
	RbID                  int64  `gorm:"not null"`
}

func (t *Table) SetTablePrefix(prefix string) {
	t.tableName = prefix + "_tables"
}

func (t *Table) TableName() string {
	return t.tableName
}

func (t *Table) Get(name string) (bool, error) {
	query := DBConn.Where("name = ?", name).First(t)
	if query.RecordNotFound() {
		return false, nil
	}
	return true, query.Error
}

func (t *Table) Create(transaction *DbTransaction) error {
	return getDB(transaction).Create(t).Error
}

func (t *Table) Delete() error {
	return DBConn.Delete(t).Error
}

func (t *Table) ToMap() map[string]string {
	result := make(map[string]string, 0)
	result["name"] = t.Name
	result["columns_and_permissions"] = t.ColumnsAndPermissions
	result["conditions"] = t.Conditions
	result["rb_id"] = strconv.FormatInt(t.RbID, 10)
	return result
}

func (t *Table) GetAll(prefix string) ([]Table, error) {
	result := make([]Table, 0)
	err := DBConn.Table(prefix + "_tables").Find(&result).Error
	return result, err
}

func (t *Table) GetTablePermissions(tablePrefix string, tableName string) (map[string]string, error) {
	var key, value string
	result := map[string]string{}
	row, err := DBConn.Table(tablePrefix+"_tables").
		Select("(jsonb_each_text(columns_and_permissions)).*").
		Where("name = ?", tableName).Rows()
	if err != nil {
		return nil, err
	}
	for row.Next() {
		row.Scan(&key, &value)
		result[key] = value
	}
	return result, err
}

func (t *Table) GetColumnsAndPermissions(tablePrefix string, tableName string) (map[string]string, error) {
	var key, value string
	result := map[string]string{}
	row, err := DBConn.Table(tablePrefix+"_tables").
		Select("(jsonb_each_text(columns_and_permissions->'update')).*").
		Where("name = ?", tableName).Rows()
	if err != nil {
		return nil, err
	}
	for row.Next() {
		row.Scan(&key, &value)
		result[key] = value
	}
	return result, err
}

func (t *Table) ExistsByName(name string) (bool, error) {
	query := DBConn.Where("name = ?", name).First(t)
	if query.Error == gorm.ErrRecordNotFound {
		return false, nil
	}
	return !query.RecordNotFound(), query.Error
}

func (t *Table) IsExistsByPermissionsAndTableName(columnName, tableName string) (bool, error) {
	query := DBConn.Where(`(columns_and_permissions->'update'-> ? ) is not null AND name = ?`, columnName, tableName).First(t)
	if query.Error == nil {
		return !query.RecordNotFound(), nil
	}
	if query.Error == gorm.ErrRecordNotFound {
		return false, nil
	}
	return false, query.Error
}

func (t *Table) GetPermissions(name, jsonKey string) (map[string]string, error) {
	keyStr := ""
	if jsonKey != "" {
		keyStr = `->'` + jsonKey + `'`
	}
	rows, err := DBConn.Raw(`SELECT data.* FROM "`+t.tableName+`", jsonb_each_text(columns_and_permissions`+keyStr+`) AS data WHERE name = ?`, name).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var key, value string
	result := map[string]string{}
	for rows.Next() {
		rows.Scan(&key, &value)
		result[key] = value
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *Table) SetActionByName(transaction *DbTransaction, table, name, action, actionValue string, rbID int64) (int64, error) {
	log.Debugf("set action by name: name = %s, actions = %s, actionsValue = %s", name, action, actionValue)
	query := getDB(transaction).Exec(`UPDATE "`+table+`" SET columns_and_permissions = jsonb_set(columns_and_permissions, '{`+action+`}', ?, true), rb_id = ? WHERE name = ?`, `"`+converter.EscapeForJSON(actionValue)+`"`, rbID, name)
	return query.RowsAffected, query.Error
}

func CreateStateTablesTable(transaction *DbTransaction, stateID string) error {
	return getDB(transaction).Exec(`CREATE TABLE "` + stateID + `_tables" (
				"name" varchar(100)  NOT NULL DEFAULT '',
				"columns_and_permissions" jsonb,
				"conditions" text  NOT NULL DEFAULT '',
				"rb_id" bigint NOT NULL DEFAULT '0'
				);
				ALTER TABLE ONLY "` + stateID + `_tables" ADD CONSTRAINT "` + stateID + `_tables_pkey" PRIMARY KEY (name);
	`).Error
}

func CreateTable(transaction *DbTransaction, tableName, colsSQL string) error {
	return getDB(transaction).Exec(`CREATE SEQUENCE "` + tableName + `_id_seq" START WITH 1;
				CREATE TABLE "` + tableName + `" (
				"id" bigint NOT NULL  default nextval('` + tableName + `_id_seq'),
				` + colsSQL + `
				"rb_id" bigint NOT NULL DEFAULT '0'
				);
				ALTER SEQUENCE "` + tableName + `_id_seq" owned by "` + tableName + `".id;
				ALTER TABLE ONLY "` + tableName + `" ADD CONSTRAINT "` + tableName + `_pkey" PRIMARY KEY (id);`).Error
}

func GetColumnsAndPermissionsAndRbIDWhereTable(table, tableName string) (map[string]string, error) {
	type proxy struct {
		ColumnsAndPermissions string
		RbID                  int64
	}
	temp := &proxy{}
	err := DBConn.Table(table).Where("name = ?", tableName).Select("columns_and_permissions, rb_id").Find(temp).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, 0)
	result["columns_and_permissions"] = temp.ColumnsAndPermissions
	result["rb_id"] = strconv.FormatInt(temp.RbID, 10)
	return result, nil
}

func GetTableWhereUpdatePermissionAndTableName(table, columnName, tableName string) (map[string]string, error) {
	type proxy struct {
		ColumnsAndPermissions string
		RbID                  int64
	}
	temp := &proxy{}
	err := DBConn.Table(table).Where("(columns_and_permissions->'update'-> ? ) is not null AND name = ?", columnName, tableName).Select("columns_and_permissions, rb_id").Find(temp).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, 0)
	result["columns_and_permissions"] = temp.ColumnsAndPermissions
	result["rb_id"] = strconv.FormatInt(temp.RbID, 10)
	return result, nil
}