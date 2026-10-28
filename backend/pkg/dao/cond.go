package dao

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 本文件集中 BuildCondition 的结构化条件拼装助手。
//
// 约定（与 golib gormdao/cond.go、ark-go-starter 风格一致）：列名与表名一律交给
// GORM 的 clause 按方言引用（PostgreSQL `"user"."status"`、MySQL "`user`.`status`"），
// 调用方**不再**手写 `tableName+".col = ?"` 之类的字符串拼接。
//
// 这样做的收益：
//   - 保留字表名（user / order …）与保留字列名都由 Dialector.QuoteTo 正确处理，
//     不会出现未引用的 `user.tenant_id` 或反引号在 PG 上语法错误；
//   - 条件语义（Eq / IN / Like / IS NULL）由 clause 表达，避免手拼 SQL 片段时漏转义或拼错。
//
// 用法：在各 Cond.BuildCondition 顶部用 condCols(tableName) 得到绑定好表名的列构造函数。
//
//	col := condCols(tableName)
//	if c.Status != "" {
//	    db.Where(clause.Eq{Column: col("status"), Value: c.Status})
//	}

// condCols 返回绑定到给定表名的列构造器：col("status") => clause.Column{Table: tableName, Name: "status"}。
//
// tableName 已是当前方言下的引用形式（见 gormdao.Cond.BuildCondition 契约），
// clause.Column 在渲染时会再次交给 Dialector 按方言引用，两种形态都幂等、安全。
func condCols(tableName string) func(name string) clause.Column {
	return func(name string) clause.Column {
		return clause.Column{Table: tableName, Name: name}
	}
}

// buildOrLike 构造「多列 LIKE 任一命中」的 OR 组：colA LIKE %kw% OR colB LIKE %kw% ...
// 无条件时返回 nil，调用方按需跳过 db.Where，避免生成非法空条件。
func buildOrLike(col func(string) clause.Column, keyword string, columns ...string) clause.Expression {
	if len(columns) == 0 {
		return nil
	}
	like := "%" + keyword + "%"
	exprs := make([]clause.Expression, 0, len(columns))
	for _, name := range columns {
		exprs = append(exprs, clause.Like{Column: col(name), Value: like})
	}
	return clause.Or(exprs...)
}

// buildOrEqPrefix 构造「某列等于 v 或按前缀匹配」的 OR 组：col = v OR col LIKE v/%
// 用于部门子树（dept_path = X OR dept_path LIKE X/%）这类同列两态判定。
func buildOrEqPrefix(col func(string) clause.Column, name, value, prefix string) clause.Expression {
	return clause.Or(
		clause.Eq{Column: col(name), Value: value},
		clause.Like{Column: col(name), Value: prefix},
	)
}

// quoteIdent 按当前方言引用单个标识符（表名或列名），用于必须手拼 SQL 片段的场景
// （如 clause.Expr 里的子查询）。普通 Eq/IN/Like 走 clause.Column 即可，无需它。
func quoteIdent(db *gorm.DB, name string) string {
	if db == nil || db.Statement == nil {
		return name
	}
	return db.Statement.Quote(name)
}

// quoteCol 按当前方言引用「表名.列名」，与 clause.Column 渲染结果一致
// （PostgreSQL `"user"."person_id"`）。用于 clause.Expr 的左侧标识符。
func quoteCol(db *gorm.DB, table, column string) string {
	return quoteIdent(db, table) + "." + quoteIdent(db, column)
}

// toAnySlice 把字符串切片转为 clause.IN 需要的 []any。
func toAnySlice(ss []string) []any {
	out := make([]any, 0, len(ss))
	for _, s := range ss {
		out = append(out, s)
	}
	return out
}
