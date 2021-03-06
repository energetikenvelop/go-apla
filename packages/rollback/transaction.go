// Copyright 2016 The go-daylight Authors
// This file is part of the go-daylight library.
//
// The go-daylight library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-daylight library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-daylight library. If not, see <http://www.gnu.org/licenses/>.

package rollback

import (
	"encoding/json"
	"strings"

	"github.com/AplaProject/go-apla/packages/consts"
	"github.com/AplaProject/go-apla/packages/converter"
	"github.com/AplaProject/go-apla/packages/model"

	log "github.com/sirupsen/logrus"
)

func rollbackUpdatedRow(tx map[string]string, where string, dbTransaction *model.DbTransaction, logger *log.Entry) error {
	var rollbackInfo map[string]string
	if err := json.Unmarshal([]byte(tx["data"]), &rollbackInfo); err != nil {
		logger.WithFields(log.Fields{"type": consts.JSONUnmarshallError, "error": err}).Error("unmarshalling rollback.Data from json")
		return err
	}
	addSQLUpdate := ""
	for k, v := range rollbackInfo {
		if v == "NULL" {
			addSQLUpdate += k + `=NULL,`
		} else if converter.IsByteColumn(tx["table_name"], k) && len(v) != 0 {
			addSQLUpdate += k + `=decode('` + string(converter.BinToHex([]byte(v))) + `','HEX'),`
		} else {
			addSQLUpdate += k + `='` + strings.Replace(v, `'`, `''`, -1) + `',`
		}
	}
	addSQLUpdate = addSQLUpdate[0 : len(addSQLUpdate)-1]
	if err := model.Update(dbTransaction, tx["table_name"], addSQLUpdate, where); err != nil {
		logger.WithFields(log.Fields{"type": consts.JSONUnmarshallError, "error": err, "query": addSQLUpdate}).Error("updating table")
		return err
	}
	return nil
}

func rollbackInsertedRow(tx map[string]string, where string, logger *log.Entry) error {
	if err := model.Delete(tx["table_name"], where); err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("deleting from table")
		return err
	}
	return nil
}

func rollbackTransaction(txHash []byte, dbTransaction *model.DbTransaction, logger *log.Entry) error {
	rollbackTx := &model.RollbackTx{}
	txs, err := rollbackTx.GetRollbackTransactions(dbTransaction, txHash)
	if err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("getting rollback transactions")
		return err
	}
	for _, tx := range txs {
		where := " WHERE id='" + tx["table_id"] + `'`
		if len(tx["data"]) > 0 {
			if err := rollbackUpdatedRow(tx, where, dbTransaction, logger); err != nil {
				return err
			}
		} else {
			if err := rollbackInsertedRow(tx, where, logger); err != nil {
				return err
			}
		}
	}
	txForDelete := &model.RollbackTx{TxHash: txHash}
	err = txForDelete.DeleteByHash(dbTransaction)
	if err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("deleting rollback transaction by hash")
		return err
	}
	return nil
}
