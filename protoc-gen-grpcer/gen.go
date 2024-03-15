// Copyright 2016, 2021 Tamás Gulácsi
//
// SPDX-License-Identifier: Apache-2.0

// protoc-gen-grpc generates a grpcer.Client from the given protoc file.
package main

// nosemgrep: go.lang.security.audit.xss.import-text-template.import-text-template
import (
	"io"
	"path/filepath"
	"strings"

	"github.com/valyala/quicktemplate"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
)

//go:generate qtc

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
func getTags(m *descriptorpb.MethodDescriptorProto) []string {
	var tags []string
	opts := m.GetOptions()
	if opts == nil {
		return nil
	}
	var buf strings.Builder
	for _, o := range opts.GetUninterpretedOption() {
		buf.Reset()
		for i, p := range o.GetName() {
			if i != 0 {
				buf.WriteByte('.')
			}
			buf.WriteString(p.GetNamePart())
		}
		if nm := buf.String(); nm == "oracall.orasrv.tag" {
			tags = append(tags, string(o.GetStringValue()))
		}
	}
	return tags
}
func trimLeftDot(s string) string { return strings.TrimLeft(s, ".") }
func changePkgTo(from, to, what string) string {
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
}

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
	W := quicktemplate.AcquireWriter(w)
	StreamXGo(W, svc, FileInfo{
		ProtoFile:    protoFn,
		Package:      destPkg,
		Import:       filepath.Dir(protoFn),
		Dependencies: deps,
	})
	quicktemplate.ReleaseWriter(W)
	return nil
}

type FileInfo struct {
	ProtoFile, Package, Import string
	Dependencies               []string
}

// vim: set fileencoding=utf-8 noet:
