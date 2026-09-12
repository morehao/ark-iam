// Package person 承载自然人（person）聚合根的跨表领域能力，
// 供 auth / tenantadmin / platformadmin 三服务复用。
package person

import (
	"context"
	"encoding/json"

	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"gorm.io/gorm"
)

// FindOrCreateReq 构造 FindOrCreate 入参。
type FindOrCreateReq struct {
	Username          string               // 全局用户名（可选）
	PrimaryEmail      string               // 主要邮箱（可选）
	PrimaryPhone      string               // 主要手机号（可选）
	PasswordEncrypted string               // 新建 person 时的加密密码
	PasswordMethod    model.PasswordMethod // 密码加密方式（取 model.PasswordMethod* 常量）
	// MustChangePassword 仅对**本次新建**的 person 生效：置 true 表示其持有临时密码，
	// 首次登录必须先改密。命中已有 person 时该字段被忽略——绝不改动既有账号的密码与登录方式。
	MustChangePassword bool
	Name               string // 姓名
	Avatar             string // 头像 URL
	CreatedBy          string // 创建人
}

// FindOrCreate person 聚合根的 find-or-create：
//   - 按 username/email/phone 全局唯一标识命中其一 → 返回已有 person（created=false），不符不建；
//   - 全部未命中 → 在 tx 事务内创建新 person（created=true），返回新建实体。
//
// 必须在调用方的事务 tx 内执行（空 tx 会 panic），返回的 person 与调用方后续
// 的 user/关联插入同属一个事务，保证原子性。查询/插入异常原样上抛。
// 并发撞唯一索引由 DB 部分唯一索引兜底，调用方自行处理冲突（如回查）。
func FindOrCreate(ctx context.Context, tx *gorm.DB, req *FindOrCreateReq) (*model.PersonEntity, bool, error) {
	personDao := dao.NewPersonDao().WithTx(tx)

	// 1. 按全局唯一标识命中已有 person，命中即复用
	if req.Username != "" {
		if p, err := personDao.GetByCond(ctx, &dao.PersonCond{Username: req.Username}); err != nil {
			return nil, false, err
		} else if p != nil && p.ID != "" {
			return p, false, nil
		}
	}
	if req.PrimaryEmail != "" {
		if p, err := personDao.GetByCond(ctx, &dao.PersonCond{PrimaryEmail: req.PrimaryEmail}); err != nil {
			return nil, false, err
		} else if p != nil && p.ID != "" {
			return p, false, nil
		}
	}
	if req.PrimaryPhone != "" {
		if p, err := personDao.GetByCond(ctx, &dao.PersonCond{PrimaryPhone: req.PrimaryPhone}); err != nil {
			return nil, false, err
		} else if p != nil && p.ID != "" {
			return p, false, nil
		}
	}

	// 2. 未命中：事务内创建
	entity := &model.PersonEntity{
		Username:           model.StrPtr(req.Username),
		PrimaryEmail:       model.StrPtr(req.PrimaryEmail),
		PrimaryPhone:       model.StrPtr(req.PrimaryPhone),
		PasswordEncrypted:  req.PasswordEncrypted,
		PasswordMethod:     req.PasswordMethod,
		MustChangePassword: req.MustChangePassword,
		Name:               req.Name,
		Avatar:             req.Avatar,
		Profile:            json.RawMessage(`{}`),
		CustomData:         json.RawMessage(`{}`),
		CreatedBy:          req.CreatedBy,
	}
	if err := personDao.Insert(ctx, entity); err != nil {
		return nil, false, err
	}
	return entity, true, nil
}
