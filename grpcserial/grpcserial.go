// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2015 The Go Authors.  All rights reserved.
// https://github.com/golang/protobuf
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// Package grpc outputs gRPC service descriptions in Go code.
// It runs as a plugin for the Go protocol buffer compiler plugin.
// It is linked in to protoc-gen-go.
package grpcserial

import (
    "fmt"
    "path"
    "strconv"
    "strings"

    pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
    "github.com/golang/protobuf/protoc-gen-go/generator"
)

// generatedCodeVersion indicates a version of the generated code.
// It is incremented whenever an incompatibility between the generated code and
// the grpc package is introduced; the generated code references
// a constant, grpc.SupportPackageIsVersionN (where N is generatedCodeVersion).
const generatedCodeVersion = 4

// Paths for packages used by code generated in this file,
// relative to the import_prefix of the generator.Generator.
const (
    contextPkgPath = "golang.org/x/net/context"
    grpcPkgPath    = "google.golang.org/grpc"
)

func init() {
    generator.RegisterPlugin(new(grpc))
}

// grpc is an implementation of the Go protocol buffer compiler's
// plugin architecture.  It generates bindings for gRPC support.
type grpc struct {
    gen *generator.Generator
}

// Name returns the name of this plugin, "grpc".
func (g *grpc) Name() string {
    return "grpcserial"
}

// The names for packages imported in the generated code.
// They may vary from the final path component of the import path
// if the name is used by other packages.
var (
    contextPkg string
    grpcPkg    string
)

// Init initializes the plugin.
func (g *grpc) Init(gen *generator.Generator) {
    g.gen = gen
    contextPkg = generator.RegisterUniquePackageName("context", nil)
    grpcPkg = generator.RegisterUniquePackageName("grpcserial", nil)
}

// Given a type name defined in a .proto, return its object.
// Also record that we're using it, to guarantee the associated import.
func (g *grpc) objectNamed(name string) generator.Object {
    g.gen.RecordTypeUse(name)
    return g.gen.ObjectNamed(name)
}

// Given a type name defined in a .proto, return its name as we will print it.
func (g *grpc) typeName(str string) string {
    return g.gen.TypeName(g.objectNamed(str))
}

// P forwards to g.gen.P.
func (g *grpc) P(args ...interface{}) { g.gen.P(args...) }

// Generate generates code for the services in the given file.
func (g *grpc) Generate(file *generator.FileDescriptor) {
    if len(file.FileDescriptorProto.Service) == 0 {
        return
    }

    g.P("// Reference imports to suppress errors if they are not otherwise used.")
    g.P("var _ ", contextPkg, ".Context")
    g.P("var _ ", grpcPkg, ".ClientConn")
    g.P()

    // Assert version compatibility.
    g.P("// This is a compile-time assertion to ensure that this generated file")
    g.P("// is compatible with the grpc package it is being compiled against.")
    g.P("const _ = ", grpcPkg, ".SupportPackageIsVersion", generatedCodeVersion)
    g.P()

    for i, service := range file.FileDescriptorProto.Service {
        g.generateService(file, service, i)
    }
}

// GenerateImports generates the import declaration for this file.
func (g *grpc) GenerateImports(file *generator.FileDescriptor) {
    if len(file.FileDescriptorProto.Service) == 0 {
        return
    }
    g.P("import (")
    g.P(contextPkg, " ", strconv.Quote(path.Join(g.gen.ImportPrefix, contextPkgPath)))
    g.P(grpcPkg, " ", strconv.Quote(path.Join(g.gen.ImportPrefix, grpcPkgPath)))
    g.P(")")
    g.P()
}

// reservedClientName records whether a client name is reserved on the client side.
var reservedClientName = map[string]bool{
// TODO: do we need any in gRPC?
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
func (g *grpc) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
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
    g.P("package your_package")
    g.P()
    g.P("import \"github.com/golang/protobuf/proto\"")
    g.P(fmt.Sprintf("import \"%s\"", goPackage))
    g.P()
    g.P("//go:generate goprotopy $GOPACKAGE $GOFILE")
    g.P()

    for i, method := range service.Method {
        g.gen.PrintComments(fmt.Sprintf("%s,2,%d", path, i)) // 2 means method in a service.
        g.generateSerializedAPI(goPackage, servName, method)
    }
    g.P("*/")
    g.P()

    // // Client structure.
    // g.P("type ", unexport(servName), "Client struct {")
    // g.P("cc *", grpcPkg, ".ClientConn")
    // g.P("}")
    // g.P()

    // // NewClient factory.
    // g.P("func New", servName, "Client (cc *", grpcPkg, ".ClientConn) ", servName, "Client {")
    // g.P("return &", unexport(servName), "Client{cc}")
    // g.P("}")
    // g.P()

    // var methodIndex, streamIndex int
    // serviceDescVar := "_" + servName + "_serviceDesc"
    // // Client method implementations.
    // for _, method := range service.Method {
    //     var descExpr string
    //     if !method.GetServerStreaming() && !method.GetClientStreaming() {
    //         // Unary RPC method
    //         descExpr = fmt.Sprintf("&%s.Methods[%d]", serviceDescVar, methodIndex)
    //         methodIndex++
    //     } else {
    //         // Streaming RPC method
    //         descExpr = fmt.Sprintf("&%s.Streams[%d]", serviceDescVar, streamIndex)
    //         streamIndex++
    //     }
    //     g.generateClientMethod(servName, fullServName, serviceDescVar, method, descExpr)
    // }

    // g.P("// Server API for ", servName, " service")
    // g.P()

    // // Server interface.
    // serverType := servName + "Server"
    // g.P("type ", serverType, " interface {")
    // for i, method := range service.Method {
    //     g.gen.PrintComments(fmt.Sprintf("%s,2,%d", path, i)) // 2 means method in a service.
    //     g.P(g.generateServerSignature(servName, method))
    // }
    // g.P("}")
    // g.P()

    // // Server registration.
    // g.P("func Register", servName, "Server(s *", grpcPkg, ".Server, srv ", serverType, ") {")
    // g.P("s.RegisterService(&", serviceDescVar, `, srv)`)
    // g.P("}")
    // g.P()

    // // Server handler implementations.
    // var handlerNames []string
    // for _, method := range service.Method {
    //     hname := g.generateServerMethod(servName, fullServName, method)
    //     handlerNames = append(handlerNames, hname)
    // }

    // // Service descriptor.
    // g.P("var ", serviceDescVar, " = ", grpcPkg, ".ServiceDesc {")
    // g.P("ServiceName: ", strconv.Quote(fullServName), ",")
    // g.P("HandlerType: (*", serverType, ")(nil),")
    // g.P("Methods: []", grpcPkg, ".MethodDesc{")
    // for i, method := range service.Method {
    //     if method.GetServerStreaming() || method.GetClientStreaming() {
    //         continue
    //     }
    //     g.P("{")
    //     g.P("MethodName: ", strconv.Quote(method.GetName()), ",")
    //     g.P("Handler: ", handlerNames[i], ",")
    //     g.P("},")
    // }
    // g.P("},")
    // g.P("Streams: []", grpcPkg, ".StreamDesc{")
    // for i, method := range service.Method {
    //     if !method.GetServerStreaming() && !method.GetClientStreaming() {
    //         continue
    //     }
    //     g.P("{")
    //     g.P("StreamName: ", strconv.Quote(method.GetName()), ",")
    //     g.P("Handler: ", handlerNames[i], ",")
    //     if method.GetServerStreaming() {
    //         g.P("ServerStreams: true,")
    //     }
    //     if method.GetClientStreaming() {
    //         g.P("ClientStreams: true,")
    //     }
    //     g.P("},")
    // }
    // g.P("},")
    // g.P("Metadata: \"", file.GetName(), "\",")
    // g.P("}")
    g.P()
}

func (g *grpc) generateSerializedAPI(goPackage string, servName string, method *pb.MethodDescriptorProto) {
    origMethodName := method.GetName()
    methodName := generator.CamelCase(origMethodName)
    if reservedClientName[methodName] {
        methodName += "_"
    }
    inputTypeName := g.typeName(method.GetInputType())
    inputVarName := unexport(inputTypeName)
    outputVarName := unexport(g.typeName(method.GetOutputType()))
    
    g.P("// @protopy")
    g.P(fmt.Sprintf("func %s(input []byte) (output []byte, err error) {", methodName))
    g.P(fmt.Sprintf("    %s := new(%s.%s)", inputVarName, goPackage, inputTypeName))
    g.P(fmt.Sprintf("    err = proto.Unmarshal(input, %s)", inputVarName))
    g.P("    if err != nil {")
    g.P("        return")
    g.P("    }")
    g.P(fmt.Sprintf("    %s, err := your%sImplementation(%s)", outputVarName, methodName, inputVarName))
    g.P(fmt.Sprintf("    output, err = proto.Marshal(%s)", outputVarName))
    g.P("    return")
    g.P("}")
    g.P()
}

// generateClientSignature returns the client-side signature for a method.
func (g *grpc) generateClientSignature(servName string, method *pb.MethodDescriptorProto) string {
    origMethName := method.GetName()
    methName := generator.CamelCase(origMethName)
    if reservedClientName[methName] {
        methName += "_"
    }
    reqArg := ", in *" + g.typeName(method.GetInputType())
    if method.GetClientStreaming() {
        reqArg = ""
    }
    respName := "*" + g.typeName(method.GetOutputType())
    if method.GetServerStreaming() || method.GetClientStreaming() {
        respName = servName + "_" + generator.CamelCase(origMethName) + "Client"
    }
    return fmt.Sprintf("%s(ctx %s.Context%s, opts ...%s.CallOption) (%s, error)", methName, contextPkg, reqArg, grpcPkg, respName)
}

func (g *grpc) generateClientMethod(servName, fullServName, serviceDescVar string, method *pb.MethodDescriptorProto, descExpr string) {
    sname := fmt.Sprintf("/%s/%s", fullServName, method.GetName())
    methName := generator.CamelCase(method.GetName())
    inType := g.typeName(method.GetInputType())
    outType := g.typeName(method.GetOutputType())

    g.P("func (c *", unexport(servName), "Client) ", g.generateClientSignature(servName, method), "{")
    if !method.GetServerStreaming() && !method.GetClientStreaming() {
        g.P("out := new(", outType, ")")
        // TODO: Pass descExpr to Invoke.
        g.P("err := ", grpcPkg, `.Invoke(ctx, "`, sname, `", in, out, c.cc, opts...)`)
        g.P("if err != nil { return nil, err }")
        g.P("return out, nil")
        g.P("}")
        g.P()
        return
    }
    streamType := unexport(servName) + methName + "Client"
    g.P("stream, err := ", grpcPkg, ".NewClientStream(ctx, ", descExpr, `, c.cc, "`, sname, `", opts...)`)
    g.P("if err != nil { return nil, err }")
    g.P("x := &", streamType, "{stream}")
    if !method.GetClientStreaming() {
        g.P("if err := x.ClientStream.SendMsg(in); err != nil { return nil, err }")
        g.P("if err := x.ClientStream.CloseSend(); err != nil { return nil, err }")
    }
    g.P("return x, nil")
    g.P("}")
    g.P()

    genSend := method.GetClientStreaming()
    genRecv := method.GetServerStreaming()
    genCloseAndRecv := !method.GetServerStreaming()

    // Stream auxiliary types and methods.
    g.P("type ", servName, "_", methName, "Client interface {")
    if genSend {
        g.P("Send(*", inType, ") error")
    }
    if genRecv {
        g.P("Recv() (*", outType, ", error)")
    }
    if genCloseAndRecv {
        g.P("CloseAndRecv() (*", outType, ", error)")
    }
    g.P(grpcPkg, ".ClientStream")
    g.P("}")
    g.P()

    g.P("type ", streamType, " struct {")
    g.P(grpcPkg, ".ClientStream")
    g.P("}")
    g.P()

    if genSend {
        g.P("func (x *", streamType, ") Send(m *", inType, ") error {")
        g.P("return x.ClientStream.SendMsg(m)")
        g.P("}")
        g.P()
    }
    if genRecv {
        g.P("func (x *", streamType, ") Recv() (*", outType, ", error) {")
        g.P("m := new(", outType, ")")
        g.P("if err := x.ClientStream.RecvMsg(m); err != nil { return nil, err }")
        g.P("return m, nil")
        g.P("}")
        g.P()
    }
    if genCloseAndRecv {
        g.P("func (x *", streamType, ") CloseAndRecv() (*", outType, ", error) {")
        g.P("if err := x.ClientStream.CloseSend(); err != nil { return nil, err }")
        g.P("m := new(", outType, ")")
        g.P("if err := x.ClientStream.RecvMsg(m); err != nil { return nil, err }")
        g.P("return m, nil")
        g.P("}")
        g.P()
    }
}

// generateServerSignature returns the server-side signature for a method.
func (g *grpc) generateServerSignature(servName string, method *pb.MethodDescriptorProto) string {
    origMethName := method.GetName()
    methName := generator.CamelCase(origMethName)
    if reservedClientName[methName] {
        methName += "_"
    }

    var reqArgs []string
    ret := "error"
    if !method.GetServerStreaming() && !method.GetClientStreaming() {
        reqArgs = append(reqArgs, contextPkg+".Context")
        ret = "(*" + g.typeName(method.GetOutputType()) + ", error)"
    }
    if !method.GetClientStreaming() {
        reqArgs = append(reqArgs, "*"+g.typeName(method.GetInputType()))
    }
    if method.GetServerStreaming() || method.GetClientStreaming() {
        reqArgs = append(reqArgs, servName+"_"+generator.CamelCase(origMethName)+"Server")
    }

    return methName + "(" + strings.Join(reqArgs, ", ") + ") " + ret
}

func (g *grpc) generateServerMethod(servName, fullServName string, method *pb.MethodDescriptorProto) string {
    methName := generator.CamelCase(method.GetName())
    hname := fmt.Sprintf("_%s_%s_Handler", servName, methName)
    inType := g.typeName(method.GetInputType())
    outType := g.typeName(method.GetOutputType())

    if !method.GetServerStreaming() && !method.GetClientStreaming() {
        g.P("func ", hname, "(srv interface{}, ctx ", contextPkg, ".Context, dec func(interface{}) error, interceptor ", grpcPkg, ".UnaryServerInterceptor) (interface{}, error) {")
        g.P("in := new(", inType, ")")
        g.P("if err := dec(in); err != nil { return nil, err }")
        g.P("if interceptor == nil { return srv.(", servName, "Server).", methName, "(ctx, in) }")
        g.P("info := &", grpcPkg, ".UnaryServerInfo{")
        g.P("Server: srv,")
        g.P("FullMethod: ", strconv.Quote(fmt.Sprintf("/%s/%s", fullServName, methName)), ",")
        g.P("}")
        g.P("handler := func(ctx ", contextPkg, ".Context, req interface{}) (interface{}, error) {")
        g.P("return srv.(", servName, "Server).", methName, "(ctx, req.(*", inType, "))")
        g.P("}")
        g.P("return interceptor(ctx, in, info, handler)")
        g.P("}")
        g.P()
        return hname
    }
    streamType := unexport(servName) + methName + "Server"
    g.P("func ", hname, "(srv interface{}, stream ", grpcPkg, ".ServerStream) error {")
    if !method.GetClientStreaming() {
        g.P("m := new(", inType, ")")
        g.P("if err := stream.RecvMsg(m); err != nil { return err }")
        g.P("return srv.(", servName, "Server).", methName, "(m, &", streamType, "{stream})")
    } else {
        g.P("return srv.(", servName, "Server).", methName, "(&", streamType, "{stream})")
    }
    g.P("}")
    g.P()

    genSend := method.GetServerStreaming()
    genSendAndClose := !method.GetServerStreaming()
    genRecv := method.GetClientStreaming()

    // Stream auxiliary types and methods.
    g.P("type ", servName, "_", methName, "Server interface {")
    if genSend {
        g.P("Send(*", outType, ") error")
    }
    if genSendAndClose {
        g.P("SendAndClose(*", outType, ") error")
    }
    if genRecv {
        g.P("Recv() (*", inType, ", error)")
    }
    g.P(grpcPkg, ".ServerStream")
    g.P("}")
    g.P()

    g.P("type ", streamType, " struct {")
    g.P(grpcPkg, ".ServerStream")
    g.P("}")
    g.P()

    if genSend {
        g.P("func (x *", streamType, ") Send(m *", outType, ") error {")
        g.P("return x.ServerStream.SendMsg(m)")
        g.P("}")
        g.P()
    }
    if genSendAndClose {
        g.P("func (x *", streamType, ") SendAndClose(m *", outType, ") error {")
        g.P("return x.ServerStream.SendMsg(m)")
        g.P("}")
        g.P()
    }
    if genRecv {
        g.P("func (x *", streamType, ") Recv() (*", inType, ", error) {")
        g.P("m := new(", inType, ")")
        g.P("if err := x.ServerStream.RecvMsg(m); err != nil { return nil, err }")
        g.P("return m, nil")
        g.P("}")
        g.P()
    }

    return hname
}
