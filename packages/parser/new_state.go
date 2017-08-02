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

package parser

import (
	"fmt"

	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/model"
	"github.com/EGaaS/go-egaas-mvp/packages/template"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
	"github.com/EGaaS/go-egaas-mvp/packages/utils/tx"

	"gopkg.in/vmihailenco/msgpack.v2"
)

type NewStateParser struct {
	*Parser
	NewState *tx.NewState
}

func (p *NewStateParser) Init() error {
	newState := &tx.NewState{}
	if err := msgpack.Unmarshal(p.TxBinaryData, newState); err != nil {
		return p.ErrInfo(err)
	}
	p.NewState = newState
	return nil
}

func (p *NewStateParser) Validate() error {
	err := p.generalCheck(`new_state`, &p.NewState.Header, map[string]string{})
	if err != nil {
		return p.ErrInfo(err)
	}

	// Check InputData
	verifyData := map[string][]interface{}{"state_name": []interface{}{p.NewState.StateName}, "currency_name": []interface{}{p.NewState.CurrencyName}}
	err = p.CheckInputData(verifyData)
	if err != nil {
		return p.ErrInfo(err)
	}

	CheckSignResult, err := utils.CheckSign(p.PublicKeys, p.NewState.ForSign(), p.NewState.Header.BinSignatures, false)
	if err != nil {
		return p.ErrInfo(err)
	}
	if !CheckSignResult {
		return p.ErrInfo("incorrect sign")
	}
	country := string(p.NewState.StateName)
	if exist, err := IsState(country); err != nil {
		return p.ErrInfo(err)
	} else if exist > 0 {
		return fmt.Errorf(`State %s already exists`, country)
	}

	return nil
}

func (p *NewStateParser) Main(country, currency string) (id string, err error) {
	systemState := &model.SystemState{RbID: 0}
	err = systemState.Create()
	if err != nil {
		return
	}
	id = converter.Int64ToStr(systemState.ID)
	rollbackTx := model.RollbackTx{BlockID: p.BlockData.BlockID, TxHash: []byte(p.TxHash), TableName: "system_states", TableID: id}
	err = rollbackTx.Create()
	if err != nil {
		return
	}
	err = model.CreateStateTable(id)
	if err != nil {
		return
	}
	sid := "ContractConditions(`MainCondition`)" //`$citizen == ` + utils.Int64ToStr(p.TxWalletID) // id + `_citizens.id=` + utils.Int64ToStr(p.TxWalletID)
	psid := sid                                  //fmt.Sprintf(`Eval(StateParam(%s, "main_conditions"))`, id) //id+`_state_parameters.main_conditions`
	err = model.CreateStateConditions(id, sid, psid, currency, country, p.TxWalletID)
	if err != nil {
		return
	}
	err = model.CreateSmartContractTable(id)
	if err != nil {
		return
	}
	sc := &model.SmartContracts{
		Name: "Main condition",
		Value: []byte(`contract MainCondition {
			data {}
			conditions {
			    if(StateVal("gov_account")!=$citizen)
			    {
				warning "Sorry, you don't have access to this action."
			    }
		        }
			action {}
		}`),
		WalletID: p.TxWalletID,
		Active:   "1"}
	sc.SetTableName(id + "_smart_contracts")
	err = sc.Create()
	if err != nil {
		return
	}
	scu := &model.SmartContracts{}
	scu.SetTableName(id + "_smart_contracts")
	err = scu.UpdateConditions(sid)
	if err != nil {
		return
	}

	err = model.CreateStateTablesTable(id)
	if err != nil {
		return
	}
	t := &model.Tables{
		Name: id + "citizens",
		ColumnsAndPermissions: `{"general_update":"` + sid + `", "update": {"public_key_0": "` + sid + `"}, "insert": "` + sid + `", "new_column":"` + sid + `"}`,
		Conditions:            psid,
	}
	t.SetTableName(id + "_tables")
	err = t.Create()
	if err != nil {
		return
	}

	err = model.CreateStatePagesTable(id)
	if err != nil {
		return
	}
	dashboardValue := `FullScreen(1)
	If(StateVal(type_office))
	Else:
	Title : Basic Apps
	Divs: col-md-4
			Divs: panel panel-default elastic
				Divs: panel-body text-center fill-area flexbox-item-grow
					Divs: flexbox-item-grow flex-center
						Divs: pv-lg
						Image("/static/img/apps/money.png", Basic, center-block img-responsive img-circle img-thumbnail thumb96 )
						DivsEnd:
						P(h4,Basic Apps)
						P(text-left,"Election and Assign, Polling, Messenger, Simple Money System")
					DivsEnd:
				DivsEnd:
				Divs: panel-footer
					Divs: clearfix
						Divs: pull-right
							BtnPage(app-basic, Install,'',btn btn-primary lang)
						DivsEnd:
					DivsEnd:
				DivsEnd:
			DivsEnd:
		DivsEnd:
	IfEnd:
	PageEnd:
`
	governmentValue := `FullScreen(1)
If(StateVal(type_office))
Else:
Title : Basic Apps
Divs: col-md-4
		Divs: panel panel-default elastic
			Divs: panel-body text-center fill-area flexbox-item-grow
				Divs: flexbox-item-grow flex-center
					Divs: pv-lg
					Image("/static/img/apps/money.png", Basic, center-block img-responsive img-circle img-thumbnail thumb96 )
					DivsEnd:
					P(h4,Basic Apps)
					P(text-left,"Election and Assign, Polling, Messenger, Simple Money System")
				DivsEnd:
			DivsEnd:
			Divs: panel-footer
				Divs: clearfix
					Divs: pull-right
						BtnPage(app-basic, Install,'',btn btn-primary lang)
					DivsEnd:
				DivsEnd:
			DivsEnd:
		DivsEnd:
	DivsEnd:
IfEnd:
PageEnd:
`
	firstPage := &model.Page{
		Name:       "dashboard_default",
		Value:      dashboardValue,
		Menu:       "menu_default",
		Conditions: sid,
	}
	firstPage.SetTableName(id + "_page")
	err = firstPage.Create()
	if err != nil {
		return
	}
	secondPage := &model.Page{
		Name:       "government",
		Value:      governmentValue,
		Menu:       "government",
		Conditions: sid,
	}
	secondPage.SetTableName(id + "_page")
	err = secondPage.Create()
	if err != nil {
		return
	}

	err = model.CreateStateMenuTable(id)
	if err != nil {
		return
	}
	firstMenu := &model.Menu{
		Name: "menu_default",
		Value: `MenuItem(Dashboard, dashboard_default)
 MenuItem(Government dashboard, government)`,
		Conditions: sid,
	}
	firstMenu.SetTableName(id + "_menu")
	err = firstMenu.Create()
	if err != nil {
		return
	}
	secondMenu := &model.Menu{
		Name: `government`,
		Value: `MenuItem(Citizen dashboard, dashboard_default)
MenuItem(Government dashboard, government)
MenuGroup(Admin tools,admin)
MenuItem(Tables,sys-listOfTables)
MenuItem(Smart contracts, sys-contracts)
MenuItem(Interface, sys-interface)
MenuItem(App List, sys-app_catalog)
MenuItem(Export, sys-export_tpl)
MenuItem(Wallet,  sys-edit_wallet)
MenuItem(Languages, sys-languages)
MenuItem(Signatures, sys-signatures)
MenuItem(Gen Keys, sys-gen_keys)
MenuEnd:
MenuBack(Welcome)`,
		Conditions: sid,
	}
	secondMenu.SetTableName(id + "_menu")
	err = secondMenu.Create()
	if err != nil {
		return
	}

	err = model.CreateCitizensStateTable(id)
	if err != nil {
		return
	}

	dltWallet := &model.DltWallet{}
	err = dltWallet.GetWallet(p.TxWalletID)
	if err != nil {
		return
	}

	citizen := &model.Citizens{ID: p.TxWalletID, PublicKey: converter.BinToHex(dltWallet.PublicKey)}
	citizen.SetTableName(converter.StrToInt64(id))
	err = citizen.Create()
	if err != nil {
		return
	}
	err = model.CreateLanguagesStateTable(id)
	if err != nil {
		return
	}
	err = model.CreateStateDefaultLanguages(id, sid)
	if err != nil {
		return
	}

	err = model.CreateSignaturesStateTable(id)
	if err != nil {
		return
	}

	err = model.CreateStateAppsTable(id)
	if err != nil {
		return
	}

	err = model.CreateStateAnonymsTable(id)
	if err != nil {
		return
	}

	err = template.LoadContract(id)
	return
}

func (p *NewStateParser) Action() error {
	country := string(p.NewState.StateName)
	currency := string(p.NewState.CurrencyName)
	_, err := p.Main(country, currency)
	if err != nil {
		return p.ErrInfo(err)
	}
	dltWallet := &model.DltWallet{}
	err = dltWallet.GetWallet(p.TxWalletID)
	if err != nil {
		return p.ErrInfo(err)
	} else if len(p.NewState.Header.PublicKey) > 30 && len(dltWallet.PublicKey) == 0 {
		_, _, err = p.selectiveLoggingAndUpd([]string{"public_key_0"}, []interface{}{converter.HexToBin(p.NewState.Header.PublicKey)}, "dlt_wallets",
			[]string{"wallet_id"}, []string{converter.Int64ToStr(p.TxWalletID)}, true)
	}
	return err
}

func (p *NewStateParser) Rollback() error {
	rollbackTx := &model.RollbackTx{}
	err := rollbackTx.Get([]byte(p.TxHash), "system_states")
	if err != nil {
		return p.ErrInfo(err)
	}
	err = p.autoRollback()
	if err != nil {
		return p.ErrInfo(err)
	}

	for _, name := range []string{`menu`, `pages`, `citizens`, `languages`, `signatures`, `tables`,
		`smart_contracts`, `state_parameters`, `apps`, `anonyms`} {
		err = model.DBConn.DropTable(fmt.Sprintf("%d_%s", rollbackTx.TableID, name)).Error
		if err != nil {
			return p.ErrInfo(err)
		}
	}

	rollbackTxToDel := &model.RollbackTx{TxHash: []byte(p.TxHash), TableName: "system_states"}
	err = rollbackTxToDel.DeleteByHashAndTableName()
	if err != nil {
		return p.ErrInfo(err)
	}

	ss := &model.SystemState{}
	err = ss.GetLast()
	if err != nil {
		return p.ErrInfo(err)
	}
	// обновляем AI
	// update  the AI
	err = model.SetAI("system_states", ss.ID+1)
	if err != nil {
		return p.ErrInfo(err)
	}
	ssToDel := &model.SystemState{ID: ss.ID}
	err = ssToDel.Delete()
	if err != nil {
		return p.ErrInfo(err)
	}

	return nil
}

func (p NewStateParser) Header() *tx.Header {
	return &p.NewState.Header
}
