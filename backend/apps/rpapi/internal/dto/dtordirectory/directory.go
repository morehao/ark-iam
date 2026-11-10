package dtordirectory

import "github.com/morehao/ark-iam/contract/directory"

// 本文件是目录 wire 契约在 rpapi 内部的**类型别名门面**：DTO 真源已下沉至
// contract/directory（服务端与 sdk/rp/directory 客户端共享同一份定义）。
//
// 使用别名而非重新定义，保证 dtordirectory.Member 与 directory.Member 完全同一类型。
//
// 注意：rpapi 应用在 T4 并入 sdk/server 的 DomainDirectory 后整体下线，
// 此门面仅为 T0 阶段保持编译而保留。

type (
	// Member 是成员摘要（对外只读视图）。
	Member = directory.Member
	// MemberListResp 是批量成员查询响应。
	MemberListResp = directory.MemberListResp
	// Department 是部门树节点。
	Department = directory.Department
	// Role 是角色清单条目。
	Role = directory.Role
	// RoleListResp 是角色清单响应。
	RoleListResp = directory.RoleListResp
)

const (
	// MaxBatchMemberIDs 是批量成员查询的 id 上限。
	MaxBatchMemberIDs = directory.MaxBatchMemberIDs
	// MaxRoleListSize 是角色清单的返回上限。
	MaxRoleListSize = directory.MaxRoleListSize
)
