package musics

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type (
	// Query stores and builds SQL query.
	// Retrieve Record that satisfies query conditions using Exec.
	Query struct {
		query string
		args  []interface{}
		table table
		err   error
	}

	// Filter expression builder for SQL WHERE statement.
	Filter struct {
		expression string
		args       []interface{}
	}

	// Field interface for table field type.
	Field interface {
		Name() string
	}

	// Queryable interface for Query that can only be executed.
	Queryable interface {
		Execute() ([]Record, error)
	}

	// Limitable interface for Query that can only build SQL LIMIT or be executed.
	Limitable interface {
		Queryable
		Limit(limit, offset int) Queryable
	}

	// Orderable interface for Query that can only build SQL ORDER BY or SQL LIMIT or be executed.
	Orderable interface {
		Limitable
		OrderBy(order OrderField, orders ...OrderField) Limitable
	}

	// Order type for SQL ORDER BY.
	Order int

	// OrderField contains Field and Order pair to build SQL ORDER BY.
	OrderField struct {
		Field Field
		Order Order
	}
)

const (
	// Ascending denotes ascending order (ASC).
	Ascending Order = iota
	// Descending denotes Descending order (DESC).
	Descending
)

// Select all records from current table.
// Make selection more precise by chaining more SQL clause builder on resulting Query.
func (t table) Select() (q Query) {
	return Query{
		query: "" +
			"SELECT\n" +
			"	id, artist, title, album, release_date, last_played, rating, description\n" +
			"FROM musics",
		table: t,
		err:   nil,
	}
}

// Where builds SQL WHERE clause to filter selection using supplied Filter.
func (q Query) Where(exp Filter) Orderable {
	q.query += "\nWHERE\n	" + exp.expression
	q.args = append(q.args, exp.args...)
	return q
}

// And chains this Filter with given expression using SQL AND clause.
func (exp Filter) And(expression Filter) Filter {
	return Filter{
		expression: exp.expression + " AND " + expression.expression,
		args:       append(exp.args, expression.args...),
	}
}

// Or chains this Filter with given expression using SQL OR clause.
func (exp Filter) Or(expression Filter) Filter {
	return Filter{
		expression: exp.expression + " OR " + expression.expression,
		args:       append(exp.args, expression.args...),
	}
}

// OrderBy builds SQL ORDER BY clause to sort selection.
func (q Query) OrderBy(order OrderField, orders ...OrderField) Limitable {
	q.query += "\nORDER BY\n	" + order.Field.Name() + " " + orderCode(order.Order)
	for _, v := range orders {
		q.query += ", " + v.Field.Name() + " " + orderCode(order.Order)
	}
	return q
}

// Limit builds SQL LIMIT clause to limit selecition.
func (q Query) Limit(limit, offset int) Queryable {
	q.query += "\nLIMIT ? OFFSET ?"
	q.args = append(q.args, limit, offset)
	return q
}

// Execute the query that has been build, returning records that match selection clause.
func (q Query) Execute() (r []Record, err error) {
	rows, err := q.table.dbSlave.Queryx(q.BuildSQL(), q.args...)
	if err != nil {
		return nil, errors.Wrap(err, "query: exec failed")
	}
	defer rows.Close()

	for rows.Next() {
		d := Data{}
		if err := rows.StructScan(&d); err != nil {
			return nil, errors.Wrap(err, "query: struct scan failed")
		}
		r = append(r, record{data: d})
	}
	return r, nil
}

// BuildSQL that will be queried.
func (q Query) BuildSQL() string {
	for i := range q.args {
		q.query = strings.Replace(q.query, "?", fmt.Sprintf("$%d", i+1), 1)
	}
	return q.query
}

func orderCode(o Order) string {
	if o == Ascending {
		return "ASC"
	}
	return "DESC"
}
