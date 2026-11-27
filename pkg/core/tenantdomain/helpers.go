package tenantdomain

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// lockClause 返回 SELECT ... FOR UPDATE 的加锁子句（PG 生效）。
func lockClause() clause.Locking {
	return clause.Locking{Strength: "UPDATE"}
}

// isNotFound 判定 gorm 记录不存在。
func isNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
