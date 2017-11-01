// Package grpcserial outputs API stubs in commented Go code.
//
// Those stubs are compilable with gomobile and generate Python bindings
// with goprotopy.
// It runs as a plugin for the Go protocol buffer compiler plugin.
// It is linked in to protoc-gen-go.
//
// Disclaimer :
// - this project is loosely based on the grpc plugin for protoc-gen-go
// - the code is provided as is, without any warranty

package grpcserial

import (
    "fmt"
    "strings"

    pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
    "github.com/golang/protobuf/protoc-gen-go/generator"
)

func init() {
    generator.RegisterPlugin(new(grpcserial))
}

// grpcserial is an implementation of the Go protocol buffer compiler's
// plugin architecture. It generates an example implementation of an API
// based on its grpc description - the implementation is compilable with
// gomobile and has annotations for Python bindings generation with goprotopy.
type grpcserial struct {
    gen *generator.Generator
}

// Name returns the name of this plugin, "grpcserial".
func (g *grpcserial) Name() string {
    return "grpcserial"
}

// Init initializes the plugin.
func (g *grpcserial) Init(gen *generator.Generator) {
    g.gen = gen
}

// Given a type name defined in a .proto, return its object.
// Also record that we're using it, to guarantee the associated import.
func (g *grpcserial) objectNamed(name string) generator.Object {
    g.gen.RecordTypeUse(name)
    return g.gen.ObjectNamed(name)
}

// Given a type name defined in a .proto, return its name as we will print it.
func (g *grpcserial) typeName(str string) string {
    return g.gen.TypeName(g.objectNamed(str))
}

// P forwards to g.gen.P.
func (g *grpcserial) P(args ...interface{}) { g.gen.P(args...) }

// Generate generates code for the services in the given file.
func (g *grpcserial) Generate(file *generator.FileDescriptor) {
    for i, service := range file.FileDescriptorProto.Service {
        g.generateService(file, service, i)
    }
}

// GenerateImports generates the import declaration for this file.
func (g *grpcserial) GenerateImports(file *generator.FileDescriptor) {
}

func unexport(s string) string { return strings.ToLower(s[:1]) + s[1:] }

// baseName returns the last path element of the name, with the last dotted suffix removed.
func baseName(name string) string {
    // First, find the last element
    if i := strings.LastIndex(name, "/"); i >= 0 {
        name = name[i+1:]
    }
    // Now drop the suffix
    if i := strings.LastIndex(name, "."); i >= 0 {
        name = name[0:i]
    }
    return name
}

// goPackageOption interprets the file's go_package option.
// If there is no go_package, it returns ("", "", false).
// If there's a simple name, it returns ("", pkg, true).
// If the option implies an import path, it returns (impPath, pkg, true).
func goPackageOption(d *generator.FileDescriptor) (impPath, pkg string, ok bool) {
    pkg = d.GetOptions().GetGoPackage()
    if pkg == "" {
        return
    }
    ok = true
    // The presence of a slash implies there's an import path.
    slash := strings.LastIndex(pkg, "/")
    if slash < 0 {
        return
    }
    impPath, pkg = pkg, pkg[slash+1:]
    // A semicolon-delimited suffix overrides the package name.
    sc := strings.IndexByte(impPath, ';')
    if sc < 0 {
        return
    }
    impPath, pkg = impPath[:sc], impPath[sc+1:]
    return
}

// goPackageName returns the Go package name to use in the
// generated Go file.  The result explicit reports whether the name
// came from an option go_package statement.  If explicit is false,
// the name was derived from the protocol buffer's package statement
// or the input file name.
func goPackageName(d *generator.FileDescriptor) (name string, explicit bool) {
    // Does the file have a "go_package" option?
    if _, pkg, ok := goPackageOption(d); ok {
        return pkg, true
    }

    // Does the file have a package clause?
    if pkg := d.GetPackage(); pkg != "" {
        return pkg, false
    }
    // Use the file base name.
    return baseName(d.GetName()), false
}

// generateService generates all the code for the named service.
func (g *grpcserial) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
    path := fmt.Sprintf("6,%d", index) // 6 means service.
    
    goPackage, _ := goPackageName(file)
    
    origServName := service.GetName()
    fullServName := origServName
    if pkg := file.GetPackage(); pkg != "" {
        fullServName = pkg + "." + fullServName
    }
    servName := generator.CamelCase(origServName)

    g.P("/* Example implementation of ", servName, " service :")
    g.P()
    g.P("package your_package // TODO change to your project package name")
    g.P()
    g.P("import \"github.com/golang/protobuf/proto\"")
    g.P(fmt.Sprintf("import pb \"%s\" // TODO change to the Go package in which your .pb.go has been generated", goPackage))
    g.P()
    g.P("//go:generate goprotopy $GOPACKAGE $GOFILE")
    g.P()

    for i, method := range service.Method {
        g.gen.PrintComments(fmt.Sprintf("%s,2,%d", path, i)) // 2 means method in a service.
        g.generateSerializedAPI(servName, method)
    }
    g.P("*/")
    g.P()
}

func (g *grpcserial) generateSerializedAPI(servName string, method *pb.MethodDescriptorProto) {
    origMethodName := method.GetName()
    methodName := generator.CamelCase(origMethodName)

    inputTypeName := g.typeName(method.GetInputType())
    inputVarName := unexport(inputTypeName)
    outputTypeName := g.typeName(method.GetOutputType())
    outputVarName := unexport(outputTypeName)
    
    g.P(fmt.Sprintf("// input is a serialized protobuf object of type %s", inputTypeName))
    g.P(fmt.Sprintf("// output is a serialized protobuf object of type %s", outputTypeName))
    g.P("// @protopy")
    g.P(fmt.Sprintf("func %s(input []byte) (output []byte, err error) {", methodName))
    g.P(fmt.Sprintf("    %s := new(pb.%s)", inputVarName, inputTypeName))
    g.P(fmt.Sprintf("    err = proto.Unmarshal(input, %s)", inputVarName))
    g.P("    if err != nil {")
    g.P("        return")
    g.P("    }")
    g.P()
    g.P(fmt.Sprintf("    // TODO : implement %s(%s *pb.%s) (*pb.%s, error)", methodName, inputVarName, inputTypeName, outputTypeName))
    g.P(fmt.Sprintf("    // %s, err := your%sImplementation(%s)", outputVarName, methodName, inputVarName))
    g.P()
    g.P(fmt.Sprintf("    %s := new(pb.%s)", outputVarName, outputTypeName))
    g.P(fmt.Sprintf("    output, err = proto.Marshal(%s)", outputVarName))
    g.P("    return")
    g.P("}")
    g.P()
}
