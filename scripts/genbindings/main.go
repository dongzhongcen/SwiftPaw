// genbindings 根据 app.go 里 *App 的公开方法，生成 frontend/wailsjs 下的 App.js、App.d.ts 和 models.ts。
//
// 为什么需要它：正常情况下 `wails dev` / `wails build` 会自动生成这些文件，
// 但有些环境（比如没有 webkit 开发库的 Linux）装不了 wails 命令行工具。
// 这个脚本只用 Go 标准库解析源码，输出格式和 wails 生成的一致。
//
// 用法（在项目根目录）：go run ./scripts/genbindings
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// modulePath 是 go.mod 里的模块名，以它开头的 import 才是项目自己的包
const modulePath = "musicplayer"

// pkgInfo 是解析好的一个包：包名、类型声明、import 别名
type pkgInfo struct {
	name    string
	types   map[string]ast.Expr          // 类型名 -> 类型定义
	imports map[string]map[string]string // 文件名 -> (别名 -> import 路径)
	files   map[string]*ast.File
}

var (
	fset     = token.NewFileSet()
	packages = map[string]*pkgInfo{} // import 路径（根目录是 "."）-> 包
)

// loadPackage 解析一个目录里的所有非测试 Go 文件
func loadPackage(importPath string) *pkgInfo {
	if p, ok := packages[importPath]; ok {
		return p
	}
	dir := "."
	if importPath != "." {
		dir = strings.TrimPrefix(importPath, modulePath+"/")
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "*.go"))
	p := &pkgInfo{types: map[string]ast.Expr{}, imports: map[string]map[string]string{}, files: map[string]*ast.File{}}
	for _, file := range matches {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			log.Fatal(err)
		}
		p.name = f.Name.Name
		p.files[file] = f
		imps := map[string]string{}
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			alias := filepath.Base(path)
			if imp.Name != nil {
				alias = imp.Name.Name
			}
			imps[alias] = path
		}
		p.imports[file] = imps
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts := spec.(*ast.TypeSpec)
				p.types[ts.Name.Name] = ts.Type
			}
		}
	}
	packages[importPath] = p
	return p
}

// typeRef 是解析之后的类型
type typeRef struct {
	kind  string // basic / struct / slice / map / any
	basic string // kind=basic 时的 TS 类型：string / number / boolean
	pkg   string // kind=struct 时的包名（namespace）
	name  string // kind=struct 时的类型名
	path  string // kind=struct 时的 import 路径
	elem  *typeRef
	key   *typeRef
}

// resolve 把源码里的类型表达式解析成 typeRef。file 用来查 import 别名。
func resolve(expr ast.Expr, importPath, file string) *typeRef {
	p := loadPackage(importPath)
	switch t := expr.(type) {
	case *ast.StarExpr:
		return resolve(t.X, importPath, file)
	case *ast.ArrayType:
		return &typeRef{kind: "slice", elem: resolve(t.Elt, importPath, file)}
	case *ast.MapType:
		return &typeRef{kind: "map", key: resolve(t.Key, importPath, file), elem: resolve(t.Value, importPath, file)}
	case *ast.InterfaceType:
		return &typeRef{kind: "any"}
	case *ast.Ident:
		switch {
		case t.Name == "string":
			return &typeRef{kind: "basic", basic: "string"}
		case t.Name == "bool":
			return &typeRef{kind: "basic", basic: "boolean"}
		case t.Name == "any":
			return &typeRef{kind: "any"}
		case t.Name == "byte":
			return &typeRef{kind: "basic", basic: "number"}
		case strings.HasPrefix(t.Name, "int"), strings.HasPrefix(t.Name, "uint"), strings.HasPrefix(t.Name, "float"):
			return &typeRef{kind: "basic", basic: "number"}
		case t.Name == "error":
			return &typeRef{kind: "basic", basic: "Error"}
		}
		def, ok := p.types[t.Name]
		if !ok {
			return &typeRef{kind: "any"}
		}
		if _, isStruct := def.(*ast.StructType); isStruct {
			return &typeRef{kind: "struct", pkg: p.name, name: t.Name, path: importPath}
		}
		// 比如 type Mode string，前端看到的就是底层类型
		return resolve(def, importPath, findTypeFile(p, t.Name))
	case *ast.SelectorExpr:
		alias := t.X.(*ast.Ident).Name
		path := p.imports[file][alias]
		if !strings.HasPrefix(path, modulePath+"/") {
			return &typeRef{kind: "any"} // 第三方包的类型（比如 time.Time）
		}
		return resolve(t.Sel, path, findTypeFile(loadPackage(path), t.Sel.Name))
	}
	return &typeRef{kind: "any"}
}

func findTypeFile(p *pkgInfo, name string) string {
	for file, f := range p.files {
		for _, decl := range f.Decls {
			if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
				for _, spec := range gd.Specs {
					if spec.(*ast.TypeSpec).Name.Name == name {
						return file
					}
				}
			}
		}
	}
	return ""
}

// methodType 是 App.d.ts 里用的写法（和 wails 一样：数组写 Array<T>，map 写 Record<K, V>）
func methodType(t *typeRef, imports *[]string) string {
	switch t.kind {
	case "basic":
		return t.basic
	case "struct":
		addUnique(imports, t.pkg)
		return t.pkg + "." + t.name
	case "slice":
		return "Array<" + methodType(t.elem, imports) + ">"
	case "map":
		return "Record<" + methodType(t.key, imports) + ", " + methodType(t.elem, imports) + ">"
	}
	return "any"
}

func addUnique(list *[]string, s string) {
	for _, v := range *list {
		if v == s {
			return
		}
	}
	*list = append(*list, s)
}

// ---- 收集 *App 的方法 ----

type method struct {
	name    string
	params  []*typeRef
	results []*typeRef
}

func collectMethods() []method {
	p := loadPackage(".")
	var methods []method
	for file, f := range p.files {
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || !fd.Name.IsExported() {
				continue
			}
			star, ok := fd.Recv.List[0].Type.(*ast.StarExpr)
			if !ok || star.X.(*ast.Ident).Name != "App" {
				continue
			}
			m := method{name: fd.Name.Name}
			for _, field := range fd.Type.Params.List {
				n := max(1, len(field.Names))
				for range n {
					m.params = append(m.params, resolve(field.Type, ".", file))
				}
			}
			if fd.Type.Results != nil {
				for _, field := range fd.Type.Results.List {
					n := max(1, len(field.Names))
					for range n {
						m.results = append(m.results, resolve(field.Type, ".", file))
					}
				}
			}
			methods = append(methods, m)
		}
	}
	sort.Slice(methods, func(i, j int) bool { return methods[i].name < methods[j].name })
	return methods
}

func isError(t *typeRef) bool { return t.kind == "basic" && t.basic == "Error" }

func writeAppFiles(methods []method) {
	var js, ts, body bytes.Buffer
	js.WriteString("// @ts-check\n// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL\n// This file is automatically generated. DO NOT EDIT\n")
	ts.WriteString("// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL\n// This file is automatically generated. DO NOT EDIT\n")
	var imports []string
	for _, m := range methods {
		var args, typed []string
		for i, p := range m.params {
			arg := fmt.Sprintf("arg%d", i+1)
			args = append(args, arg)
			typed = append(typed, arg+":"+methodType(p, &imports))
		}
		fmt.Fprintf(&js, "\nexport function %s(%s) {\n  return window['go']['main']['App']['%s'](%s);\n}\n",
			m.name, strings.Join(args, ", "), m.name, strings.Join(args, ", "))

		ret := "Promise<void>"
		if len(m.results) > 0 && !(len(m.results) == 1 && isError(m.results[0])) {
			ret = "Promise<" + methodType(m.results[0], &imports)
			if len(m.results) == 2 && !isError(m.results[1]) {
				ret += "|" + methodType(m.results[1], &imports)
			}
			ret += ">"
		}
		fmt.Fprintf(&body, "\nexport function %s(%s):%s;\n", m.name, strings.Join(typed, ","), ret)
	}
	for _, ns := range imports {
		fmt.Fprintf(&ts, "import {%s} from '../models';\n", ns)
	}
	ts.Write(body.Bytes())
	must(os.WriteFile("frontend/wailsjs/go/main/App.js", js.Bytes(), 0o644))
	must(os.WriteFile("frontend/wailsjs/go/main/App.d.ts", ts.Bytes(), 0o644))
}

// ---- models.ts ----

// structQueue 记录需要生成的结构体：namespace -> 类型名 -> typeRef
var structs = map[string]map[string]*typeRef{}

func collectStruct(t *typeRef) {
	switch t.kind {
	case "slice", "map":
		collectStruct(t.elem)
	case "struct":
		if structs[t.pkg] == nil {
			structs[t.pkg] = map[string]*typeRef{}
		}
		if _, done := structs[t.pkg][t.name]; done {
			return
		}
		structs[t.pkg][t.name] = t
		for _, f := range structFields(t) {
			collectStruct(f.typ)
		}
	}
}

type field struct {
	name     string
	optional bool
	typ      *typeRef
}

// structFields 按源码顺序列出会被 JSON 序列化的字段
func structFields(t *typeRef) []field {
	p := loadPackage(t.path)
	st := p.types[t.name].(*ast.StructType)
	file := findTypeFile(p, t.name)
	var fields []field
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			continue // 嵌入字段，这个项目里没用到
		}
		jsonName, optional := "", false
		if f.Tag != nil {
			tag, _ := strconv.Unquote(f.Tag.Value)
			parts := strings.Split(reflect.StructTag(tag).Get("json"), ",")
			jsonName = parts[0]
			for _, opt := range parts[1:] {
				if opt == "omitempty" {
					optional = true
				}
			}
		}
		if jsonName == "-" {
			continue
		}
		for _, n := range f.Names {
			if !n.IsExported() {
				continue
			}
			name := jsonName
			if name == "" {
				name = n.Name
			}
			fields = append(fields, field{name: name, optional: optional, typ: resolve(f.Type, t.path, file)})
		}
	}
	return fields
}

// modelType 是 models.ts 字段里的写法（数组写 T[]）
func modelType(t *typeRef, ns string) string {
	switch t.kind {
	case "basic":
		return t.basic
	case "struct":
		if t.pkg == ns {
			return t.name
		}
		return t.pkg + "." + t.name
	case "slice":
		return modelType(t.elem, ns) + "[]"
	}
	return "any"
}

const convertValues = `
	convertValues(a: any, classs: any, asMap: boolean = false): any {
	    if (!a) {
	        return a;
	    }
	    if (a.slice && a.map) {
	        return (a as any[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                a[key] = this.convertValues(a[key], classs);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}`

func classCode(t *typeRef) string {
	var b strings.Builder
	var fieldLines, ctorLines []string
	needConvert := false
	for _, f := range structFields(t) {
		name := f.name
		if f.optional {
			name += "?"
		}
		inner := f.typ
		for inner.kind == "slice" {
			inner = inner.elem
		}
		switch {
		case f.typ.kind == "map":
			valType := "any"
			if f.typ.elem.kind == "basic" {
				valType = f.typ.elem.basic
			}
			fieldLines = append(fieldLines, fmt.Sprintf("    %s: Record<%s, %s>;", name, modelType(f.typ.key, t.pkg), valType))
			ctorLines = append(ctorLines, fmt.Sprintf("        this.%s = source[\"%s\"];", f.name, f.name))
		case inner.kind == "struct":
			needConvert = true
			fieldLines = append(fieldLines, fmt.Sprintf("    %s: %s;", name, modelType(f.typ, t.pkg)))
			ctorLines = append(ctorLines, fmt.Sprintf("        this.%s = this.convertValues(source[\"%s\"], %s);", f.name, f.name, modelType(inner, t.pkg)))
		default:
			fieldLines = append(fieldLines, fmt.Sprintf("    %s: %s;", name, modelType(f.typ, t.pkg)))
			ctorLines = append(ctorLines, fmt.Sprintf("        this.%s = source[\"%s\"];", f.name, f.name))
		}
	}
	fmt.Fprintf(&b, "export class %s {\n", t.name)
	for _, l := range fieldLines {
		b.WriteString(l + "\n")
	}
	fmt.Fprintf(&b, "\n    static createFrom(source: any = {}) {\n        return new %s(source);\n    }\n", t.name)
	b.WriteString("\n    constructor(source: any = {}) {\n        if ('string' === typeof source) source = JSON.parse(source);\n")
	for _, l := range ctorLines {
		b.WriteString(l + "\n")
	}
	b.WriteString("    }\n")
	if needConvert {
		// convertValues 这段在 wails 的输出里本来就多一个 tab 缩进，原样照抄
		b.WriteString(convertValues + "\n")
	}
	b.WriteString("}")
	return b.String()
}

func writeModels(methods []method) {
	for _, m := range methods {
		for _, t := range append(append([]*typeRef{}, m.params...), m.results...) {
			collectStruct(t)
		}
	}
	var namespaces []string
	for ns := range structs {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)

	var out bytes.Buffer
	for _, ns := range namespaces {
		var names []string
		for name := range structs[ns] {
			names = append(names, name)
		}
		sort.Strings(names)
		code := ""
		for _, name := range names {
			code += "\n" + classCode(structs[ns][name])
		}
		out.WriteString("export namespace " + ns + " {\n")
		for _, line := range strings.Split(code, "\n") {
			out.WriteString("\t" + line + "\n")
		}
		out.WriteString("\n}\n\n")
	}
	must(os.WriteFile("frontend/wailsjs/go/models.ts", out.Bytes(), 0o644))
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	methods := collectMethods()
	writeAppFiles(methods)
	writeModels(methods)
	fmt.Printf("已生成 %d 个方法的绑定\n", len(methods))
}
