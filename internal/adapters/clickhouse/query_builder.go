package clickhouse

import "time"

// OptionalFilter represents an optional WHERE clause filter.
// When Value is non-empty the filter is appended as "AND column = ?".
type OptionalFilter struct {
	Column string
	Value  string
}

// BuildQuery constructs a parameterized SELECT query with mandatory and optional filters.
// All readers share this pattern: base SELECT + mandatory WHERE + optional string filters +
// time range + ORDER BY DESC + LIMIT.
//
// The generated SQL matches the exact format previously produced by each Build*Query function:
//
//	SELECT <selectClause>
//	FROM <table>
//	WHERE <mandatoryWhere>[ AND <filter.Column> = ?][ AND <timeColumn> >= ?][ AND <timeColumn> <= ?] ORDER BY <orderColumn> DESC LIMIT ?
func BuildQuery(selectClause, table string, mandatoryWhere string, mandatoryArgs []any, filters []OptionalFilter, timeColumn string, since, until int64, orderColumn string, limit int) (string, []any) {
	q := "SELECT " + selectClause + "\nFROM " + table + "\nWHERE " + mandatoryWhere
	args := append([]any{}, mandatoryArgs...)

	for _, f := range filters {
		if f.Value != "" {
			q += " AND " + f.Column + " = ?"
			args = append(args, f.Value)
		}
	}

	if since > 0 {
		q += " AND " + timeColumn + " >= ?"
		args = append(args, time.Unix(since, 0))
	}
	if until > 0 {
		q += " AND " + timeColumn + " <= ?"
		args = append(args, time.Unix(until, 0))
	}

	q += " ORDER BY " + orderColumn + " DESC LIMIT ?"
	args = append(args, limit)

	return q, args
}
