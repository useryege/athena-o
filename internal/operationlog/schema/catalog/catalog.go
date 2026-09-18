// Package catalog reads a deterministic PostgreSQL schema description without
// importing the embedded contract or changing database objects.
package catalog

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type entry struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Definition string `json:"definition"`
}

// Read returns canonical JSON for all application objects in the operation_log schema.
// OIDs, ownership, statistics, sequence values and table rows are excluded.
func Read(ctx context.Context, tx pgx.Tx) ([]byte, error) {
	// Deparser output must not depend on the connection's search_path.
	if _, err := tx.Exec(ctx, `SET LOCAL search_path = pg_catalog, operation_log`); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `
 WITH relations AS (
 SELECT c.* FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
 WHERE n.nspname='operation_log' AND c.relkind IN ('r','p','v','m','S')
 ), entries AS (
 SELECT 'table' AS kind, c.relname::text AS name,
 concat(c.relkind, ' persistence=',c.relpersistence,' rls=',c.relrowsecurity,' force_rls=',c.relforcerowsecurity) AS definition
 FROM relations c WHERE c.relkind IN ('r','p')
 UNION ALL
 SELECT 'column', c.relname||'.'||a.attname,
 concat(format_type(a.atttypid,a.atttypmod), CASE WHEN a.attnotnull THEN ' NOT NULL' ELSE ' NULL' END,
 ' DEFAULT ',coalesce(pg_get_expr(d.adbin,d.adrelid),''),' identity=',a.attidentity,' generated=',a.attgenerated,
 ' collation=',coalesce(coll.collname,''))
 FROM relations c JOIN pg_attribute a ON a.attrelid=c.oid AND a.attnum>0 AND NOT a.attisdropped
 LEFT JOIN pg_attrdef d ON d.adrelid=c.oid AND d.adnum=a.attnum
 LEFT JOIN pg_collation coll ON coll.oid=a.attcollation
 WHERE c.relkind IN ('r','p','v','m')
 UNION ALL
 SELECT 'constraint', c.relname||'.'||con.conname,
 concat(pg_get_constraintdef(con.oid,false),' validated=',con.convalidated)
 FROM relations c JOIN pg_constraint con ON con.conrelid=c.oid
 UNION ALL
 SELECT 'index', idx.relname::text,
 concat(pg_get_indexdef(i.indexrelid,0,false),' valid=',i.indisvalid,' ready=',i.indisready)
 FROM relations c JOIN pg_index i ON i.indrelid=c.oid JOIN pg_class idx ON idx.oid=i.indexrelid
 UNION ALL
 SELECT 'trigger', c.relname||'.'||t.tgname,
 concat(pg_get_triggerdef(t.oid,false),' enabled=',t.tgenabled)
 FROM relations c JOIN pg_trigger t ON t.tgrelid=c.oid WHERE NOT t.tgisinternal
 UNION ALL
 SELECT 'function', p.proname||'('||pg_get_function_identity_arguments(p.oid)||')',
 pg_get_functiondef(p.oid) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='operation_log' AND p.prokind IN ('f','p')
 UNION ALL
 SELECT 'sequence', c.relname::text,
 concat(format_type(s.seqtypid,NULL),' start=',s.seqstart,' increment=',s.seqincrement,' min=',s.seqmin,
 ' max=',s.seqmax,' cache=',s.seqcache,' cycle=',s.seqcycle)
 FROM relations c JOIN pg_sequence s ON s.seqrelid=c.oid
 UNION ALL
 SELECT 'view', c.relname::text, pg_get_viewdef(c.oid,false) FROM relations c WHERE c.relkind IN ('v','m')
 ) SELECT kind,name,definition FROM entries ORDER BY kind COLLATE "C",name COLLATE "C",definition COLLATE "C"`)
	if err != nil {
		return nil, fmt.Errorf("read postgres schema catalog: %w", err)
	}
	defer rows.Close()
	entries := []entry{}
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.Kind, &e.Name, &e.Definition); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
