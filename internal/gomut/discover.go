package gomut

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type binaryMutationSpec struct {
	kind        MutationKind
	replacement string
}

type basicLitMutationSpec struct {
	kind                 MutationKind
	defaultReplacement   string
	alternateReplacement string
}

type unaryMutationSpec struct {
	kind MutationKind
}

var binaryMutationSpecs = map[token.Token]binaryMutationSpec{
	token.EQL:     {kind: MutationKindComparisonOperator, replacement: "!="},
	token.NEQ:     {kind: MutationKindComparisonOperator, replacement: "=="},
	token.LSS:     {kind: MutationKindComparisonOperator, replacement: "<="},
	token.GTR:     {kind: MutationKindComparisonOperator, replacement: ">="},
	token.LEQ:     {kind: MutationKindComparisonOperator, replacement: "<"},
	token.GEQ:     {kind: MutationKindComparisonOperator, replacement: ">"},
	token.LAND:    {kind: MutationKindLogicalOperator, replacement: "||"},
	token.LOR:     {kind: MutationKindLogicalOperator, replacement: "&&"},
	token.ADD:     {kind: MutationKindArithmeticOperator, replacement: "-"},
	token.SUB:     {kind: MutationKindArithmeticOperator, replacement: "+"},
	token.MUL:     {kind: MutationKindArithmeticOperator, replacement: "/"},
	token.QUO:     {kind: MutationKindArithmeticOperator, replacement: "*"},
	token.REM:     {kind: MutationKindArithmeticOperator, replacement: "*"},
	token.AND:     {kind: MutationKindBitwiseOperator, replacement: "|"},
	token.OR:      {kind: MutationKindBitwiseOperator, replacement: "&"},
	token.XOR:     {kind: MutationKindBitwiseOperator, replacement: "&"},
	token.AND_NOT: {kind: MutationKindBitwiseOperator, replacement: "|"},
	token.SHL:     {kind: MutationKindShiftOperator, replacement: ">>"},
	token.SHR:     {kind: MutationKindShiftOperator, replacement: "<<"},
}

type assignMutationSpec struct {
	kind        MutationKind
	replacement string
}

var assignMutationSpecs = map[token.Token]assignMutationSpec{
	token.AND_ASSIGN:     {kind: MutationKindAssignmentBitwise, replacement: "|="},
	token.OR_ASSIGN:      {kind: MutationKindAssignmentBitwise, replacement: "&="},
	token.XOR_ASSIGN:     {kind: MutationKindAssignmentBitwise, replacement: "&="},
	token.AND_NOT_ASSIGN: {kind: MutationKindAssignmentBitwise, replacement: "|="},
}

var arithmeticAssignMutationSpecs = map[token.Token]assignMutationSpec{
	token.ADD_ASSIGN: {kind: MutationKindAssignmentArithmetic, replacement: "-="},
	token.SUB_ASSIGN: {kind: MutationKindAssignmentArithmetic, replacement: "+="},
	token.MUL_ASSIGN: {kind: MutationKindAssignmentArithmetic, replacement: "/="},
	token.QUO_ASSIGN: {kind: MutationKindAssignmentArithmetic, replacement: "*="},
	token.REM_ASSIGN: {kind: MutationKindAssignmentArithmetic, replacement: "*="},
}

var shiftAssignMutationSpecs = map[token.Token]assignMutationSpec{
	token.SHL_ASSIGN: {kind: MutationKindAssignmentShift, replacement: ">>="},
	token.SHR_ASSIGN: {kind: MutationKindAssignmentShift, replacement: "<<="},
}

var basicLitMutationSpecs = map[token.Token]basicLitMutationSpec{
	token.INT: {
		kind:                 MutationKindIntegerLiteral,
		defaultReplacement:   "0",
		alternateReplacement: "1",
	},
	token.FLOAT: {
		kind:                 MutationKindFloatLiteral,
		defaultReplacement:   "0.0",
		alternateReplacement: "1.0",
	},
	token.CHAR: {
		kind:                 MutationKindRuneLiteral,
		defaultReplacement:   "'a'",
		alternateReplacement: "'b'",
	},
	token.STRING: {
		kind:                 MutationKindStringLiteral,
		defaultReplacement:   `""`,
		alternateReplacement: `"mutated"`,
	},
}

var unaryMutationSpecs = map[token.Token]unaryMutationSpec{
	token.NOT: {kind: MutationKindUnaryNot},
	token.SUB: {kind: MutationKindUnaryMinus},
	token.XOR: {kind: MutationKindUnaryBitwiseNot},
}

func DiscoverCandidates(root string, packages []string, target Target, coverage map[string]FileCoverage) ([]Candidate, error) {
	var candidates []Candidate

	for _, pkg := range packages {
		files, err := packageGoFiles(root, pkg)
		if err != nil {
			return nil, fmt.Errorf("list go files for %s: %w", pkg, err)
		}

		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}

			fileCandidates, err := discoverFileCandidates(root, pkg, file, target, coverage)
			if err != nil {
				return nil, fmt.Errorf("discover candidates for %s: %w", file, err)
			}

			candidates = append(candidates, fileCandidates...)
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].File != candidates[j].File {
			return candidates[i].File < candidates[j].File
		}

		if candidates[i].Line != candidates[j].Line {
			return candidates[i].Line < candidates[j].Line
		}

		return candidates[i].Kind < candidates[j].Kind
	})

	return candidates, nil
}

func discoverFileCandidates(root, pkg, file string, target Target, coverage map[string]FileCoverage) ([]Candidate, error) {
	src, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()

	astFile, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	covered := coverage[repoRel(root, file)]

	var candidates []Candidate

	ast.Inspect(astFile, func(n ast.Node) bool {
		if candidate, ok := mutationCandidateFromNode(root, fset, src, file, pkg, n, target, covered); ok {
			candidates = append(candidates, candidate)
		}

		return true
	})

	return candidates, nil
}

func mutationCandidateFromNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target Target, coverage FileCoverage) (Candidate, bool) {
	handlers := []func(string, *token.FileSet, []byte, string, string, ast.Node, Target, FileCoverage) (Candidate, bool){
		mutationFromBinaryNode,
		mutationFromBooleanNode,
		mutationFromBasicLitNode,
		mutationFromUnaryNode,
		mutationFromAssignNode,
		mutationFromIncDecNode,
		mutationFromIfNode,
		mutationFromSwitchNode,
		mutationFromReturnNode,
	}

	for _, handler := range handlers {
		if candidate, ok := handler(root, fset, src, file, pkg, node, target, coverage); ok {
			return candidate, true
		}
	}

	return Candidate{}, false
}

func mutationFromBinaryNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target Target, coverage FileCoverage) (Candidate, bool) {
	binaryNode, ok := node.(*ast.BinaryExpr)
	if !ok {
		return Candidate{}, false
	}

	if candidate, ok := mutationFromNilCheckBinaryExpr(root, fset, src, file, pkg, binaryNode, target, coverage); ok {
		return candidate, true
	}

	return mutationFromBinaryExpr(root, fset, src, file, pkg, binaryNode, target, coverage)
}

func mutationFromBooleanNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target Target, coverage FileCoverage) (Candidate, bool) {
	ident, ok := node.(*ast.Ident)
	if !ok {
		return Candidate{}, false
	}

	return mutationFromBooleanLiteral(root, fset, src, file, pkg, ident, target, coverage)
}

func mutationFromBasicLitNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target Target, coverage FileCoverage) (Candidate, bool) {
	basicLit, ok := node.(*ast.BasicLit)
	if !ok {
		return Candidate{}, false
	}

	return mutationFromBasicLit(root, fset, src, file, pkg, basicLit, target, coverage)
}

func mutationFromUnaryNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target Target, coverage FileCoverage) (Candidate, bool) {
	unaryExpr, ok := node.(*ast.UnaryExpr)
	if !ok {
		return Candidate{}, false
	}

	return mutationFromUnaryExpr(root, fset, src, file, pkg, unaryExpr, target, coverage)
}

func mutationFromAssignNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target Target, coverage FileCoverage) (Candidate, bool) {
	assignStmt, ok := node.(*ast.AssignStmt)
	if !ok {
		return Candidate{}, false
	}

	return mutationFromAssignStmt(root, fset, src, file, pkg, assignStmt, target, coverage)
}

func mutationFromIncDecNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target Target, coverage FileCoverage) (Candidate, bool) {
	incDecStmt, ok := node.(*ast.IncDecStmt)
	if !ok {
		return Candidate{}, false
	}

	return mutationFromIncDecStmt(root, fset, src, file, pkg, incDecStmt, target, coverage)
}

func mutationFromIfNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target Target, coverage FileCoverage) (Candidate, bool) {
	ifStmt, ok := node.(*ast.IfStmt)
	if !ok {
		return Candidate{}, false
	}

	return mutationFromConditionExpr(root, fset, src, file, pkg, ifStmt.Cond, MutationKindControlFlow, target, coverage)
}

func mutationFromSwitchNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target Target, coverage FileCoverage) (Candidate, bool) {
	switchStmt, ok := node.(*ast.SwitchStmt)
	if !ok {
		return Candidate{}, false
	}

	return mutationFromConditionExpr(root, fset, src, file, pkg, switchStmt.Tag, MutationKindSwitchCondition, target, coverage)
}

func mutationFromReturnNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target Target, coverage FileCoverage) (Candidate, bool) {
	returnStmt, ok := node.(*ast.ReturnStmt)
	if !ok {
		return Candidate{}, false
	}

	return mutationFromReturnStmt(root, fset, src, file, pkg, returnStmt, target, coverage)
}

func mutationFromBooleanLiteral(root string, fset *token.FileSet, src []byte, file, pkg string, node *ast.Ident, target Target, coverage FileCoverage) (Candidate, bool) {
	if node == nil || (node.Name != "true" && node.Name != "false") {
		return Candidate{}, false
	}

	pos := fset.Position(node.Pos())

	line := pos.Line
	if !mutationAllowedByTarget(file, line, target) {
		return Candidate{}, false
	}

	start := pos.Offset
	end := start + len(node.Name)

	replacement := "false"
	if node.Name == "false" {
		replacement = "true"
	}

	return Candidate{
		File:        repoRel(root, file),
		Line:        line,
		Kind:        MutationKindBooleanLiteral,
		Original:    node.Name,
		Replacement: replacement,
		Start:       start,
		End:         end,
		PackagePath: pkg,
		Covered:     lineCovered(coverage, line),
	}, true
}

func mutationFromBasicLit(root string, fset *token.FileSet, src []byte, file, pkg string, node *ast.BasicLit, target Target, coverage FileCoverage) (Candidate, bool) {
	if node == nil {
		return Candidate{}, false
	}

	spec, ok := basicLitMutationSpecs[node.Kind]
	if !ok {
		return Candidate{}, false
	}

	replacement := spec.defaultReplacement
	if node.Value == spec.defaultReplacement {
		replacement = spec.alternateReplacement
	}

	pos := fset.Position(node.Pos())

	line := pos.Line
	if !mutationAllowedByTarget(file, line, target) {
		return Candidate{}, false
	}

	start := pos.Offset
	end := start + len(node.Value)

	return Candidate{
		File:        repoRel(root, file),
		Line:        line,
		Kind:        spec.kind,
		Original:    node.Value,
		Replacement: replacement,
		Start:       start,
		End:         end,
		PackagePath: pkg,
		Covered:     lineCovered(coverage, line),
	}, true
}

func mutationFromUnaryExpr(root string, fset *token.FileSet, src []byte, file, pkg string, node *ast.UnaryExpr, target Target, coverage FileCoverage) (Candidate, bool) {
	if node == nil {
		return Candidate{}, false
	}

	spec, ok := unaryMutationSpecs[node.Op]
	if !ok {
		return Candidate{}, false
	}

	pos := fset.Position(node.Pos())
	end := fset.Position(node.End())

	line := pos.Line
	if !mutationAllowedByTarget(file, line, target) {
		return Candidate{}, false
	}

	if pos.Offset < 0 || end.Offset > len(src) || pos.Offset >= end.Offset {
		return Candidate{}, false
	}

	return Candidate{
		File:        repoRel(root, file),
		Line:        line,
		Kind:        spec.kind,
		Original:    string(src[pos.Offset:end.Offset]),
		Replacement: string(src[pos.Offset+1 : end.Offset]),
		Start:       pos.Offset,
		End:         end.Offset,
		PackagePath: pkg,
		Covered:     lineCovered(coverage, line),
	}, true
}

func mutationFromNilCheckBinaryExpr(root string, fset *token.FileSet, src []byte, file, pkg string, node *ast.BinaryExpr, target Target, coverage FileCoverage) (Candidate, bool) {
	if node.Op != token.EQL && node.Op != token.NEQ {
		return Candidate{}, false
	}

	if !isNilExpr(node.X) && !isNilExpr(node.Y) {
		return Candidate{}, false
	}

	pos := fset.Position(node.OpPos)

	line := pos.Line
	if !mutationAllowedByTarget(file, line, target) {
		return Candidate{}, false
	}

	start := pos.Offset
	end := start + len(node.Op.String())

	replacement := "!="
	if node.Op == token.NEQ {
		replacement = "=="
	}

	return Candidate{
		File:        repoRel(root, file),
		Line:        line,
		Kind:        MutationKindNilCheck,
		Original:    node.Op.String(),
		Replacement: replacement,
		Start:       start,
		End:         end,
		PackagePath: pkg,
		Covered:     lineCovered(coverage, line),
	}, true
}

func mutationFromBinaryExpr(root string, fset *token.FileSet, src []byte, file, pkg string, node *ast.BinaryExpr, target Target, coverage FileCoverage) (Candidate, bool) {
	spec, ok := binaryMutationSpecs[node.Op]
	if !ok {
		return Candidate{}, false
	}

	pos := fset.Position(node.OpPos)

	line := pos.Line
	if !mutationAllowedByTarget(file, line, target) {
		return Candidate{}, false
	}

	start := pos.Offset
	end := start + len(node.Op.String())

	return Candidate{
		File:        repoRel(root, file),
		Line:        line,
		Kind:        spec.kind,
		Original:    node.Op.String(),
		Replacement: spec.replacement,
		Start:       start,
		End:         end,
		PackagePath: pkg,
		Covered:     lineCovered(coverage, line),
	}, true
}

func mutationFromReturnStmt(root string, fset *token.FileSet, src []byte, file, pkg string, node *ast.ReturnStmt, target Target, coverage FileCoverage) (Candidate, bool) {
	if len(node.Results) != 1 {
		return Candidate{}, false
	}

	result := node.Results[0]

	ident, ok := result.(*ast.Ident)
	if !ok || ident.Name == "nil" {
		return Candidate{}, false
	}

	pos := fset.Position(result.Pos())

	line := pos.Line
	if !mutationAllowedByTarget(file, line, target) {
		return Candidate{}, false
	}

	start := pos.Offset

	end := start + len(ident.Name)
	if start < 0 || end > len(src) {
		return Candidate{}, false
	}

	if ident.Name == "true" || ident.Name == "false" {
		replacement := "false"
		if ident.Name == "false" {
			replacement = "true"
		}

		return Candidate{
			File:        repoRel(root, file),
			Line:        line,
			Kind:        MutationKindReturn,
			Original:    ident.Name,
			Replacement: replacement,
			Start:       start,
			End:         end,
			PackagePath: pkg,
			Covered:     lineCovered(coverage, line),
		}, true
	}

	return Candidate{
		File:        repoRel(root, file),
		Line:        line,
		Kind:        MutationKindGuardClause,
		Original:    ident.Name,
		Replacement: "nil",
		Start:       start,
		End:         end,
		PackagePath: pkg,
		Covered:     lineCovered(coverage, line),
	}, true
}

func mutationFromAssignStmt(root string, fset *token.FileSet, src []byte, file, pkg string, node *ast.AssignStmt, target Target, coverage FileCoverage) (Candidate, bool) {
	spec, ok := assignMutationSpecs[node.Tok]
	if !ok {
		spec, ok = arithmeticAssignMutationSpecs[node.Tok]
	}

	if !ok {
		spec, ok = shiftAssignMutationSpecs[node.Tok]
	}

	if !ok {
		return Candidate{}, false
	}

	pos := fset.Position(node.TokPos)

	line := pos.Line
	if !mutationAllowedByTarget(file, line, target) {
		return Candidate{}, false
	}

	start := pos.Offset
	end := start + len(node.Tok.String())

	return Candidate{
		File:        repoRel(root, file),
		Line:        line,
		Kind:        spec.kind,
		Original:    node.Tok.String(),
		Replacement: spec.replacement,
		Start:       start,
		End:         end,
		PackagePath: pkg,
		Covered:     lineCovered(coverage, line),
	}, true
}

func mutationFromIncDecStmt(root string, fset *token.FileSet, src []byte, file, pkg string, node *ast.IncDecStmt, target Target, coverage FileCoverage) (Candidate, bool) {
	if node == nil {
		return Candidate{}, false
	}

	pos := fset.Position(node.TokPos)

	line := pos.Line
	if !mutationAllowedByTarget(file, line, target) {
		return Candidate{}, false
	}

	start := pos.Offset
	end := start + len(node.Tok.String())

	replacement := "--"
	if node.Tok == token.DEC {
		replacement = "++"
	}

	return Candidate{
		File:        repoRel(root, file),
		Line:        line,
		Kind:        MutationKindIncDec,
		Original:    node.Tok.String(),
		Replacement: replacement,
		Start:       start,
		End:         end,
		PackagePath: pkg,
		Covered:     lineCovered(coverage, line),
	}, true
}

func isNilExpr(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "nil"
}

// mutationFromIfStmt reserves the control-flow mutation hook for if-condition inversion.
func mutationFromConditionExpr(root string, fset *token.FileSet, src []byte, file, pkg string, expr ast.Expr, kind MutationKind, target Target, coverage FileCoverage) (Candidate, bool) {
	if expr == nil {
		return Candidate{}, false
	}

	pos := fset.Position(expr.Pos())

	end := fset.Position(expr.End())
	if pos.Offset < 0 || end.Offset > len(src) || pos.Offset >= end.Offset {
		return Candidate{}, false
	}

	line := pos.Line
	if !mutationAllowedByTarget(file, line, target) {
		return Candidate{}, false
	}

	original := string(src[pos.Offset:end.Offset])
	replacement := negateCondition(expr, original)

	return Candidate{
		File:        repoRel(root, file),
		Line:        line,
		Kind:        kind,
		Original:    original,
		Replacement: replacement,
		Start:       pos.Offset,
		End:         end.Offset,
		PackagePath: pkg,
		Covered:     lineCovered(coverage, line),
	}, true
}

// negateCondition returns the textual negation used for control flow mutations.
func negateCondition(cond ast.Expr, original string) string {
	switch cond.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		return "!" + original
	default:
		return "!(" + original + ")"
	}
}

func mutationAllowedByTarget(file string, line int, target Target) bool {
	switch target.Mode {
	case TargetModeDiff:
		return DiffLineAllowed(file, line)
	default:
		return true
	}
}

func lineCovered(coverage FileCoverage, line int) bool {
	if len(coverage.Ranges) == 0 {
		return true
	}

	for _, block := range coverage.Ranges {
		if line >= block.StartLine && line <= block.EndLine && block.Covered {
			return true
		}
	}

	return false
}

func ApplyMutation(root string, candidate Candidate) ([]byte, error) {
	path := resolveSourcePath(root, candidate.File)

	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if candidate.Start < 0 || candidate.End > len(src) || candidate.Start > candidate.End {
		return nil, fmt.Errorf("invalid mutation range for %s:%d", candidate.File, candidate.Line)
	}

	out := make([]byte, 0, len(src)-(candidate.End-candidate.Start)+len(candidate.Replacement))
	out = append(out, src[:candidate.Start]...)
	out = append(out, candidate.Replacement...)
	out = append(out, src[candidate.End:]...)

	return out, nil
}

func resolveSourcePath(root, file string) string {
	if filepath.IsAbs(file) {
		return file
	}

	return filepath.Join(root, file)
}
