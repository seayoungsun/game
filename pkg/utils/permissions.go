package utils

// 权限代码常量定义
const (
	// 仪表盘
	PermissionDashboardView = "admin:dashboard:view"

	// 用户管理
	PermissionUsersList   = "admin:users:list"
	PermissionUsersDetail = "admin:users:detail"
	PermissionUsersUpdate = "admin:users:update"
	PermissionUsersBan    = "admin:users:ban"

	// 充值订单
	PermissionRechargeOrdersList   = "admin:recharge_orders:list"
	PermissionRechargeOrdersDetail = "admin:recharge_orders:detail"

	// 提现订单
	PermissionWithdrawOrdersList   = "admin:withdraw_orders:list"
	PermissionWithdrawOrdersDetail = "admin:withdraw_orders:detail"
	PermissionWithdrawOrdersAudit  = "admin:withdraw_orders:audit"

	// 充值地址
	PermissionDepositAddressesList = "admin:deposit_addresses:list"

	// 支付管理
	PermissionPaymentsCollect      = "admin:payments:collect"
	PermissionPaymentsBatchCollect = "admin:payments:batch_collect"

	// 系统管理 - 角色
	PermissionRolesList             = "admin:roles:list"
	PermissionRolesCreate           = "admin:roles:create"
	PermissionRolesUpdate           = "admin:roles:update"
	PermissionRolesDelete           = "admin:roles:delete"
	PermissionRolesAssignPermission = "admin:roles:assign_permission"

	// 系统管理 - 管理员
	PermissionAdminsList       = "admin:admins:list"
	PermissionAdminsCreate     = "admin:admins:create"
	PermissionAdminsUpdate     = "admin:admins:update"
	PermissionAdminsDelete     = "admin:admins:delete"
	PermissionAdminsAssignRole = "admin:admins:assign_role"
)

// 角色代码常量
const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleOperator   = "operator"
	RoleAuditor    = "auditor"
)

// GetAllPermissions 获取所有权限代码列表
func GetAllPermissions() []string {
	return []string{
		PermissionDashboardView,
		PermissionUsersList,
		PermissionUsersDetail,
		PermissionUsersUpdate,
		PermissionUsersBan,
		PermissionRechargeOrdersList,
		PermissionRechargeOrdersDetail,
		PermissionWithdrawOrdersList,
		PermissionWithdrawOrdersDetail,
		PermissionWithdrawOrdersAudit,
		PermissionDepositAddressesList,
		PermissionPaymentsCollect,
		PermissionPaymentsBatchCollect,
		PermissionRolesList,
		PermissionRolesCreate,
		PermissionRolesUpdate,
		PermissionRolesDelete,
		PermissionRolesAssignPermission,
		PermissionAdminsList,
		PermissionAdminsCreate,
		PermissionAdminsUpdate,
		PermissionAdminsDelete,
		PermissionAdminsAssignRole,
	}
}
