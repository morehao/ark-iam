package dao

import (
	"context"
	"errors"

	"github.com/morehao/golib/dbaccess/gormdao"
)

// ErrUpdateFieldsEmpty 表示调用方没有列出任何要更新的字段。
//
// 必须拒绝，而不是退回「更新实体上全部非零字段」：GORM 的结构化 Updates 在未 Select 时会把
// 实体所有非零字段写回，与本函数「只更新显式列出的字段」的契约相反。调用方一旦漏传列名
// （例如只想改 status，结果变成一次整行覆盖），错误会静默扩散进数据，故直接报错暴露。
var ErrUpdateFieldsEmpty = errors.New("dao: UpdateFields 需要至少一个待更新字段")

// UpdateFields 按主键只更新显式列出的字段（含零值），走 GORM 结构化 Updates 路径，
// 使 gorm:"serializer:json" 生效。UpdateMap 不经过 serializer，写 JSON 列会静默落脏值。
//
// fields 必须非空（否则返回 ErrUpdateFieldsEmpty）：Select 是本函数语义的一部分，不是可选优化。
func UpdateFields[T gormdao.Entity, L ~[]T, ID gormdao.IDType](ctx context.Context, d *gormdao.Dao[T, L, ID], id ID, entity *T, fields ...string) error {
	if len(fields) == 0 {
		return ErrUpdateFieldsEmpty
	}
	return d.DB(ctx).
		Model(new(T)).
		Table(d.TableName).
		Select(fields).
		Where("id = ?", id).
		Updates(entity).Error
}
