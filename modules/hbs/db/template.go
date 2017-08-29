package db

import (
	"github.com/open-falcon/falcon-plus/common/model"
	"log"
)
/*
查询hostgroup id对应的template id

SQL: select grp_id, tpl_id from grp_tpl

返回：
{
  grp_id1: [tpl_id1, tpl_id2, tpl_id3],
  grp_id2: [tpl_id1, tpl_id2, tpl_id5],
  grp_id3: [tpl_id3, tpl_id4],
}
 */
func QueryGroupTemplates() (map[int][]int, error) {
	m := make(map[int][]int)

	sql := "select grp_id, tpl_id from grp_tpl"
	rows, err := DB.Query(sql)
	if err != nil {
		log.Println("ERROR:", err)
		return m, err
	}

	defer rows.Close()
	for rows.Next() {
		var gid, tid int
		err = rows.Scan(&gid, &tid)
		if err != nil {
			log.Println("ERROR:", err)
			continue
		}

		if _, exists := m[gid]; exists {
			m[gid] = append(m[gid], tid)
		} else {
			m[gid] = []int{tid}
		}
	}

	return m, nil
}

// 获取所有的策略模板列表
/*
查询所有template信息

SQL: select id, tpl_name, parent_id, action_id, create_user from tpl

返回：
{
  Templateid1: &model.Template{},
  Templateid2: &model.Template{},
  Templateid3: &model.Template{},
}
 */
func QueryTemplates() (map[int]*model.Template, error) {

	templates := make(map[int]*model.Template)

	sql := "select id, tpl_name, parent_id, action_id, create_user from tpl"
	rows, err := DB.Query(sql)
	if err != nil {
		log.Println("ERROR:", err)
		return templates, err
	}

	defer rows.Close()
	for rows.Next() {
		t := model.Template{}
		err = rows.Scan(&t.Id, &t.Name, &t.ParentId, &t.ActionId, &t.Creator)
		if err != nil {
			log.Println("ERROR:", err)
			continue
		}
		templates[t.Id] = &t
	}

	return templates, nil
}

// 一个机器ID对应了多个模板ID
/*
查询host id对应的template id

SQL: select a.tpl_id, b.host_id from grp_tpl as a inner join grp_host as b on a.grp_id=b.grp_id

返回：
{
  hostid1: [templateid1, templateid2, templateid3],
  hostid2: [templateid1, templateid2, templateid4],
  hostid3: [templateid4, templateid5],
}
 */
func QueryHostTemplateIds() (map[int][]int, error) {
	ret := make(map[int][]int)
	rows, err := DB.Query("select a.tpl_id, b.host_id from grp_tpl as a inner join grp_host as b on a.grp_id=b.grp_id")
	if err != nil {
		log.Println("ERROR:", err)
		return ret, err
	}

	defer rows.Close()
	for rows.Next() {
		var tid, hid int

		err = rows.Scan(&tid, &hid)
		if err != nil {
			log.Println("ERROR:", err)
			continue
		}

		if _, ok := ret[hid]; ok {
			ret[hid] = append(ret[hid], tid)
		} else {
			ret[hid] = []int{tid}
		}
	}

	return ret, nil
}
