package casbin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestEnforcer(t *testing.T) (*gorm.DB, string, func()) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	tmpDir := t.TempDir()
	modelPath := filepath.Join(tmpDir, "rbac_model.conf")

	modelContent := `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
`
	err = os.WriteFile(modelPath, []byte(modelContent), 0644)
	require.NoError(t, err)

	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return db, modelPath, cleanup
}

func TestNewEnforcer(t *testing.T) {
	db, modelPath, cleanup := setupTestEnforcer(t)
	defer cleanup()

	e, err := NewEnforcer(modelPath, db)
	require.NoError(t, err)
	assert.NotNil(t, e)
}

func TestNewEnforcer_InvalidModelPath(t *testing.T) {
	db, _, cleanup := setupTestEnforcer(t)
	defer cleanup()

	e, err := NewEnforcer("/nonexistent/path/model.conf", db)
	assert.Error(t, err)
	assert.Nil(t, e)
}

func TestAddRoleForUser(t *testing.T) {
	db, modelPath, cleanup := setupTestEnforcer(t)
	defer cleanup()

	e, err := NewEnforcer(modelPath, db)
	require.NoError(t, err)

	err = AddRoleForUser(e, "1", "admin")
	require.NoError(t, err)

	roles, err := GetRolesForUser(e, "1")
	require.NoError(t, err)
	assert.Contains(t, roles, "admin")
}

func TestDeleteRoleForUser(t *testing.T) {
	db, modelPath, cleanup := setupTestEnforcer(t)
	defer cleanup()

	e, err := NewEnforcer(modelPath, db)
	require.NoError(t, err)

	err = AddRoleForUser(e, "1", "admin")
	require.NoError(t, err)

	err = DeleteRoleForUser(e, "1", "admin")
	require.NoError(t, err)

	roles, err := GetRolesForUser(e, "1")
	require.NoError(t, err)
	assert.NotContains(t, roles, "admin")
}

func TestDeleteRolesForUser(t *testing.T) {
	db, modelPath, cleanup := setupTestEnforcer(t)
	defer cleanup()

	e, err := NewEnforcer(modelPath, db)
	require.NoError(t, err)

	err = AddRoleForUser(e, "1", "admin")
	require.NoError(t, err)
	err = AddRoleForUser(e, "1", "editor")
	require.NoError(t, err)

	err = DeleteRolesForUser(e, "1")
	require.NoError(t, err)

	roles, err := GetRolesForUser(e, "1")
	require.NoError(t, err)
	assert.Empty(t, roles)
}

func TestCheckPermission(t *testing.T) {
	db, modelPath, cleanup := setupTestEnforcer(t)
	defer cleanup()

	e, err := NewEnforcer(modelPath, db)
	require.NoError(t, err)

	err = AddRoleForUser(e, "1", "admin")
	require.NoError(t, err)

	err = AddPermissionForRole(e, "admin", "users", "read")
	require.NoError(t, err)

	allowed, err := CheckPermission(e, "1", "users", "read")
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = CheckPermission(e, "1", "users", "delete")
	require.NoError(t, err)
	assert.False(t, allowed)

	allowed, err = CheckPermission(e, "2", "users", "read")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestAddPermissionForRole(t *testing.T) {
	db, modelPath, cleanup := setupTestEnforcer(t)
	defer cleanup()

	e, err := NewEnforcer(modelPath, db)
	require.NoError(t, err)

	err = AddPermissionForRole(e, "admin", "users", "create")
	require.NoError(t, err)

	perms, err := GetPermissionsForRole(e, "admin")
	require.NoError(t, err)
	assert.Len(t, perms, 1)
	assert.Equal(t, []string{"admin", "users", "create"}, perms[0])
}

func TestDeletePermissionForRole(t *testing.T) {
	db, modelPath, cleanup := setupTestEnforcer(t)
	defer cleanup()

	e, err := NewEnforcer(modelPath, db)
	require.NoError(t, err)

	err = AddPermissionForRole(e, "admin", "users", "create")
	require.NoError(t, err)

	err = DeletePermissionForRole(e, "admin", "users", "create")
	require.NoError(t, err)

	perms, err := GetPermissionsForRole(e, "admin")
	require.NoError(t, err)
	assert.Empty(t, perms)
}

func TestDeletePermissionsForRole(t *testing.T) {
	db, modelPath, cleanup := setupTestEnforcer(t)
	defer cleanup()

	e, err := NewEnforcer(modelPath, db)
	require.NoError(t, err)

	err = AddPermissionForRole(e, "admin", "users", "create")
	require.NoError(t, err)
	err = AddPermissionForRole(e, "admin", "users", "read")
	require.NoError(t, err)

	err = DeletePermissionsForRole(e, "admin")
	require.NoError(t, err)

	perms, err := GetPermissionsForRole(e, "admin")
	require.NoError(t, err)
	assert.Empty(t, perms)
}

func TestGetImplicitPermissionsForUser(t *testing.T) {
	db, modelPath, cleanup := setupTestEnforcer(t)
	defer cleanup()

	e, err := NewEnforcer(modelPath, db)
	require.NoError(t, err)

	err = AddRoleForUser(e, "1", "admin")
	require.NoError(t, err)
	err = AddPermissionForRole(e, "admin", "users", "read")
	require.NoError(t, err)
	err = AddPermissionForRole(e, "admin", "users", "write")
	require.NoError(t, err)

	perms, err := GetImplicitPermissionsForUser(e, "1")
	require.NoError(t, err)
	assert.Len(t, perms, 2)
}

func TestGetPermissionKeysForUser(t *testing.T) {
	db, modelPath, cleanup := setupTestEnforcer(t)
	defer cleanup()

	e, err := NewEnforcer(modelPath, db)
	require.NoError(t, err)

	err = AddRoleForUser(e, "1", "admin")
	require.NoError(t, err)
	err = AddPermissionForRole(e, "admin", "users", "read")
	require.NoError(t, err)
	err = AddPermissionForRole(e, "admin", "orders", "write")
	require.NoError(t, err)

	keys, err := GetPermissionKeysForUser(e, "1")
	require.NoError(t, err)
	assert.Contains(t, keys, "users:read")
	assert.Contains(t, keys, "orders:write")
}

func TestDataScope(t *testing.T) {
	db, modelPath, cleanup := setupTestEnforcer(t)
	defer cleanup()

	e, err := NewEnforcer(modelPath, db)
	require.NoError(t, err)

	t.Run("SetRoleDataScope", func(t *testing.T) {
		err := SetRoleDataScope(e, "admin", "all")
		require.NoError(t, err)

		scope, err := GetRoleDataScope(e, "admin")
		require.NoError(t, err)
		assert.Equal(t, "all", scope)
	})

	t.Run("GetRoleDataScope_Default", func(t *testing.T) {
		scope, err := GetRoleDataScope(e, "nonexistent")
		require.NoError(t, err)
		assert.Equal(t, DefaultDataScope, scope)
	})

	t.Run("GetUserDataScope_EmptyUserID", func(t *testing.T) {
		scope, err := GetUserDataScope(e, "")
		require.NoError(t, err)
		assert.Equal(t, "all", scope)
	})

	t.Run("GetUserDataScope_NoRoles", func(t *testing.T) {
		scope, err := GetUserDataScope(e, "999")
		require.NoError(t, err)
		assert.Equal(t, DefaultDataScope, scope)
	})

	t.Run("GetUserDataScope_WithRoles", func(t *testing.T) {
		err := AddRoleForUser(e, "2", "editor")
		require.NoError(t, err)
		err = SetRoleDataScope(e, "editor", "dept")
		require.NoError(t, err)

		err = AddRoleForUser(e, "2", "viewer")
		require.NoError(t, err)
		err = SetRoleDataScope(e, "viewer", "own")
		require.NoError(t, err)

		scope, err := GetUserDataScope(e, "2")
		require.NoError(t, err)
		assert.Equal(t, "dept", scope)
	})

	t.Run("GetUserDataScope_AllPriority", func(t *testing.T) {
		err := AddRoleForUser(e, "3", "superadmin")
		require.NoError(t, err)
		err = SetRoleDataScope(e, "superadmin", "all")
		require.NoError(t, err)

		err = AddRoleForUser(e, "3", "user")
		require.NoError(t, err)
		err = SetRoleDataScope(e, "user", "own")
		require.NoError(t, err)

		scope, err := GetUserDataScope(e, "3")
		require.NoError(t, err)
		assert.Equal(t, "all", scope)
	})
}

func TestGetUsersForRole(t *testing.T) {
	db, modelPath, cleanup := setupTestEnforcer(t)
	defer cleanup()

	e, err := NewEnforcer(modelPath, db)
	require.NoError(t, err)

	err = AddRoleForUser(e, "1", "admin")
	require.NoError(t, err)
	err = AddRoleForUser(e, "2", "admin")
	require.NoError(t, err)

	users, err := GetUsersForRole(e, "admin")
	require.NoError(t, err)
	assert.Contains(t, users, "1")
	assert.Contains(t, users, "2")
}
