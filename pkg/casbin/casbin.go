package casbin

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

const (
	// DefaultDataScope 默认数据权限范围
	DefaultDataScope = "own"
)

// NewEnforcer 创建并初始化 Casbin enforcer
// modelPath: RBAC 模型配置文件路径（如 "config/rbac_model.conf"）
// db: GORM 数据库实例
func NewEnforcer(modelPath string, db *gorm.DB) (*casbin.Enforcer, error) {
	// 创建 GORM 适配器
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin adapter: %w", err)
	}

	// 创建 enforcer
	enforcer, err := casbin.NewEnforcer(modelPath, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	// 从数据库加载策略
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load casbin policy: %w", err)
	}

	return enforcer, nil
}

// AddRoleForUser 为用户分配角色
func AddRoleForUser(e *casbin.Enforcer, userID string, roleName string) error {
	_, err := e.AddGroupingPolicy(userIDToSubject(userID), roleName)
	return err
}

// DeleteRoleForUser 删除用户的角色
func DeleteRoleForUser(e *casbin.Enforcer, userID string, roleName string) error {
	_, err := e.RemoveGroupingPolicy(userIDToSubject(userID), roleName)
	return err
}

// DeleteRolesForUser 删除用户的所有角色
func DeleteRolesForUser(e *casbin.Enforcer, userID string) error {
	_, err := e.DeleteRolesForUser(userIDToSubject(userID))
	return err
}

// GetRolesForUser 获取用户的所有角色
func GetRolesForUser(e *casbin.Enforcer, userID string) ([]string, error) {
	return e.GetRolesForUser(userIDToSubject(userID))
}

// GetUsersForRole 获取拥有某个角色的所有用户
func GetUsersForRole(e *casbin.Enforcer, roleName string) ([]string, error) {
	users, err := e.GetUsersForRole(roleName)
	if err != nil {
		return nil, err
	}

	var userIDs []string
	for _, user := range users {
		// 解析 "user:xxx" 格式
		var id string
		if _, err := fmt.Sscanf(user, "user:%s", &id); err == nil {
			userIDs = append(userIDs, id)
		}
	}

	return userIDs, nil
}

// GetImplicitPermissionsForUser 获取用户的所有权限（包括从角色继承的）
func GetImplicitPermissionsForUser(e *casbin.Enforcer, userID string) ([][]string, error) {
	return e.GetImplicitPermissionsForUser(userIDToSubject(userID))
}

// GetPermissionKeysForUser 获取用户的权限键列表（格式: resource:action）
func GetPermissionKeysForUser(e *casbin.Enforcer, userID string) ([]string, error) {
	perms, err := GetImplicitPermissionsForUser(e, userID)
	if err != nil {
		return nil, err
	}

	var keys []string
	seen := make(map[string]bool)

	for _, perm := range perms {
		if len(perm) >= 3 {
			key := perm[1] + ":" + perm[2]
			if !seen[key] {
				keys = append(keys, key)
				seen[key] = true
			}
		}
	}

	return keys, nil
}

// AddPermissionForRole 为角色添加权限
func AddPermissionForRole(e *casbin.Enforcer, roleName, resource, action string) error {
	_, err := e.AddPolicy(roleName, resource, action)
	return err
}

// DeletePermissionForRole 删除角色的权限
func DeletePermissionForRole(e *casbin.Enforcer, roleName, resource, action string) error {
	_, err := e.RemovePolicy(roleName, resource, action)
	return err
}

// DeletePermissionsForRole 删除角色的所有权限
func DeletePermissionsForRole(e *casbin.Enforcer, roleName string) error {
	_, err := e.DeletePermissionsForUser(roleName)
	return err
}

// GetPermissionsForRole 获取角色的所有权限
func GetPermissionsForRole(e *casbin.Enforcer, roleName string) ([][]string, error) {
	return e.GetPermissionsForUser(roleName)
}

// CheckPermission 检查用户是否有权限
func CheckPermission(e *casbin.Enforcer, userID, resource, action string) (bool, error) {
	return e.Enforce(userIDToSubject(userID), resource, action)
}

// userIDToSubject 将用户 ID 转换为 Casbin subject
func userIDToSubject(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

// SetRoleDataScope 设置角色的数据范围
// dataScope 可选值: "all", "own", "dept", "custom"
func SetRoleDataScope(e *casbin.Enforcer, roleName, dataScope string) error {
	// 删除旧的 data_scope 权限
	_, _ = e.RemoveFilteredPolicy(0, roleName, "data_scope")

	// 添加新的 data_scope 权限
	_, err := e.AddPolicy(roleName, "data_scope", dataScope)
	return err
}

// GetRoleDataScope 获取角色的数据范围
func GetRoleDataScope(e *casbin.Enforcer, roleName string) (string, error) {
	perms, err := e.GetPermissionsForUser(roleName)
	if err != nil {
		return "", err
	}

	for _, perm := range perms {
		if len(perm) >= 3 && perm[1] == "data_scope" {
			return perm[2], nil
		}
	}

	return DefaultDataScope, nil
}

// GetUserDataScope 获取用户的数据范围（取所有角色中权限最高的）
func GetUserDataScope(e *casbin.Enforcer, userID string) (string, error) {
	if userID == "" {
		return "all", nil
	}

	roles, err := e.GetRolesForUser(userIDToSubject(userID))
	if err != nil {
		return "", err
	}

	if len(roles) == 0 {
		return DefaultDataScope, nil
	}

	priority := map[string]int{
		"all":    4,
		"dept":   3,
		"own":    2,
		"custom": 1,
	}

	maxScope := DefaultDataScope
	maxPriority := 2

	for _, roleName := range roles {
		scope, err := GetRoleDataScope(e, roleName)
		if err != nil {
			continue
		}

		if p := priority[scope]; p > maxPriority {
			maxScope = scope
			maxPriority = p
		}
	}

	return maxScope, nil
}
