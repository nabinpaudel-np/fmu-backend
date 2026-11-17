package nlsearch

import (
	"fmt"
	"strconv"
	"strings"
)

// The compiler turns a validated plan into SQL. Identifiers come from
// vocabulary.go and values are always bound as parameters, so nothing the
// model said ends up in the query text.

const (
	// trigramThreshold is the similarity floor for a fuzzy text match. It
	// sits below Postgres' default of 0.3 so "kathamndu" still matches
	// "Kathmandu".
	trigramThreshold = 0.2
)

// Compiled is a plan rendered as SQL fragments.
type Compiled struct {
	Where   string
	Args    []any
	OrderBy string
}

// builder accumulates bound arguments and hands out placeholder numbers.
type builder struct {
	args []any
	// relevance is the first fuzzy text filter on the returned entity, used
	// to order results by how well they match.
	relevance *relevance
	// rank is the full-text search on the returned entity, used to order
	// results by how well their prose matches.
	rank *rank
}

// relevance is the column and parameter a fuzzy match is scored against.
type relevance struct {
	column string
	param  string
}

// rank is the vector column and query a full-text search is scored against.
type rank struct {
	column string
	param  string
}

// bind records a value and returns its placeholder.
func (b *builder) bind(v any) string {
	b.args = append(b.args, v)
	return "$" + strconv.Itoa(len(b.args))
}

// Compile renders a validated plan into a WHERE clause, its arguments and an
// ordering.
func Compile(plan *Plan) *Compiled {
	b := &builder{}
	entity := plan.Entity

	clauses := []string{}
	if status := statusClause(entity); status != "" {
		clauses = append(clauses, status)
	}

	if plan.Search != "" {
		clauses = append(clauses, b.searchClause(entity, plan.Search))
	}

	for _, f := range plan.Filters {
		clauses = append(clauses, b.filterClause(entity, f))
	}

	for _, rel := range plan.Related {
		// A fuzzy filter inside a related clause ranks rows in that
		// subquery, not the rows being returned.
		saved := b.relevance
		clauses = append(clauses, b.relatedClause(entity, rel))
		b.relevance = saved
	}

	where := ""
	if len(clauses) > 0 {
		where = " AND " + strings.Join(clauses, " AND ")
	}

	return &Compiled{
		Where:   where,
		Args:    b.args,
		OrderBy: orderBy(plan, b),
	}
}

// statusClause restricts a result set to published rows. It is added here,
func statusClause(entity *Entity) string {
	if entity.StatusColumn == "" {
		return ""
	}
	return fmt.Sprintf("%s.%s = 'published'", entity.Alias, entity.StatusColumn)
}

// searchClause matches the entity's prose against a free-text term. The
// vector column holds the indexed text, so the match uses the index.
//
// The trigram match is kept as a fallback: a term the dictionary does not
// know, or a word typed in the middle of a phrase, still finds its row.
func (b *builder) searchClause(entity *Entity, search string) string {
	if len(entity.Content) == 0 {
		return "FALSE"
	}

	vector := entity.Alias + ".search_vector"
	query := b.bind(search)
	like := b.bind("%" + escapeLike(search) + "%")

	b.rank = &rank{column: vector, param: query}

	parts := []string{}
	for _, c := range entity.Content {
		parts = append(parts, fmt.Sprintf(
			"(%s ILIKE %s ESCAPE '\\')",
			entity.Alias+"."+c.Column, like,
		))
	}

	return fmt.Sprintf(
		"(%s @@ websearch_to_tsquery('english', %s) OR %s)",
		vector, query, strings.Join(parts, " OR "),
	)
}

// filterClause renders one validated filter. A group of alternatives becomes
// a bracketed OR of its members.
func (b *builder) filterClause(entity *Entity, f FilterSpec) string {
	if len(f.Any) > 0 {
		parts := make([]string, 0, len(f.Any))
		for _, alt := range f.Any {
			parts = append(parts, b.filterClause(entity, alt))
		}
		return "(" + strings.Join(parts, " OR ") + ")"
	}

	field, ok := entity.Field(f.Field)
	if !ok {
		// Unreachable: Validate rejected unknown fields.
		return "FALSE"
	}
	if field.Lookup != nil {
		return b.lookupClause(entity, field, f)
	}

	column := field.Column
	switch f.Op {
	case OpIn:
		if len(field.CommonValues) > 0 {
			return b.looseClause(column, f)
		}
		return fmt.Sprintf("%s = ANY(%s::text[])", lower(column), b.bind(lowerAll(stringList(f))))

	case OpEq:
		switch field.Kind {
		case KindBool:
			return fmt.Sprintf("%s = %s", column, b.bind(f.Value.Bool))
		case KindNumber:
			return fmt.Sprintf("%s = %s", column, b.bind(f.Value.Num))
		default:
			if len(field.CommonValues) > 0 {
				return b.looseClause(column, f)
			}
			return fmt.Sprintf("%s = %s", lower(column), b.bind(strings.ToLower(f.Value.Str)))
		}

	case OpNe:
		switch field.Kind {
		case KindBool:
			return fmt.Sprintf("%s IS DISTINCT FROM %s", column, b.bind(f.Value.Bool))
		case KindNumber:
			return fmt.Sprintf("%s IS DISTINCT FROM %s", column, b.bind(f.Value.Num))
		default:
			return fmt.Sprintf("%s IS DISTINCT FROM %s", lower(column), b.bind(strings.ToLower(f.Value.Str)))
		}

	case OpGt, OpGte, OpLt, OpLte:
		return fmt.Sprintf("%s %s %s", column, f.Op, b.bind(f.Value.Num))

	case OpContains:
		return fmt.Sprintf("%s ILIKE '%%' || %s || '%%' ESCAPE '\\'", column, b.bind(escapeLike(f.Value.Str)))

	case OpMatches:
		param := b.bind(f.Value.Str)
		likeParam := b.bind(escapeLike(f.Value.Str))
		if b.relevance == nil {
			b.relevance = &relevance{column: column, param: param}
		}
		return fmt.Sprintf(
			"(similarity(%s, %s) > %s OR %s ILIKE '%%' || %s || '%%' ESCAPE '\\')",
			column, param, strconv.FormatFloat(trigramThreshold, 'f', -1, 64), column, likeParam,
		)
	}

	return "FALSE"
}

// looseClause matches a value as a fragment rather than exactly.
func (b *builder) looseClause(column string, f FilterSpec) string {
	parts := make([]string, 0, len(f.Value.List)+1)
	for _, value := range f.Value.candidates() {
		parts = append(parts, fmt.Sprintf(
			"%s ILIKE %s ESCAPE '\\'", column, b.bind("%"+escapeLike(value)+"%")))
	}
	if len(parts) == 0 {
		return "FALSE"
	}
	return "(" + strings.Join(parts, " OR ") + ")"
}

// lookupClause renders a facet filter as an EXISTS over the junction table.
func (b *builder) lookupClause(entity *Entity, field Field, f FilterSpec) string {
	lookup := field.Lookup
	values := lowerAll(stringList(f))

	if f.Op == OpNe {
		return fmt.Sprintf(
			"NOT EXISTS (SELECT 1 FROM %s jt JOIN %s lk ON lk.id = jt.%s WHERE jt.%s = %s.id AND lower(lk.name) = ANY(%s::text[]))",
			lookup.JoinTable, lookup.LookupTable, lookup.FKColumn, lookup.ParentColumn,
			entity.Alias, b.bind(values),
		)
	}

	return fmt.Sprintf(
		"EXISTS (SELECT 1 FROM %s jt JOIN %s lk ON lk.id = jt.%s WHERE jt.%s = %s.id AND lower(lk.name) = ANY(%s::text[]))",
		lookup.JoinTable, lookup.LookupTable, lookup.FKColumn, lookup.ParentColumn,
		entity.Alias, b.bind(values),
	)
}

// relatedClause renders a filter on a related entity as an EXISTS.
func (b *builder) relatedClause(entity *Entity, rel RelatedSpec) string {
	relation := rel.Relation

	var inner []string
	if relation.Through != nil {
		inner = append(inner, b.throughClause(entity, relation, rel.Filters)...)
	} else {
		inner = append(inner,
			fmt.Sprintf("%s.%s = %s.id", relation.Entity.Alias, relation.FK, entity.Alias),
			statusClause(relation.Entity),
		)
		for _, f := range rel.Filters {
			inner = append(inner, b.filterClause(relation.Entity, f))
		}
	}

	return fmt.Sprintf(
		"EXISTS (SELECT 1 FROM %s %s WHERE %s)",
		relation.Entity.Table, relation.Entity.Alias, strings.Join(inner, " AND "),
	)
}

// throughClause renders a relation that runs through a shared middle table.
func (b *builder) throughClause(entity *Entity, relation *Relation, filters []FilterSpec) []string {
	through := relation.Through
	related := relation.Entity

	// The middle table gets its own alias so its columns do not collide
	// with either side's.
	const middle = "mid"

	clauses := []string{}

	clauses = append(clauses, fmt.Sprintf(
		"EXISTS (SELECT 1 FROM %s %s WHERE %s)",
		through.Table, middle, b.throughSides(entity, related, relation, middle),
	))

	// A draft row on the other side must not satisfy the constraint.
	if status := statusClause(related); status != "" {
		clauses = append(clauses, status)
	}

	for _, f := range filters {
		clauses = append(clauses, b.filterClause(related, f))
	}

	return clauses
}

// throughSides renders the two halves of a shared-middle relation.
func (b *builder) throughSides(entity, related *Entity, relation *Relation, middle string) string {
	through := relation.Through

	var sides []string

	switch {
	case through.ParentJunction != "":
		sides = append(sides, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM %s pj WHERE pj.%s = %s.id AND pj.%s = %s.id)",
			through.ParentJunction,
			through.ParentColumn, entity.Alias,
			through.MiddleColumn, middle,
		))
	default:
		// The parent reaches the middle table directly, through its own
		// column: a programme points at its degree level with degree_id.
		sides = append(sides, fmt.Sprintf(
			"%s.%s = %s.%s",
			entity.Alias, through.ParentColumn, middle, through.MiddleColumn,
		))
	}

	switch {
	case through.EntityJunction != "":
		sides = append(sides, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM %s ej WHERE ej.%s = %s.id AND ej.%s = %s.id)",
			through.EntityJunction,
			through.EntityColumn, related.Alias,
			through.MiddleEntityColumn, middle,
		))
	default:
		sides = append(sides, fmt.Sprintf(
			"%s.%s = %s.%s",
			related.Alias, through.EntityColumn, middle, through.MiddleEntityColumn,
		))
	}

	return strings.Join(sides, " AND ")
}

// orderBy renders the requested ordering.
func orderBy(plan *Plan, b *builder) string {
	if plan.Sort != nil {
		if field, ok := plan.Entity.Field(plan.Sort.Field); ok && field.Column != "" {
			return fmt.Sprintf("%s %s", field.Column, strings.ToUpper(plan.Sort.Direction))
		}
	}
	if b.rank != nil {
		return fmt.Sprintf("ts_rank(%s, websearch_to_tsquery('english', %s)) DESC, %s",
			b.rank.column, b.rank.param, plan.Entity.DefaultOrder)
	}
	if b.relevance != nil {
		return fmt.Sprintf("similarity(%s, %s) DESC, %s", b.relevance.column, b.relevance.param, plan.Entity.DefaultOrder)
	}
	return plan.Entity.DefaultOrder
}

// escapeLike neutralises the wildcards in a value so it is matched
// literally. Without this, "%" matches every row.
func escapeLike(s string) string {
	return likeEscaper.Replace(s)
}

// likeEscaper escapes the backslash first, so an escape written by the
// escaper is not escaped again.
var likeEscaper = strings.NewReplacer(
	`\`, `\\`,
	`%`, `\%`,
	`_`, `\_`,
)

// lower wraps a column so a comparison ignores case.
func lower(column string) string {
	return "lower(" + column + ")"
}

// stringList flattens a filter value into a list of strings.
func stringList(f FilterSpec) []string {
	if f.Value.Kind == ValueList {
		return f.Value.List
	}
	return []string{f.Value.Str}
}

func lowerAll(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, strings.ToLower(v))
	}
	return out
}
