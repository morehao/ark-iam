package svcsetting

import (
	"context"
	"errors"
	"sort"

	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtosetting"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gsetting"
	"github.com/morehao/golib/gutil"
)

// MaskedValue 是敏感配置项回显的掩码值。更新时收到该值表示「不修改该字段」。
const MaskedValue = "******"

// settingGroups 是页面可管理分组的**唯一事实源**（白名单 + 展示元数据合一）。
//
// 新增一类运行时配置时，只需在此登记一条（分组名 + 中文标题 + 说明），
// AllowedGroups 由它派生、Groups() 直接下发，前端零改动。见设计 §3.2。
var settingGroups = []dtosetting.SettingGroupMeta{
	{
		Group:       "notify",
		Description: "通知渠道（临时密码投递）：账号初始/重置临时密码的投递渠道，总闸关闭时回退为回显明文",
	},
	{
		Group:       "security.login",
		Description: "登录安全：登录失败锁定阈值与登录接口频率限流参数，保存后下一请求生效",
	},
}

// AllowedGroups 是页面可管理的配置分组白名单，由 settingGroups 派生（保持既有对外常量语义）。
var AllowedGroups = func() []string {
	out := make([]string, 0, len(settingGroups))
	for _, g := range settingGroups {
		out = append(out, g.Group)
	}
	return out
}()

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
	// Groups 下发页面可管理的配置分组元数据（白名单 + 展示信息，供前端渲染分组名/说明与新增下拉）。
	Groups(ctx context.Context) (*dtosetting.SettingGroupListResp, error)
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

// Groups 下发页面可管理的配置分组元数据。纯内存、无 DB 访问。
func (svc *settingSvc) Groups(_ context.Context) (*dtosetting.SettingGroupListResp, error) {
	list := make([]*dtosetting.SettingGroupMeta, 0, len(settingGroups))
	for i := range settingGroups {
		g := settingGroups[i]
		list = append(list, &g)
	}
	return &dtosetting.SettingGroupListResp{List: list}, nil
}

// PageList 分页查询配置项，支持 group/key/status 过滤。
//
// 分页语义（修复历史缺陷）：gsetting 的 ListPage 只能按单个 group 过滤，原实现对每个分组
// 各带同一 Page/PageSize 查询再拼接，导致「不指定 group 时 total 累加、单页条数 = PageSize × 分组数」
// 并非真正的全局分页。由于配置项总量极小（数十条量级、长期稳定），这里改为
// 「逐组各取全量（PageSize 取 MaxPageSize 上限）→ 内存合并 → 按 (group,key) 稳定排序 → 全局切片」，
// 得到确定性全局分页。将来配置项上量时应改为 gsetting 支持 GroupIn 的单次分页查询。
//
// 指定 group 时仍走单组原生分页（该组内分页本就正确）。
func (svc *settingSvc) PageList(ctx context.Context, req *dtosetting.SettingPageListReq) (*dtosetting.SettingPageListResp, error) {
	// 分组过滤：显式指定则必须是白名单内；未指定则限定在全部白名单分组内。
	groups := AllowedGroups
	if req.Group != "" {
		if !allowedGroup(req.Group) {
			return nil, code.GetError(code.SettingGroupNotAllowedError)
		}
		groups = []string{req.Group}
	}

	page, pageSize := req.Page, req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	admin := gsetting.GetAdmin()

	// 单组：走 gsetting 原生分页，组内分页正确。
	if len(groups) == 1 {
		cond := &gsetting.SettingCond{
			Group:     groups[0],
			Key:       req.Key,
			ValueType: req.ValueType,
			Status:    req.Status,
		}
		cond.Page = page
		cond.PageSize = pageSize
		resp, err := admin.ListPage(ctx, cond)
		if err != nil {
			glog.Errorf(ctx, "[svcsetting.PageList] ListPage fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return nil, code.GetError(code.SettingGetPageListError)
		}
		list := make([]*dtosetting.SettingItem, 0, len(resp.List))
		for _, info := range resp.List {
			list = append(list, toItem(info))
		}
		return &dtosetting.SettingPageListResp{List: list, Total: resp.Total}, nil
	}

	// 多组（含未指定 group）：逐组取全量后内存合并 + 全局切片。
	merged := make([]*gsetting.SettingInfo, 0)
	for _, g := range groups {
		cond := &gsetting.SettingCond{
			Group:     g,
			Key:       req.Key,
			ValueType: req.ValueType,
			Status:    req.Status,
		}
		cond.Page = 1
		cond.PageSize = gormdao.MaxPageSize
		resp, err := admin.ListPage(ctx, cond)
		if err != nil {
			glog.Errorf(ctx, "[svcsetting.PageList] ListPage fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return nil, code.GetError(code.SettingGetPageListError)
		}
		merged = append(merged, resp.List...)
	}

	// 稳定排序：先按 group 白名单顺序（groups 的顺序），再按 key，保证跨页无重复无遗漏。
	groupOrder := make(map[string]int, len(groups))
	for i, g := range groups {
		groupOrder[g] = i
	}
	sort.SliceStable(merged, func(i, j int) bool {
		gi, gj := groupOrder[merged[i].GroupName], groupOrder[merged[j].GroupName]
		if gi != gj {
			return gi < gj
		}
		return merged[i].Key < merged[j].Key
	})

	total := int64(len(merged))
	start := (page - 1) * pageSize
	if start > len(merged) {
		start = len(merged)
	}
	end := start + pageSize
	if end > len(merged) {
		end = len(merged)
	}
	slice := merged[start:end]

	list := make([]*dtosetting.SettingItem, 0, len(slice))
	for _, info := range slice {
		list = append(list, toItem(info))
	}
	return &dtosetting.SettingPageListResp{List: list, Total: total}, nil
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
