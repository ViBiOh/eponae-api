package readings

import (
	"database/sql"
	"fmt"

	"github.com/ViBiOh/auth/auth"
	"github.com/ViBiOh/httputils/db"
)

const listTagsOfUserQuery = `
SELECT
  id,
  name
FROM
  tags
WHERE
  user_id = $1
`

const listTagsByidsQuery = `
SELECT
  id,
  name
FROM
  tags
WHERE
  id IN ($1)
`

const listReadingsTagsOfReadingsQuery = `
SELECT
  readings_id,
  tags_id
FROM
  readings_tags
WHERE
  readings_id IN ($1)
`

const insertTag = `
INSERT INTO
  tags
(
  name,
  user_id
) VALUES (
  $1,
  $2
)
RETURNING id
`
const updateTag = `
UPDATE
  tags
SET
  name = $2
WHERE
  id = $1
`

func scanTags(rows *sql.Rows) ([]*tag, error) {
	var (
		id   int64
		name string
	)

	list := make([]*tag, 0)

	for rows.Next() {
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf(`Error while scanning tag line: %v`, err)
		}

		list = append(list, &tag{ID: id, Name: name})
	}

	return list, nil
}

func scanReadingsTagsForTag(rows *sql.Rows) (map[int64][]int64, error) {
	var (
		readingID int64
		tagID     int64
	)

	list := make(map[int64][]int64, 0)

	for rows.Next() {
		if err := rows.Scan(&readingID, &tagID); err != nil {
			return nil, fmt.Errorf(`Error while scanning reading-tag line: %v`, err)
		}

		if _, ok := list[tagID]; ok {
			list[tagID] = append(list[tagID], readingID)
		} else {
			list[tagID] = []int64{readingID}
		}
	}

	return list, nil
}

func listTagsOfUser(user *auth.User) ([]*tag, error) {
	rows, err := readingsDB.Query(listTagsOfUserQuery, user.ID)
	if err != nil {
		return nil, fmt.Errorf(`Error while listing tags of user: %v`, err)
	}

	defer func() {
		err = db.RowsClose(`listing tags of user`, rows, err)
	}()

	return scanTags(rows)
}

func listTagsByIds(ids []int64) ([]*tag, error) {
	rows, err := readingsDB.Query(listTagsByidsQuery, db.WhereInInt64(ids))
	if err != nil {
		return nil, fmt.Errorf(`Error while listing tags by ids: %v`, err)
	}

	defer func() {
		err = db.RowsClose(`listing tags by ids`, rows, err)
	}()

	return scanTags(rows)
}

func addTagsForReadings(readings []*reading) error {
	if len(readings) == 0 {
		return nil
	}

	ids := make([]int64, 0)
	for _, reading := range readings {
		ids = append(ids, reading.ID)
	}

	rows, err := readingsDB.Query(listReadingsTagsOfReadingsQuery, db.WhereInInt64(ids))
	if err != nil {
		return fmt.Errorf(`Error while listing reading-tag of readings: %v`, err)
	}

	defer func() {
		err = db.RowsClose(`listing reading-tag of readings`, rows, err)
	}()

	tagLinks, err := scanReadingsTagsForTag(rows)
	if err != nil {
		return fmt.Errorf(`Error while scanning reading-tag of readings: %v`, err)
	} else if len(tagLinks) == 0 {
		return nil
	}

	tagsIds := make([]int64, 0)
	tagsByReading := make(map[int64][]int64, 0)
	for tagID, readingsIds := range tagLinks {
		tagsIds = append(tagsIds, tagID)

		for _, readingID := range readingsIds {
			if _, ok := tagsByReading[readingID]; ok {
				tagsByReading[readingID] = append(tagsByReading[readingID], tagID)
			} else {
				tagsByReading[readingID] = []int64{tagID}
			}
		}
	}

	tags, err := listTagsByIds(tagsIds)
	if err != nil {
		return fmt.Errorf(`Error while tags for readings: %v`, err)
	}

	tagsByID := make(map[int64]*tag, 0)
	for _, tag := range tags {
		tagsByID[tag.ID] = tag
	}

	for _, reading := range readings {
		for _, tagID := range tagsByReading[reading.ID] {
			if reading.Tags == nil {
				reading.Tags = make([]*tag, 0)
			}

			reading.Tags = append(reading.Tags, tagsByID[tagID])
		}
	}

	return nil
}

func saveTag(tag *tag, tx *sql.Tx) (err error) {
	if tag == nil {
		return fmt.Errorf(`Unable to save nil Tag`)
	}

	var usedTx *sql.Tx
	if usedTx, err = db.GetTx(readingsDB, `save tag`, tx); err != nil {
		return
	}

	if usedTx != tx {
		defer func() {
			err = db.EndTx(`save tag`, usedTx, err)
		}()
	}

	if tag.ID != 0 {
		if _, err = usedTx.Exec(updateTag, tag.ID, tag.Name); err != nil {
			err = fmt.Errorf(`Error while updating tag for user=%s: %v`, tag.user.Username, err)
		}
	} else {
		var newID int64

		if err = usedTx.QueryRow(insertTag, tag.Name, tag.user.ID).Scan(&newID); err != nil {
			err = fmt.Errorf(`Error while creating tag for user=%s: %v`, tag.user.Username, err)
		} else {
			tag.ID = newID
		}
	}

	return
}
