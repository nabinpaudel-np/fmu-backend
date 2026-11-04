package nlsearch

import (
	"fmt"
	"strconv"
	"strings"
)

// The compiler is the only place where a model's answer becomes SQL. It
// receives a Plan that has already been checked against the vocabulary, so
// every identifier it writes comes from vocabulary.go and every value becomes
// a bound parameter. Nothing the model said is ever concatenated into the
// query text.
//
// The model cannot widen its own access: the publication filter is added here,
// after the model is finished, and the model's filters are AND-ed on top of
// it.

const (
	// trigramThreshold is the similarity floor for a fuzzy text match. It
	// sits below Postgres' default of 0.3 so near-misses still match, which
	// is what makes "kathamndu" find "Kathmandu".
	trigramThreshold = 0.2
)

// Compiled is a plan rendered as SQL fragments. The caller assembles them
// into a statement; it cannot add a predicate of its own.
type Compiled struct {
	// Where is a set of AND-ed predicates, each prefixed with " AND ".
	Where string
	// Args holds the bound parameters in placeholder order.
	Args []any
	// OrderBy is a fully-formed ORDER BY expression.
	OrderBy string
}

// builder accumulates bound arguments and hands out placeholder numbers.
type builder struct {
	args []any
	// relevance records the first fuzzy text filter on the entity being
	// returned, so results can be ordered by how well they match.
	relevance *relevance
}

// relevance is the column and parameter a fuzzy match is scored against.
// The two travel together so they cannot be mixed up.
type relevance struct {
	column string
	param  string
}

// bind records a value and returns its placeholder.
func (b *builder) bind(v any) string {
	b.args = append(b.args, v)
	return "$" + strconv.Itoa(len(b.args))
}

// Compile renders a validated plan into a WHERE clause, its arguments and an
// ordering. The plan is expected to have come from Validate; a plan that has
// not is a programming error, not a user error.
func Compile(plan *Plan) *Compiled {
	b := &builder{}
	entity := plan.Entity

	clauses := []string{statusClause(entity)}

	for _, f := range plan.Filters {
		clauses = append(clauses, b.filterClause(entity, f))
	}

	for _, rel := range plan.Related {
		// A fuzzy filter inside a related clause ranks rows in that
		// subquery, not the rows being returned, so the outer ranking is
		// left as it was.
		saved := b.relevance
		clauses = append(clauses, b.relatedClause(entity, rel))
		b.relevance = saved
	}

	return &Compiled{
		Where:   " AND " + strings.Join(clauses, " AND "),
		Args:    b.args,
		OrderBy: orderBy(plan, b),
	}
}

// statusClause restricts a result set to published rows. It is added by the
// server, so a model that asks for a draft never sees one.
func statusClause(entity *Entity) string {
	return fmt.Sprintf("%s.%s = 'published'", entity.Alias, entity.StatusColumn)
}

// filterClause renders one validated filter.
func (b *builder) filterClause(entity *Entity, f FilterSpec) string {
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
		return fmt.Sprintf("%s = ANY(%s::text[])", lower(column), b.bind(lowerAll(stringList(f))))

	case OpEq:
		switch field.Kind {
		case KindBool:
			return fmt.Sprintf("%s = %s", column, b.bind(f.Value.Bool))
		case KindNumber:
			return fmt.Sprintf("%s = %s", column, b.bind(f.Value.Num))
		default:
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

// lookupClause renders a facet filter as an EXISTS over the junction table.
// The reference row's name is matched, so the value the model picked is
// checked against the seeded list at query time as well as at validation
// time.
func (b *builder) lookupClause(entity *Entity, field Field, f FilterSpec) string {
	lookup := field.Lookup
	values := lowerAll(stringList(f))
	return fmt.Sprintf(
		"EXISTS (SELECT 1 FROM %s jt JOIN %s lk ON lk.id = jt.%s WHERE jt.%s = %s.id AND lower(lk.name) = ANY(%s::text[]))",
		lookup.JoinTable, lookup.LookupTable, lookup.FKColumn, lookup.ParentColumn,
		entity.Alias, b.bind(values),
	)
}

// relatedClause renders a filter on a related entity as an EXISTS. The
// related rows are scoped to published as well, so a draft scholarship
// cannot make a university look like it offers one.
func (b *builder) relatedClause(entity *Entity, rel RelatedSpec) string {
	relation := rel.Relation
	inner := []string{
		fmt.Sprintf("%s.%s = %s.id", relation.Entity.Alias, relation.FK, entity.Alias),
		statusClause(relation.Entity),
	}
	for _, f := range rel.Filters {
		inner = append(inner, b.filterClause(relation.Entity, f))
	}
	return fmt.Sprintf(
		"EXISTS (SELECT 1 FROM %s %s WHERE %s)",
		relation.Entity.Table, relation.Entity.Alias, strings.Join(inner, " AND "),
	)
}

// orderBy renders the requested ordering. A question that names something in
// passing ("universities like Kathmandu University") is answered best by the
// closest match first, so a fuzzy filter outranks the default order unless
// the model asked for a specific one.
func orderBy(plan *Plan, b *builder) string {
	if plan.Sort != nil {
		if field, ok := plan.Entity.Field(plan.Sort.Field); ok && field.Column != "" {
			return fmt.Sprintf("%s %s", field.Column, strings.ToUpper(plan.Sort.Direction))
		}
	}
	if b.relevance != nil {
		return fmt.Sprintf("similarity(%s, %s) DESC, %s", b.relevance.column, b.relevance.param, plan.Entity.DefaultOrder)
	}
	return plan.Entity.DefaultOrder
}

// escapeLike neutralises the wildcards in a user-supplied value so it is
// matched literally. Without this, a value of "%" matches every row and a
// value of "_" matches any single character.
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

// lower wraps a column so a comparison ignores case. The model writes values
// the way a person writes them ("nepal"), the database stores them the way
// the seed did ("Nepal").
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
