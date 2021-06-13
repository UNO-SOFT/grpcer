// Copyright 2016, 2021 Tamás Gulácsi
//
// SPDX-License-Identifier: Apache-2.0

// protoc-gen-grpc generates a grpcer.Client from the given protoc file.
package main

import (
	"io"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
)

var opts protogen.Options

func main() {
	opts.Run(Main)
}

func Main(p *protogen.Plugin) error {
	req := p.Request
	destPkg := req.GetParameter()
	if destPkg == "" {
		destPkg = "main"
	}

	// Find roots.
	rootNames := req.GetFileToGenerate()
	files := req.GetProtoFile()
	roots := make(map[string]*descriptorpb.FileDescriptorProto, len(rootNames))
	allTypes := make(map[string]*descriptorpb.DescriptorProto, 1024)
	var found int
	for i := len(files) - 1; i >= 0; i-- {
		f := files[i]
		for _, m := range f.GetMessageType() {
			allTypes["."+f.GetPackage()+"."+m.GetName()] = m
		}
		if found == len(rootNames) {
			continue
		}
		for _, root := range rootNames {
			if f.GetName() == root {
				roots[root] = files[i]
				found++
				break
			}
		}
	}

	msgTypes := make(map[string]*descriptorpb.DescriptorProto, len(allTypes))
	for _, root := range roots {
		//k := "." + root.GetName() + "."
		var k string
		for _, svc := range root.GetService() {
			for _, m := range svc.GetMethod() {
				if kk := k + m.GetInputType(); len(kk) > len(k) {
					msgTypes[kk] = allTypes[kk]
				}
				if kk := k + m.GetOutputType(); len(kk) > len(k) {
					msgTypes[kk] = allTypes[kk]
				}
			}
		}
	}

	for _, root := range roots {
		root := root
		pkg := root.GetName()
		for _, svc := range root.GetService() {
			svc := svc
			destFn := strings.TrimSuffix(filepath.Base(pkg), ".proto") + ".grpcer.go"
			if err := genGo(p.NewGeneratedFile(destFn, protogen.GoImportPath(pkg)), destPkg, pkg, svc, root.GetDependency()); err != nil {
				p.Error(err)
			}
		}
	}

	return nil
}

var goTmpl = template.Must(template.
	New("go").
	Funcs(template.FuncMap{
		"trimLeft":    strings.TrimLeft,
		"trimLeftDot": func(s string) string { return strings.TrimLeft(s, ".") },
		"base": func(s string) string {
			if i := strings.LastIndexByte(s, '.'); i >= 0 {
				return s[i+1:]
			}
			return s
		},
		"now": func(patterns ...string) string {
			pattern := time.RFC3339
			if len(patterns) > 0 {
				pattern = patterns[0]
			}
			return time.Now().Format(pattern)
		},
		"changePkgTo": func(from, to, what string) string {
			if j := strings.LastIndexByte(from, '/'); j >= 0 {
				from = from[j+1:]
			}
			if from != "" {
				if strings.HasPrefix(what, from+".") {
					return to + what[len(from):]
				}
				return what
			}
			i := strings.IndexByte(what, '.')
			if i < 0 {
				return what
			}
			return to + what[i:]
		},
	}).
	Parse(`// Generated with protoc-gen-grpcer
//	from "{{.ProtoFile}}"
//	at   {{now}}
//
// DO NOT EDIT!

package {{.Package}}

import (
	"context"
	"fmt"
	"io"

	grpc "google.golang.org/grpc"
	grpcer "github.com/UNO-SOFT/grpcer"

	pb "{{.Import}}"
	{{range .Dependencies}}"{{.}}"
	{{end}}
)

{{ $import := .Import }}

type client struct {
	pb.{{.GetName}}Client
	m map[string]inputAndCall
}

func (c client) List() []string {
	names := make([]string, 0, len(c.m))
	for k := range c.m {
		names = append(names, k)
	}
	return names
}

func (c client) Input(name string) interface{} {
	iac := c.m[name]
	if iac.Input == nil {
		return nil
	}
	return iac.Input()
}

func (c client) Call(name string, ctx context.Context, in interface{}, opts ...grpc.CallOption) (grpcer.Receiver, error) {
	iac := c.m[name]
	if iac.Call == nil {
		return nil, fmt.Errorf("name %q not found", name)
	}
	return iac.Call(ctx, in, opts...)
}
func NewClient(cc *grpc.ClientConn) grpcer.Client {
	c := pb.New{{.GetName}}Client(cc)
	return client{
		{{.GetName}}Client: c,
		m: map[string]inputAndCall{
		{{range .GetMethod}}"{{.GetName}}": inputAndCall{
			Input: func() interface{} { return new({{ trimLeftDot .GetInputType | changePkgTo $import "pb" }}) },
			Call: func(ctx context.Context, in interface{}, opts ...grpc.CallOption) (grpcer.Receiver, error) {
				input := in.(*{{ trimLeftDot .GetInputType | changePkgTo $import "pb" }})
				res, err := c.{{.Name}}(ctx, input, opts...)
				if err != nil {
					return &onceRecv{Out:res}, err
				}
				{{if .GetServerStreaming -}}
				return multiRecv(func() (interface{}, error) { return res.Recv() }), nil
				{{else -}}
				return &onceRecv{Out:res}, err
				{{end}}
			},
		},
		{{end}}
		},
	}
}

type inputAndCall struct {
	Input func() interface{}
	Call func(ctx context.Context, in interface{}, opts ...grpc.CallOption) (grpcer.Receiver, error)
}

type onceRecv struct {
	Out interface{}
	done bool
}
func (o *onceRecv) Recv() (interface{}, error) {
	if o.done {
		return nil, io.EOF
	}
	out := o.Out
	o.done, o.Out = true, nil
	return out, nil
}

type multiRecv func() (interface{}, error)
func (m multiRecv) Recv() (interface{}, error) {
	return m()
}

var _ = multiRecv(nil) // against "unused"

`))

func genGo(w io.Writer, destPkg, protoFn string, svc *descriptorpb.ServiceDescriptorProto, dependencies []string) error {
	if destPkg == "" {
		destPkg = "main"
	}
	needed := make(map[string]struct{}, len(dependencies))
	for _, m := range svc.GetMethod() {
		//for _, t := range []string{m.GetInputType(), m.GetOutputType()} {
		t := m.GetInputType()
		if !strings.HasPrefix(t, ".") {
			continue
		}
		t = t[1:]
		needed[strings.SplitN(t, ".", 2)[0]] = struct{}{}
	}
	deps := make([]string, 0, len(dependencies))
	for _, dep := range dependencies {
		k := filepath.Dir(dep)
		if _, ok := needed[filepath.Base(k)]; !ok {
			continue
		}
		deps = append(deps, k)
	}
	return goTmpl.Execute(w, struct {
		*descriptorpb.ServiceDescriptorProto
		ProtoFile, Package, Import string
		Dependencies               []string
	}{
		ProtoFile:              protoFn,
		Package:                destPkg,
		Import:                 filepath.Dir(protoFn),
		Dependencies:           deps,
		ServiceDescriptorProto: svc,
	})
}

// vim: set fileencoding=utf-8 noet:
