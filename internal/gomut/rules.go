package gomut

import (
	"go/ast"
	"go/token"

	"github.com/goropikari/gomut/internal/gomut/result"
)

type binaryMutationSpec struct {
	kind        result.MutationKind
	replacement string
}

type basicLitMutationSpec struct {
	kind                 result.MutationKind
	defaultReplacement   string
	alternateReplacement string
}

type unaryMutationSpec struct {
	kind result.MutationKind
}

var binaryMutationSpecs = map[token.Token]binaryMutationSpec{
	token.EQL:     {kind: result.MutationKindComparisonOperator, replacement: "!="},
	token.NEQ:     {kind: result.MutationKindComparisonOperator, replacement: "=="},
	token.LSS:     {kind: result.MutationKindComparisonOperator, replacement: "<="},
	token.GTR:     {kind: result.MutationKindComparisonOperator, replacement: ">="},
	token.LEQ:     {kind: result.MutationKindComparisonOperator, replacement: "<"},
	token.GEQ:     {kind: result.MutationKindComparisonOperator, replacement: ">"},
	token.LAND:    {kind: result.MutationKindLogicalOperator, replacement: "||"},
	token.LOR:     {kind: result.MutationKindLogicalOperator, replacement: "&&"},
	token.ADD:     {kind: result.MutationKindArithmeticOperator, replacement: "-"},
	token.SUB:     {kind: result.MutationKindArithmeticOperator, replacement: "+"},
	token.MUL:     {kind: result.MutationKindArithmeticOperator, replacement: "/"},
	token.QUO:     {kind: result.MutationKindArithmeticOperator, replacement: "*"},
	token.REM:     {kind: result.MutationKindArithmeticOperator, replacement: "*"},
	token.AND:     {kind: result.MutationKindBitwiseOperator, replacement: "|"},
	token.OR:      {kind: result.MutationKindBitwiseOperator, replacement: "&"},
	token.XOR:     {kind: result.MutationKindBitwiseOperator, replacement: "&"},
	token.AND_NOT: {kind: result.MutationKindBitwiseOperator, replacement: "|"},
	token.SHL:     {kind: result.MutationKindShiftOperator, replacement: ">>"},
	token.SHR:     {kind: result.MutationKindShiftOperator, replacement: "<<"},
}

type assignMutationSpec struct {
	kind        result.MutationKind
	replacement string
}

var assignMutationSpecs = map[token.Token]assignMutationSpec{
	token.AND_ASSIGN:     {kind: result.MutationKindAssignmentBitwise, replacement: "|="},
	token.OR_ASSIGN:      {kind: result.MutationKindAssignmentBitwise, replacement: "&="},
	token.XOR_ASSIGN:     {kind: result.MutationKindAssignmentBitwise, replacement: "&="},
	token.AND_NOT_ASSIGN: {kind: result.MutationKindAssignmentBitwise, replacement: "|="},
}

var arithmeticAssignMutationSpecs = map[token.Token]assignMutationSpec{
	token.ADD_ASSIGN: {kind: result.MutationKindAssignmentArithmetic, replacement: "-="},
	token.SUB_ASSIGN: {kind: result.MutationKindAssignmentArithmetic, replacement: "+="},
	token.MUL_ASSIGN: {kind: result.MutationKindAssignmentArithmetic, replacement: "/="},
	token.QUO_ASSIGN: {kind: result.MutationKindAssignmentArithmetic, replacement: "*="},
	token.REM_ASSIGN: {kind: result.MutationKindAssignmentArithmetic, replacement: "*="},
}

var shiftAssignMutationSpecs = map[token.Token]assignMutationSpec{
	token.SHL_ASSIGN: {kind: result.MutationKindAssignmentShift, replacement: ">>="},
	token.SHR_ASSIGN: {kind: result.MutationKindAssignmentShift, replacement: "<<="},
}

var basicLitMutationSpecs = map[token.Token]basicLitMutationSpec{
	token.INT: {
		kind:                 result.MutationKindIntegerLiteral,
		defaultReplacement:   "0",
		alternateReplacement: "1",
	},
	token.FLOAT: {
		kind:                 result.MutationKindFloatLiteral,
		defaultReplacement:   "0.0",
		alternateReplacement: "1.0",
	},
	token.CHAR: {
		kind:                 result.MutationKindRuneLiteral,
		defaultReplacement:   "'a'",
		alternateReplacement: "'b'",
	},
	token.STRING: {
		kind:                 result.MutationKindStringLiteral,
		defaultReplacement:   `""`,
		alternateReplacement: `"mutated"`,
	},
}

var unaryMutationSpecs = map[token.Token]unaryMutationSpec{
	token.NOT: {kind: result.MutationKindUnaryNot},
	token.SUB: {kind: result.MutationKindUnaryMinus},
	token.XOR: {kind: result.MutationKindUnaryBitwiseNot},
}

func mutationCandidateFromNode(root string, fset *token.FileSet, src []byte, astFile *ast.File, file, pkg string, node ast.Node, ancestors []ast.Node, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	finders := []func() (result.Candidate, bool){
		func() (result.Candidate, bool) {
			return mutationFromBinaryNode(root, fset, src, file, pkg, node, target, coverage)
		},
		func() (result.Candidate, bool) {
			return mutationFromBooleanNode(root, fset, src, file, pkg, node, target, coverage)
		},
		func() (result.Candidate, bool) {
			return mutationFromBasicLitNode(root, fset, src, astFile, file, pkg, node, target, coverage)
		},
		func() (result.Candidate, bool) {
			return mutationFromUnaryNode(root, fset, src, file, pkg, node, target, coverage)
		},
		func() (result.Candidate, bool) {
			return mutationFromAssignNode(root, fset, src, file, pkg, node, target, coverage)
		},
		func() (result.Candidate, bool) {
			return mutationFromIncDecNode(root, fset, src, file, pkg, node, target, coverage)
		},
		func() (result.Candidate, bool) {
			return mutationFromBranchNode(root, fset, src, file, pkg, node, ancestors, target, coverage)
		},
		func() (result.Candidate, bool) {
			return mutationFromIfNode(root, fset, src, file, pkg, node, target, coverage)
		},
		func() (result.Candidate, bool) {
			return mutationFromForNode(root, fset, src, file, pkg, node, target, coverage)
		},
		func() (result.Candidate, bool) {
			return mutationFromSwitchNode(root, fset, src, file, pkg, node, target, coverage)
		},
		func() (result.Candidate, bool) {
			return mutationFromReturnNode(root, fset, src, file, pkg, node, target, coverage)
		},
	}

	for _, find := range finders {
		if candidate, ok := find(); ok {
			return candidate, true
		}
	}

	return result.Candidate{}, false
}

func mutationFromBinaryNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	binaryNode, ok := node.(*ast.BinaryExpr)
	if !ok {
		return result.Candidate{}, false
	}

	if candidate, ok := mutationFromNilCheckBinaryExpr(root, fset, src, file, pkg, binaryNode, target, coverage); ok {
		return candidate, true
	}

	return mutationFromBinaryExpr(root, fset, src, file, pkg, binaryNode, target, coverage)
}

func mutationFromBooleanNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	ident, ok := node.(*ast.Ident)
	if !ok {
		return result.Candidate{}, false
	}

	return mutationFromBooleanLiteral(root, fset, src, file, pkg, ident, target, coverage)
}

func mutationFromBasicLitNode(root string, fset *token.FileSet, src []byte, astFile *ast.File, file, pkg string, node ast.Node, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	basicLit, ok := node.(*ast.BasicLit)
	if !ok {
		return result.Candidate{}, false
	}

	if isImportPathBasicLit(astFile, basicLit) {
		return result.Candidate{}, false
	}

	return mutationFromBasicLit(root, fset, src, file, pkg, basicLit, target, coverage)
}

func isImportPathBasicLit(astFile *ast.File, node *ast.BasicLit) bool {
	if astFile == nil || node == nil {
		return false
	}

	for _, spec := range astFile.Imports {
		if spec != nil && spec.Path == node {
			return true
		}
	}

	return false
}

func mutationFromUnaryNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	unaryExpr, ok := node.(*ast.UnaryExpr)
	if !ok {
		return result.Candidate{}, false
	}

	return mutationFromUnaryExpr(root, fset, src, file, pkg, unaryExpr, target, coverage)
}

func mutationFromAssignNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	assignStmt, ok := node.(*ast.AssignStmt)
	if !ok {
		return result.Candidate{}, false
	}

	return mutationFromAssignStmt(root, fset, src, file, pkg, assignStmt, target, coverage)
}

func mutationFromIncDecNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	incDecStmt, ok := node.(*ast.IncDecStmt)
	if !ok {
		return result.Candidate{}, false
	}

	return mutationFromIncDecStmt(root, fset, src, file, pkg, incDecStmt, target, coverage)
}

func mutationFromIfNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	ifStmt, ok := node.(*ast.IfStmt)
	if !ok {
		return result.Candidate{}, false
	}

	return mutationFromConditionExpr(root, fset, src, file, pkg, ifStmt.Cond, result.MutationKindControlFlow, target, coverage)
}

func mutationFromForNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	forStmt, ok := node.(*ast.ForStmt)
	if !ok {
		return result.Candidate{}, false
	}

	return mutationFromConditionExpr(root, fset, src, file, pkg, forStmt.Cond, result.MutationKindControlFlow, target, coverage)
}

func mutationFromSwitchNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	switchStmt, ok := node.(*ast.SwitchStmt)
	if !ok {
		return result.Candidate{}, false
	}

	return mutationFromConditionExpr(root, fset, src, file, pkg, switchStmt.Tag, result.MutationKindSwitchCondition, target, coverage)
}

func mutationFromReturnNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	returnStmt, ok := node.(*ast.ReturnStmt)
	if !ok {
		return result.Candidate{}, false
	}

	return mutationFromReturnStmt(root, fset, src, file, pkg, returnStmt, target, coverage)
}

func mutationFromBooleanLiteral(root string, fset *token.FileSet, src []byte, file, pkg string, node *ast.Ident, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	if node == nil || (node.Name != "true" && node.Name != "false") {
		return result.Candidate{}, false
	}

	pos := fset.Position(node.Pos())

	line := pos.Line
	if !mutationAllowedByTarget(root, file, line, target) {
		return result.Candidate{}, false
	}

	start := pos.Offset
	end := start + len(node.Name)

	replacement := "false"
	if node.Name == "false" {
		replacement = "true"
	}

	return result.Candidate{
		File:        repoRel(root, file),
		Line:        line,
		Kind:        result.MutationKindBooleanLiteral,
		Original:    node.Name,
		Replacement: replacement,
		Start:       start,
		End:         end,
		PackagePath: pkg,
		Covered:     lineCovered(coverage, line),
	}, true
}

func mutationFromBasicLit(root string, fset *token.FileSet, src []byte, file, pkg string, node *ast.BasicLit, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	if node == nil {
		return result.Candidate{}, false
	}

	spec, ok := basicLitMutationSpecs[node.Kind]
	if !ok {
		return result.Candidate{}, false
	}

	replacement := spec.defaultReplacement
	if node.Value == spec.defaultReplacement {
		replacement = spec.alternateReplacement
	}

	pos := fset.Position(node.Pos())

	line := pos.Line
	if !mutationAllowedByTarget(root, file, line, target) {
		return result.Candidate{}, false
	}

	start := pos.Offset
	end := start + len(node.Value)

	return result.Candidate{
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

func mutationFromUnaryExpr(root string, fset *token.FileSet, src []byte, file, pkg string, node *ast.UnaryExpr, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	if node == nil {
		return result.Candidate{}, false
	}

	spec, ok := unaryMutationSpecs[node.Op]
	if !ok {
		return result.Candidate{}, false
	}

	pos := fset.Position(node.Pos())
	end := fset.Position(node.End())

	line := pos.Line
	if !mutationAllowedByTarget(root, file, line, target) {
		return result.Candidate{}, false
	}

	if pos.Offset < 0 || end.Offset > len(src) || pos.Offset >= end.Offset {
		return result.Candidate{}, false
	}

	return result.Candidate{
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

func mutationFromNilCheckBinaryExpr(root string, fset *token.FileSet, src []byte, file, pkg string, node *ast.BinaryExpr, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	if node.Op != token.EQL && node.Op != token.NEQ {
		return result.Candidate{}, false
	}

	if !isNilExpr(node.X) && !isNilExpr(node.Y) {
		return result.Candidate{}, false
	}

	pos := fset.Position(node.OpPos)

	line := pos.Line
	if !mutationAllowedByTarget(root, file, line, target) {
		return result.Candidate{}, false
	}

	start := pos.Offset
	end := start + len(node.Op.String())

	replacement := "!="
	if node.Op == token.NEQ {
		replacement = "=="
	}

	return result.Candidate{
		File:        repoRel(root, file),
		Line:        line,
		Kind:        result.MutationKindNilCheck,
		Original:    node.Op.String(),
		Replacement: replacement,
		Start:       start,
		End:         end,
		PackagePath: pkg,
		Covered:     lineCovered(coverage, line),
	}, true
}

func mutationFromBinaryExpr(root string, fset *token.FileSet, src []byte, file, pkg string, node *ast.BinaryExpr, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	spec, ok := binaryMutationSpecs[node.Op]
	if !ok {
		return result.Candidate{}, false
	}

	pos := fset.Position(node.OpPos)

	line := pos.Line
	if !mutationAllowedByTarget(root, file, line, target) {
		return result.Candidate{}, false
	}

	start := pos.Offset
	end := start + len(node.Op.String())

	return result.Candidate{
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

func mutationFromReturnStmt(root string, fset *token.FileSet, src []byte, file, pkg string, node *ast.ReturnStmt, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	if len(node.Results) != 1 {
		return result.Candidate{}, false
	}

	returnValue := node.Results[0]

	ident, ok := returnValue.(*ast.Ident)
	if !ok || ident.Name == "nil" {
		return result.Candidate{}, false
	}

	pos := fset.Position(returnValue.Pos())

	line := pos.Line
	if !mutationAllowedByTarget(root, file, line, target) {
		return result.Candidate{}, false
	}

	start := pos.Offset

	end := start + len(ident.Name)
	if start < 0 || end > len(src) {
		return result.Candidate{}, false
	}

	if ident.Name == "true" || ident.Name == "false" {
		replacement := "false"
		if ident.Name == "false" {
			replacement = "true"
		}

		return result.Candidate{
			File:        repoRel(root, file),
			Line:        line,
			Kind:        result.MutationKindReturn,
			Original:    ident.Name,
			Replacement: replacement,
			Start:       start,
			End:         end,
			PackagePath: pkg,
			Covered:     lineCovered(coverage, line),
		}, true
	}

	return result.Candidate{
		File:        repoRel(root, file),
		Line:        line,
		Kind:        result.MutationKindGuardClause,
		Original:    ident.Name,
		Replacement: "nil",
		Start:       start,
		End:         end,
		PackagePath: pkg,
		Covered:     lineCovered(coverage, line),
	}, true
}

func mutationFromAssignStmt(root string, fset *token.FileSet, src []byte, file, pkg string, node *ast.AssignStmt, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	spec, ok := assignMutationSpecs[node.Tok]
	if !ok {
		spec, ok = arithmeticAssignMutationSpecs[node.Tok]
	}

	if !ok {
		spec, ok = shiftAssignMutationSpecs[node.Tok]
	}

	if !ok {
		return result.Candidate{}, false
	}

	pos := fset.Position(node.TokPos)

	line := pos.Line
	if !mutationAllowedByTarget(root, file, line, target) {
		return result.Candidate{}, false
	}

	start := pos.Offset
	end := start + len(node.Tok.String())

	return result.Candidate{
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

func mutationFromIncDecStmt(root string, fset *token.FileSet, src []byte, file, pkg string, node *ast.IncDecStmt, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	if node == nil {
		return result.Candidate{}, false
	}

	pos := fset.Position(node.TokPos)

	line := pos.Line
	if !mutationAllowedByTarget(root, file, line, target) {
		return result.Candidate{}, false
	}

	start := pos.Offset
	end := start + len(node.Tok.String())

	replacement := "--"
	if node.Tok == token.DEC {
		replacement = "++"
	}

	return result.Candidate{
		File:        repoRel(root, file),
		Line:        line,
		Kind:        result.MutationKindIncDec,
		Original:    node.Tok.String(),
		Replacement: replacement,
		Start:       start,
		End:         end,
		PackagePath: pkg,
		Covered:     lineCovered(coverage, line),
	}, true
}

func mutationFromBranchNode(root string, fset *token.FileSet, src []byte, file, pkg string, node ast.Node, ancestors []ast.Node, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	branchStmt, ok := node.(*ast.BranchStmt)
	if !ok {
		return result.Candidate{}, false
	}

	if branchStmt.Label != nil {
		return result.Candidate{}, false
	}

	replacement, ok := loopControlReplacement(branchStmt.Tok)
	if !ok {
		return result.Candidate{}, false
	}

	if !isLoopBranch(ancestors) {
		return result.Candidate{}, false
	}

	return loopControlCandidate(root, fset, src, file, pkg, branchStmt, replacement, target, coverage)
}

func loopControlReplacement(tok token.Token) (string, bool) {
	switch tok {
	case token.BREAK:
		return "continue", true
	case token.CONTINUE:
		return "break", true
	default:
		return "", false
	}
}

func loopControlCandidate(root string, fset *token.FileSet, src []byte, file, pkg string, branchStmt *ast.BranchStmt, replacement string, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	pos := fset.Position(branchStmt.TokPos)
	line := pos.Line

	if !mutationAllowedByTarget(root, file, line, target) {
		return result.Candidate{}, false
	}

	start := pos.Offset
	end := start + len(branchStmt.Tok.String())

	if start < 0 || end > len(src) || start >= end {
		return result.Candidate{}, false
	}

	return result.Candidate{
		File:        repoRel(root, file),
		Line:        line,
		Kind:        result.MutationKindLoopControl,
		Original:    branchStmt.Tok.String(),
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

func mutationFromConditionExpr(root string, fset *token.FileSet, src []byte, file, pkg string, expr ast.Expr, kind result.MutationKind, target result.Target, coverage result.FileCoverage) (result.Candidate, bool) {
	if expr == nil {
		return result.Candidate{}, false
	}

	pos := fset.Position(expr.Pos())

	end := fset.Position(expr.End())
	if pos.Offset < 0 || end.Offset > len(src) || pos.Offset >= end.Offset {
		return result.Candidate{}, false
	}

	line := pos.Line
	if !mutationAllowedByTarget(root, file, line, target) {
		return result.Candidate{}, false
	}

	original := string(src[pos.Offset:end.Offset])
	replacement := negateCondition(expr, original)

	return result.Candidate{
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

func negateCondition(cond ast.Expr, original string) string {
	switch cond.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		return "!" + original
	default:
		return "!(" + original + ")"
	}
}

func mutationAllowedByTarget(root, file string, line int, target result.Target) bool {
	switch target.Mode {
	case result.TargetModeDiff:
		return DiffLineAllowed(repoRel(root, file), line)
	default:
		return true
	}
}

func lineCovered(coverage result.FileCoverage, line int) bool {
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

func isLoopBranch(ancestors []ast.Node) bool {
	for i := len(ancestors) - 1; i >= 0; i-- {
		switch ancestors[i].(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			return true
		case *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
			return false
		}
	}

	return false
}
