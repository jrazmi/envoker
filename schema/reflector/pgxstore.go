package reflector

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore implements the Store interface for PostgreSQL databases using pgx
type PostgresStore struct {
	pool   *pgxpool.Pool
	dbName string
	ctx    context.Context
}

// NewPostgresStore creates a new PostgreSQL store from an existing connection pool
func NewPostgresStore(ctx context.Context, pool *pgxpool.Pool, dbName string) *PostgresStore {
	return &PostgresStore{
		pool:   pool,
		dbName: dbName,
		ctx:    ctx,
	}
}

// GetDatabaseName implements the Store interface
func (s *PostgresStore) GetDatabaseName() string {
	return s.dbName
}

// GetSourceType implements the Store interface
func (s *PostgresStore) GetSourceType() string {
	return "postgres"
}

// GetTables implements the Store interface
func (s *PostgresStore) GetTables(schemaName string) ([]string, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = $1
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := s.pool.Query(s.ctx, query, schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, rows.Err()
}

// GetColumns implements the Store interface
func (s *PostgresStore) GetColumns(schemaName, tableName string) ([]ColumnInfo, error) {
	query := `
		SELECT
			c.column_name,
			c.data_type,
			c.udt_name,
			c.is_nullable,
			c.column_default,
			c.character_maximum_length,
			c.numeric_precision,
			c.numeric_scale,
			pgd.description
		FROM information_schema.columns c
		LEFT JOIN pg_catalog.pg_statio_all_tables pst
			ON c.table_schema = pst.schemaname
			AND c.table_name = pst.relname
		LEFT JOIN pg_catalog.pg_description pgd
			ON pgd.objoid = pst.relid
			AND pgd.objsubid = c.ordinal_position
		WHERE c.table_schema = $1
		  AND c.table_name = $2
		ORDER BY c.ordinal_position
	`

	rows, err := s.pool.Query(s.ctx, query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var dataType, udtName string
		var isNullable string
		var defaultValue *string
		var maxLength *int64
		var precision *int64
		var scale *int64
		var comment *string

		err := rows.Scan(
			&col.Name,
			&dataType,
			&udtName,
			&isNullable,
			&defaultValue,
			&maxLength,
			&precision,
			&scale,
			&comment,
		)
		if err != nil {
			return nil, err
		}

		col.DBType = normalizePostgresType(dataType, udtName, maxLength, precision, scale)
		col.IsNullable = isNullable == "YES"

		if defaultValue != nil {
			col.HasDefault = true
			col.DefaultValue = cleanDefaultValue(*defaultValue)
		}

		if maxLength != nil {
			col.MaxLength = int(*maxLength)
		}

		if precision != nil {
			col.Precision = int(*precision)
		}

		if scale != nil {
			col.Scale = int(*scale)
		}

		if comment != nil {
			col.Comment = *comment
		}

		col.GoType, col.GoImport = mapPostgreSQLTypeToGo(col.DBType, col.IsNullable)
		col.ValidationTags = deriveValidationTags(col)

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// GetPrimaryKey implements the Store interface
func (s *PostgresStore) GetPrimaryKey(schemaName, tableName string, columns []ColumnInfo) (*PrimaryKeyInfo, error) {
	query := `
		SELECT
			kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
		  AND tc.table_schema = $1
		  AND tc.table_name = $2
		ORDER BY kcu.ordinal_position
		LIMIT 1
	`

	var columnName string
	err := s.pool.QueryRow(s.ctx, query, schemaName, tableName).Scan(&columnName)
	if err != nil {
		return nil, fmt.Errorf("no primary key found")
	}

	var pkColumn *ColumnInfo
	for i := range columns {
		if columns[i].Name == columnName {
			pkColumn = &columns[i]
			break
		}
	}

	if pkColumn == nil {
		return nil, fmt.Errorf("primary key column %s not found in columns list", columnName)
	}

	return &PrimaryKeyInfo{
		Column:      columnName,
		DBType:      pkColumn.DBType,
		GoType:      pkColumn.GoType,
		HasDefault:  pkColumn.HasDefault,
		DefaultExpr: pkColumn.DefaultValue,
	}, nil
}

// GetForeignKeys implements the Store interface
func (s *PostgresStore) GetForeignKeys(schemaName, tableName string) ([]ForeignKeyInfo, error) {
	query := `
		SELECT
			kcu.column_name,
			ccu.table_schema AS foreign_table_schema,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name,
			rc.update_rule,
			rc.delete_rule
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		JOIN information_schema.referential_constraints AS rc
			ON rc.constraint_name = tc.constraint_name
			AND rc.constraint_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema = $1
		  AND tc.table_name = $2
		ORDER BY kcu.ordinal_position
	`

	rows, err := s.pool.Query(s.ctx, query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fks []ForeignKeyInfo
	for rows.Next() {
		var fk ForeignKeyInfo
		err := rows.Scan(
			&fk.ColumnName,
			&fk.RefSchema,
			&fk.RefTable,
			&fk.RefColumn,
			&fk.OnUpdate,
			&fk.OnDelete,
		)
		if err != nil {
			return nil, err
		}

		fk.OnUpdate = strings.ToUpper(strings.ReplaceAll(fk.OnUpdate, " ", "_"))
		fk.OnDelete = strings.ToUpper(strings.ReplaceAll(fk.OnDelete, " ", "_"))

		fks = append(fks, fk)
	}

	return fks, rows.Err()
}

// GetIndexes implements the Store interface
func (s *PostgresStore) GetIndexes(schemaName, tableName string) ([]IndexInfo, error) {
	query := `
		SELECT
			i.relname AS index_name,
			am.amname AS index_method,
			ix.indisunique AS is_unique,
			ARRAY_AGG(a.attname ORDER BY array_position(ix.indkey, a.attnum)) AS column_names
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_am am ON i.relam = am.oid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE n.nspname = $1
		  AND t.relname = $2
		  AND NOT ix.indisprimary
		GROUP BY i.relname, am.amname, ix.indisunique
		ORDER BY i.relname
	`

	rows, err := s.pool.Query(s.ctx, query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var idx IndexInfo
		var columns []string

		err := rows.Scan(
			&idx.Name,
			&idx.Method,
			&idx.Unique,
			&columns,
		)
		if err != nil {
			return nil, err
		}

		idx.Columns = columns
		indexes = append(indexes, idx)
	}

	return indexes, rows.Err()
}

// GetConstraints implements the Store interface
func (s *PostgresStore) GetConstraints(schemaName, tableName string) ([]ConstraintInfo, error) {
	query := `
		SELECT
			con.conname AS constraint_name,
			CASE con.contype
				WHEN 'c' THEN 'CHECK'
				WHEN 'u' THEN 'UNIQUE'
				WHEN 'x' THEN 'EXCLUDE'
			END AS constraint_type,
			pg_get_constraintdef(con.oid) AS constraint_definition
		FROM pg_constraint con
		JOIN pg_namespace nsp ON nsp.oid = con.connamespace
		JOIN pg_class cls ON cls.oid = con.conrelid
		WHERE nsp.nspname = $1
		  AND cls.relname = $2
		  AND con.contype IN ('c', 'u', 'x')
		ORDER BY con.conname
	`

	rows, err := s.pool.Query(s.ctx, query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var constraints []ConstraintInfo
	for rows.Next() {
		var c ConstraintInfo
		err := rows.Scan(&c.Name, &c.Type, &c.Definition)
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, c)
	}

	return constraints, rows.Err()
}

// GetTableComment implements the Store interface
func (s *PostgresStore) GetTableComment(schemaName, tableName string) (string, error) {
	query := `
		SELECT pg_catalog.obj_description(c.oid, 'pg_class')
		FROM pg_catalog.pg_class c
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relname = $2
	`

	var comment *string
	err := s.pool.QueryRow(s.ctx, query, schemaName, tableName).Scan(&comment)
	if err != nil {
		return "", err
	}

	if comment != nil {
		return *comment, nil
	}
	return "", nil
}

// Helper functions

func normalizePostgresType(dataType, udtName string, maxLength, precision, scale *int64) string {
	baseType := udtName

	switch baseType {
	case "varchar", "character varying":
		if maxLength != nil && *maxLength > 0 {
			return fmt.Sprintf("varchar(%d)", *maxLength)
		}
		return "varchar"
	case "bpchar":
		if maxLength != nil && *maxLength > 0 {
			return fmt.Sprintf("char(%d)", *maxLength)
		}
		return "char"
	case "numeric", "decimal":
		if precision != nil && scale != nil && *precision > 0 && *scale > 0 {
			return fmt.Sprintf("numeric(%d,%d)", *precision, *scale)
		} else if precision != nil && *precision > 0 {
			return fmt.Sprintf("numeric(%d)", *precision)
		}
		return "numeric"
	case "_text":
		return "text[]"
	case "_varchar":
		return "varchar[]"
	case "_int4":
		return "integer[]"
	default:
		return baseType
	}
}

func mapPostgreSQLTypeToGo(dbType string, isNullable bool) (goType string, importPath string) {
	baseType := extractBaseType(dbType)

	var baseGoType string
	var imp string

	switch baseType {
	case "uuid":
		baseGoType = "string"
	case "text", "varchar", "character varying", "char", "bpchar":
		baseGoType = "string"
	case "integer", "int", "int4":
		baseGoType = "int"
	case "smallint", "int2":
		baseGoType = "int16"
	case "bigint", "int8":
		baseGoType = "int64"
	case "real", "float4":
		baseGoType = "float32"
	case "double precision", "float8":
		baseGoType = "float64"
	case "numeric", "decimal":
		baseGoType = "float64"
	case "boolean", "bool":
		baseGoType = "bool"
	case "timestamp", "timestamptz", "date", "time":
		baseGoType = "time.Time"
		imp = "time"
	case "json", "jsonb":
		baseGoType = "json.RawMessage"
		imp = "encoding/json"
	case "text[]", "varchar[]":
		return "[]string", ""
	case "integer[]":
		return "[]int", ""
	case "bytea":
		return "[]byte", ""
	case "inet":
		baseGoType = "string"
	default:
		baseGoType = "interface{}"
	}

	if isNullable && !strings.HasPrefix(baseGoType, "[]") && !strings.HasPrefix(baseGoType, "*") {
		return "*" + baseGoType, imp
	}

	return baseGoType, imp
}

func extractBaseType(dbType string) string {
	if idx := strings.Index(dbType, "("); idx != -1 {
		return strings.TrimSpace(dbType[:idx])
	}
	return dbType
}

func cleanDefaultValue(defaultVal string) string {
	re := regexp.MustCompile(`::[\w\s]+(\[\])?`)
	defaultVal = re.ReplaceAllString(defaultVal, "")
	defaultVal = strings.TrimSpace(defaultVal)
	defaultVal = strings.Trim(defaultVal, "'")
	return defaultVal
}

func deriveValidationTags(col ColumnInfo) string {
	var tags []string

	if !col.IsNullable && !col.IsPrimaryKey {
		tags = append(tags, "required")
	}

	if col.DBType == "uuid" {
		tags = append(tags, "uuid")
	}

	if col.MaxLength > 0 {
		tags = append(tags, fmt.Sprintf("max=%d", col.MaxLength))
	}

	if strings.Contains(strings.ToLower(col.Name), "email") {
		tags = append(tags, "email")
	}

	if len(tags) == 0 {
		return ""
	}

	return strings.Join(tags, ",")
}
