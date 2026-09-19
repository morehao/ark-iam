// Package svcdirectory 承载面向应用的只读目录查询（/v1/rp/directory/*）。
//
// 设计约束（见 .dsh/docs/specs/2026-09-18-rp-sdk-sharing-design.md 关键决策六）：
//   - 租户作用域**一律取令牌的 tenant_id**，不接受入参指定租户；
//   - 跨租户与不存在在响应上不可区分（防枚举）；
//   - 绝不把 Member.Status 用于放行/拒绝判定（挂起与否由令牌侧决定）；
//   - 实体查询复用 pkg/dao，租户隔离由插件按 ctx 作用域注入，缺失即 fail-closed。
package svcdirectory

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/rpapi/internal/dto/dtordirectory"
)

// DirectorySvc 是只读目录服务接口。
type DirectorySvc interface {
	// Members 批量取成员摘要（命中进 List，未命中进 Missing，缺项不影响整体成功）。
	Members(ctx *gin.Context, ids []string) (*dtordirectory.MemberListResp, error)
	// Member 取单个成员摘要；不存在/跨租户返回 gerror 的 NotExist 语义由调用方判定。
	Member(ctx *gin.Context, userID string) (*dtordirectory.Member, error)
	// DepartmentTree 取本租户部门树（仅 enable 节点）。
	DepartmentTree(ctx *gin.Context) ([]*dtordirectory.Department, error)
	// Roles 取本租户角色清单（≤MaxRoleListSize，超出置 truncated）。
	Roles(ctx *gin.Context) (*dtordirectory.RoleListResp, error)
}

type directorySvc struct{}

var _ DirectorySvc = (*directorySvc)(nil)

// NewDirectorySvc 构造只读目录服务。
func NewDirectorySvc() DirectorySvc {
	return &directorySvc{}
}

func (s *directorySvc) Members(ctx *gin.Context, ids []string) (*dtordirectory.MemberListResp, error) {
	resp := &dtordirectory.MemberListResp{List: []dtordirectory.Member{}, Missing: []string{}}
	unique := normalizeIDs(ids)
	if len(unique) == 0 {
		return resp, nil
	}
	users, err := dao.NewUserDao().GetListByCond(ctx, &dao.UserCond{IDs: unique})
	if err != nil {
		return nil, err
	}
	found := make(map[string]model.UserEntity, len(users))
	for i := range users {
		found[users[i].ID] = users[i]
	}
	deptNames, err := departmentNamesByUser(ctx, unique)
	if err != nil {
		return nil, err
	}
	for _, id := range unique {
		user, ok := found[id]
		if !ok {
			// 跨租户 id 与不存在 id 在 missing 中不可区分（刻意防枚举）。
			resp.Missing = append(resp.Missing, id)
			continue
		}
		resp.List = append(resp.List, toMember(user, deptNames[id]))
	}
	return resp, nil
}

func (s *directorySvc) Member(ctx *gin.Context, userID string) (*dtordirectory.Member, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}
	users, err := dao.NewUserDao().GetListByCond(ctx, &dao.UserCond{IDs: []string{userID}})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	deptNames, err := departmentNamesByUser(ctx, []string{userID})
	if err != nil {
		return nil, err
	}
	member := toMember(users[0], deptNames[userID])
	return &member, nil
}

func (s *directorySvc) DepartmentTree(ctx *gin.Context) ([]*dtordirectory.Department, error) {
	departments, err := dao.NewDepartmentDao().GetListByCond(ctx, &dao.DepartmentCond{
		Status: model.DeptNodeStatusEnable,
	})
	if err != nil {
		return nil, err
	}
	return buildDepartmentTree(departments), nil
}

func (s *directorySvc) Roles(ctx *gin.Context) (*dtordirectory.RoleListResp, error) {
	// 角色清单是无上界响应：多取一条即可判断是否被截断，避免 COUNT 查询。
	roles, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{})
	if err != nil {
		return nil, err
	}
	resp := &dtordirectory.RoleListResp{List: []dtordirectory.Role{}}
	sort.Slice(roles, func(i, j int) bool {
		if roles[i].AppID != roles[j].AppID {
			return roles[i].AppID < roles[j].AppID
		}
		return roles[i].Code < roles[j].Code
	})
	for i := range roles {
		if i >= dtordirectory.MaxRoleListSize {
			resp.Truncated = true
			break
		}
		resp.List = append(resp.List, dtordirectory.Role{
			Code:  string(roles[i].Code),
			Name:  roles[i].Name,
			AppID: roles[i].AppID,
		})
	}
	return resp, nil
}

// toMember 把实体转成对外只读摘要。
func toMember(user model.UserEntity, departmentNames []string) dtordirectory.Member {
	if departmentNames == nil {
		departmentNames = []string{}
	}
	return dtordirectory.Member{
		UserID:          user.ID,
		Name:            user.Name,
		Avatar:          user.Avatar,
		Status:          string(user.Status),
		UserType:        string(user.UserType),
		DepartmentNames: departmentNames,
	}
}

// departmentNamesByUser 批量取"用户 → 部门名列表"（避免 N+1）。
// 仅返回名称，不返回状态：目录只服务展示。
func departmentNamesByUser(ctx *gin.Context, userIDs []string) (map[string][]string, error) {
	if len(userIDs) == 0 {
		return map[string][]string{}, nil
	}
	relations, err := dao.NewDepartmentUserDao().GetListByCond(ctx, &dao.DepartmentUserCond{UserIDs: userIDs})
	if err != nil {
		return nil, err
	}
	if len(relations) == 0 {
		return map[string][]string{}, nil
	}
	deptIDs := make([]string, 0, len(relations))
	seen := make(map[string]bool, len(relations))
	for i := range relations {
		if relations[i].DepartmentID == "" || seen[relations[i].DepartmentID] {
			continue
		}
		seen[relations[i].DepartmentID] = true
		deptIDs = append(deptIDs, relations[i].DepartmentID)
	}
	departments, err := dao.NewDepartmentDao().GetListByCond(ctx, &dao.DepartmentCond{IDs: deptIDs})
	if err != nil {
		return nil, err
	}
	nameByID := make(map[string]string, len(departments))
	for i := range departments {
		nameByID[departments[i].ID] = departments[i].Name
	}
	out := make(map[string][]string, len(userIDs))
	for i := range relations {
		name, ok := nameByID[relations[i].DepartmentID]
		if !ok || name == "" {
			continue
		}
		out[relations[i].UserID] = append(out[relations[i].UserID], name)
	}
	for id := range out {
		sort.Strings(out[id])
	}
	return out, nil
}

// buildDepartmentTree 由扁平列表构建树；父节点缺失（不可见/已删）的节点按根节点处理，
// 保证"存在子节点却整棵子树消失"这种静默丢数据不会发生。
func buildDepartmentTree(departments model.DepartmentEntityList) []*dtordirectory.Department {
	nodes := make(map[string]*dtordirectory.Department, len(departments))
	for i := range departments {
		nodes[departments[i].ID] = &dtordirectory.Department{
			ID:       departments[i].ID,
			ParentID: departments[i].ParentID,
			Name:     departments[i].Name,
		}
	}
	roots := make([]*dtordirectory.Department, 0, len(departments))
	order := make([]string, 0, len(departments))
	for i := range departments {
		order = append(order, departments[i].ID)
	}
	sort.SliceStable(order, func(i, j int) bool {
		left, right := nodes[order[i]], nodes[order[j]]
		if left.ParentID != right.ParentID {
			return left.ParentID < right.ParentID
		}
		return left.Name < right.Name
	})
	for _, id := range order {
		node := nodes[id]
		parent, ok := nodes[node.ParentID]
		if !ok || node.ParentID == "" || node.ParentID == node.ID {
			roots = append(roots, node)
			continue
		}
		parent.Children = append(parent.Children, *node)
	}
	return roots
}

// normalizeIDs 去空白、去重、保序（保序便于缺项列表可预测）。
func normalizeIDs(raw []string) []string {
	out := make([]string, 0, len(raw))
	seen := make(map[string]bool, len(raw))
	for _, id := range raw {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
