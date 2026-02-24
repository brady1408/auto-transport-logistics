package store

import (
	"fmt"
	"strings"
)

// queryBuilder helps build parameterized WHERE clauses with auto-incrementing $N placeholders.
type queryBuilder struct {
	conds []string
	args  []any
	argN  int
}

func newQueryBuilder() *queryBuilder {
	return &queryBuilder{argN: 1}
}

// Add adds a condition with arguments. Each ? is replaced with $N.
func (qb *queryBuilder) Add(cond string, args ...any) {
	for _, a := range args {
		cond = strings.Replace(cond, "?", fmt.Sprintf("$%d", qb.argN), 1)
		qb.args = append(qb.args, a)
		qb.argN++
	}
	qb.conds = append(qb.conds, cond)
}

// AddRaw adds a condition without any arguments.
func (qb *queryBuilder) AddRaw(cond string) {
	qb.conds = append(qb.conds, cond)
}

// Where returns "WHERE cond1 AND cond2 ..." or empty string if no conditions.
func (qb *queryBuilder) Where() string {
	if len(qb.conds) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(qb.conds, " AND ")
}

// Paginate appends LIMIT/OFFSET args and returns the SQL fragment.
func (qb *queryBuilder) Paginate(pageSize, page int) string {
	offset := (page - 1) * pageSize
	s := fmt.Sprintf("LIMIT $%d OFFSET $%d", qb.argN, qb.argN+1)
	qb.args = append(qb.args, pageSize, offset)
	qb.argN += 2
	return s
}

// Args returns all accumulated arguments.
func (qb *queryBuilder) Args() []any {
	return qb.args
}
