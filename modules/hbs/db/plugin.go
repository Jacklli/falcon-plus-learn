package db

import (
	"log"
)
/*
查询hostgroup id对应的plugin dir

SQL: select grp_id, dir from plugin_dir

返回:
{
  grp_id1: ["plugin dir1", "plugin dir2"],
  grp_id2: ["plugin dir1", "plugin dir3"],
  grp_id3: ["plugin dir4"],
}
 */
func QueryPlugins() (map[int][]string, error) {
	m := make(map[int][]string)

	sql := "select grp_id, dir from plugin_dir"
	rows, err := DB.Query(sql)
	if err != nil {
		log.Println("ERROR:", err)
		return m, err
	}

	defer rows.Close()
	for rows.Next() {
		var (
			id  int
			dir string
		)

		err = rows.Scan(&id, &dir)
		if err != nil {
			log.Println("ERROR:", err)
			continue
		}

		if _, exists := m[id]; exists {
			m[id] = append(m[id], dir)
		} else {
			m[id] = []string{dir}
		}
	}

	return m, nil
}
