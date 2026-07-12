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

func DiscoverCandidates(root string, packages []string, target Target, coverage map[string]FileCoverage) ([]Candidate, error) {
	var candidates []Candidate
	for _, pkg := range packages {
		files, err := packageGoFiles(root, pkg)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			fileCandidates, err := discoverFileCandidates(root, pkg, file, target, coverage)
			if err != nil {
				return nil, err
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

	covered := coverage[repoRel(file)]
	var candidates []Candidate
	ast.Inspect(astFile, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.BinaryExpr:
			if candidate, ok := mutationFromBinaryExpr(fset, src, file, pkg, node, target, covered); ok {
				candidates = append(candidates, candidate)
			}
		case *ast.ReturnStmt:
			if candidate, ok := mutationFromReturnStmt(fset, src, file, pkg, node, target, covered); ok {
				candidates = append(candidates, candidate)
			}
		}
		return true
	})
	return candidates, nil
}

func mutationFromBinaryExpr(fset *token.FileSet, src []byte, file, pkg string, node *ast.BinaryExpr, target Target, coverage FileCoverage) (Candidate, bool) {
	var kind MutationKind
	var replacement string
	switch node.Op {
	case token.EQL:
		kind, replacement = MutationKindComparisonOperator, "!="
	case token.NEQ:
		kind, replacement = MutationKindComparisonOperator, "=="
	case token.LSS:
		kind, replacement = MutationKindComparisonOperator, "<="
	case token.GTR:
		kind, replacement = MutationKindComparisonOperator, ">="
	case token.LEQ:
		kind, replacement = MutationKindComparisonOperator, "<"
	case token.GEQ:
		kind, replacement = MutationKindComparisonOperator, ">"
	case token.LAND:
		kind, replacement = MutationKindLogicalOperator, "||"
	case token.LOR:
		kind, replacement = MutationKindLogicalOperator, "&&"
	case token.ADD:
		kind, replacement = MutationKindArithmeticOperator, "-"
	case token.SUB:
		kind, replacement = MutationKindArithmeticOperator, "+"
	case token.MUL:
		kind, replacement = MutationKindArithmeticOperator, "/"
	case token.QUO:
		kind, replacement = MutationKindArithmeticOperator, "*"
	case token.REM:
		kind, replacement = MutationKindArithmeticOperator, "*"
	default:
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
		File:        repoRel(file),
		Line:        line,
		Kind:        kind,
		Original:    node.Op.String(),
		Replacement: replacement,
		Start:       start,
		End:         end,
		PackagePath: pkg,
		Covered:     lineCovered(coverage, line),
	}, true
}

func mutationFromReturnStmt(fset *token.FileSet, src []byte, file, pkg string, node *ast.ReturnStmt, target Target, coverage FileCoverage) (Candidate, bool) {
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
	return Candidate{
		File:        repoRel(file),
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
