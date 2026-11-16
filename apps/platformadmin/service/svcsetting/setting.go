package svcsetting

import (
	"context"
	"errors"

	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtosetting"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gsetting"
	"github.com/morehao/golib/gutil"
)

// MaskedValue 是敏感配置项回显的掩码值。更新时收到该值表示「不修改该字段」。
const MaskedValue = "******"

// AllowedGroups 是页面可管理的配置分组白名单。
//
// 新增一类运行时配置时，只需在此登记分组名（如后续加 "smtp" 等），
// 管理面零新增路由。见设计 §3.3。
var AllowedGroups = []string{"notify", "security.login"}

// SettingSvc 是运行时配置中心的管理面服务（面向页面）。
//
// 薄封装 golib gsetting.AdminAPI：只做「分组白名单校验 + 敏感字段掩码」两件事，
// CRUD 本身完全复用 gsetting（见设计 §6，不自造轮子）。
type SettingSvc interface {
	Create(ctx context.Context, req *dtosetting.SettingCreateReq) error
	Update(ctx context.Context, req *dtosetting.SettingUpdateReq) error
	UpdateStatus(ctx context.Context, req *dtosetting.SettingUpdateStatusReq) error
	Delete(ctx context.Context, req *dtosetting.SettingDeleteReq) error
	Detail(ctx context.Context, req *dtosetting.SettingDetailReq) (*dtosetting.SettingDetailResp, error)
	PageList(ctx context.Context, req *dtosetting.SettingPageListReq) (*dtosetting.SettingPageListResp, error)
}

type settingSvc struct{}

var _ SettingSvc = (*settingSvc)(nil)

// NewSettingSvc 构造配置项服务。
//
// 说明：AdminAPI 用 gsetting.GetAdmin()（进程级单例，由 sdk/server 的 gsetting.New 初始化）。
// 一进程一实例下与业务消费者共用同一 store；仅页面侧调用 GetAdmin，业务侧只用 Get*（见 R2）。
func NewSettingSvc() SettingSvc {
	return &settingSvc{}
}

// allowedGroup 判断分组是否在白名单内。
func allowedGroup(group string) bool {
	for _, g := range AllowedGroups {
		if g == group {
			return true
		}
	}
	return false
}

func (svc *settingSvc) Create(ctx context.Context, req *dtosetting.SettingCreateReq) error {
	if !allowedGroup(req.Group) {
		return code.GetError(code.SettingGroupNotAllowedError)
	}
	admin := gsetting.GetAdmin()
	if err := admin.Create(ctx, &gsetting.CreateReq{
		Group:       req.Group,
		Key:         req.Key,
		ValueType:   req.ValueType,
		Value:       req.Value,
		Encrypted:   req.Encrypted,
		Status:      req.Status,
		Description: req.Description,
	}); err != nil {
		glog.Errorf(ctx, "[svcsetting.Create] gsetting create fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SettingCreateError)
	}
	return nil
}

func (svc *settingSvc) Update(ctx context.Context, req *dtosetting.SettingUpdateReq) error {
	admin := gsetting.GetAdmin()
	info, err := admin.GetByID(ctx, req.SettingID)
	if err != nil {
		glog.Errorf(ctx, "[svcsetting.Update] GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SettingUpdateError)
	}
	if info == nil {
		return code.GetError(code.SettingNotExistError)
	}
	if !allowedGroup(info.GroupName) {
		return code.GetError(code.SettingGroupNotAllowedError)
	}

	ureq := &gsetting.UpdateReq{
		Status:      req.Status,
		Description: req.Description,
	}
	// 掩码值表示「不改值」；未传值（空串）同样不改。
	if req.Value != "" && req.Value != MaskedValue {
		ureq.Value = req.Value
		ureq.Encrypted = req.Encrypted
	}
	if err := admin.Update(ctx, req.SettingID, ureq); err != nil {
		glog.Errorf(ctx, "[svcsetting.Update] gsetting update fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SettingUpdateError)
	}
	return nil
}

func (svc *settingSvc) UpdateStatus(ctx context.Context, req *dtosetting.SettingUpdateStatusReq) error {
	admin := gsetting.GetAdmin()
	info, err := admin.GetByID(ctx, req.SettingID)
	if err != nil {
		glog.Errorf(ctx, "[svcsetting.UpdateStatus] GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SettingUpdateError)
	}
	if info == nil {
		return code.GetError(code.SettingNotExistError)
	}
	if !allowedGroup(info.GroupName) {
		return code.GetError(code.SettingGroupNotAllowedError)
	}
	if err := admin.UpdateStatus(ctx, req.SettingID, gsetting.Status(req.Status)); err != nil {
		// 状态非法与系统错误分流：gsetting 的 errInvalidStatus 属业务边界。
		if errors.Is(err, gsetting.ErrSettingNotFound) {
			return code.GetError(code.SettingNotExistError)
		}
		glog.Errorf(ctx, "[svcsetting.UpdateStatus] gsetting update status fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SettingStatusInvalidError)
	}
	return nil
}

func (svc *settingSvc) Delete(ctx context.Context, req *dtosetting.SettingDeleteReq) error {
	admin := gsetting.GetAdmin()
	info, err := admin.GetByID(ctx, req.SettingID)
	if err != nil {
		glog.Errorf(ctx, "[svcsetting.Delete] GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SettingDeleteError)
	}
	if info == nil {
		return code.GetError(code.SettingNotExistError)
	}
	if !allowedGroup(info.GroupName) {
		return code.GetError(code.SettingGroupNotAllowedError)
	}
	if err := admin.Delete(ctx, req.SettingID); err != nil {
		glog.Errorf(ctx, "[svcsetting.Delete] gsetting delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SettingDeleteError)
	}
	return nil
}

func (svc *settingSvc) Detail(ctx context.Context, req *dtosetting.SettingDetailReq) (*dtosetting.SettingDetailResp, error) {
	admin := gsetting.GetAdmin()
	info, err := admin.GetByID(ctx, req.SettingID)
	if err != nil {
		glog.Errorf(ctx, "[svcsetting.Detail] GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.SettingGetDetailError)
	}
	if info == nil {
		return nil, code.GetError(code.SettingNotExistError)
	}
	if !allowedGroup(info.GroupName) {
		return nil, code.GetError(code.SettingGroupNotAllowedError)
	}
	item := toItem(info)
	return &dtosetting.SettingDetailResp{SettingItem: *item}, nil
}

func (svc *settingSvc) PageList(ctx context.Context, req *dtosetting.SettingPageListReq) (*dtosetting.SettingPageListResp, error) {
	// 分组过滤：显式指定则必须是白名单内；未指定则限定在全部白名单分组内。
	groups := AllowedGroups
	if req.Group != "" {
		if !allowedGroup(req.Group) {
			return nil, code.GetError(code.SettingGroupNotAllowedError)
		}
		groups = []string{req.Group}
	}

	admin := gsetting.GetAdmin()
	all := make([]*dtosetting.SettingItem, 0)
	var total int64
	for _, g := range groups {
		cond := &gsetting.SettingCond{
			Group:     g,
			Key:       req.Key,
			ValueType: req.ValueType,
			Status:    req.Status,
		}
		cond.Page = req.Page
		cond.PageSize = req.PageSize
		page, err := admin.ListPage(ctx, cond)
		if err != nil {
			glog.Errorf(ctx, "[svcsetting.PageList] ListPage fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return nil, code.GetError(code.SettingGetPageListError)
		}
		for _, info := range page.List {
			all = append(all, toItem(info))
		}
		total += page.Total
	}
	return &dtosetting.SettingPageListResp{List: all, Total: total}, nil
}

// toItem 把 gsetting.SettingInfo 转对外契约；加密项 value 掩码。
func toItem(info *gsetting.SettingInfo) *dtosetting.SettingItem {
	value := info.Value
	if info.EncryptionMode == gsetting.EncryptionModeEncrypted {
		value = MaskedValue
	}
	return &dtosetting.SettingItem{
		ID:             info.ID,
		Group:          info.GroupName,
		Key:            info.Key,
		ValueType:      string(info.ValueType),
		Value:          value,
		EncryptionMode: string(info.EncryptionMode),
		Description:    info.Description,
		Status:         string(info.Status),
		CreatedAt:      info.CreatedAt,
		UpdatedAt:      info.UpdatedAt,
	}
}
