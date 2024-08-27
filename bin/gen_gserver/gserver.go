/*
 *
 * Copyright 2020 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// from https://github.com/grpc/grpc-go/blob/cmd/protoc-gen-go-grpc/v1.3.0/cmd/protoc-gen-go-grpc/grpc.go

package main

import (
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	contextPackage = protogen.GoImportPath("context")
	fmtPackage     = protogen.GoImportPath("fmt")
	grpcPackage    = protogen.GoImportPath("google.golang.org/grpc")
	codesPackage   = protogen.GoImportPath("google.golang.org/grpc/codes")
	statusPackage  = protogen.GoImportPath("google.golang.org/grpc/status")
	syncPackage    = protogen.GoImportPath("sync")
	giraPackage    = protogen.GoImportPath("github.com/Lyndon-Zhang/gira")
	facadePackage  = protogen.GoImportPath("github.com/Lyndon-Zhang/gira/facade")
	optionsPackage = protogen.GoImportPath("github.com/Lyndon-Zhang/gira/options/service_options")
	metaPackage    = protogen.GoImportPath("google.golang.org/grpc/metadata")
	errorsPackage  = protogen.GoImportPath("github.com/Lyndon-Zhang/gira/errors")
)

type serviceGenerateHelperInterface interface {
	formatFullMethodSymbol(service *protogen.Service, method *protogen.Method) string
	genServiceName(g *protogen.GeneratedFile, service *protogen.Service)
	genFullMethods(g *protogen.GeneratedFile, service *protogen.Service)
	generateServerRouterType(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service)
	generateServerRouterInterface(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service)
	generateServerRouterHandler(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service)
	generateServerRouterMiddleware(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service)
	generateServerRouterMiddlewareContext(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service)
	generateServerFunctions(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service, serverType string, serviceDescVar string)
	formatHandlerFuncName(service *protogen.Service, hname string) string
}

type serviceGenerateHelper struct{}

func (serviceGenerateHelper) formatFullMethodSymbol(service *protogen.Service, method *protogen.Method) string {
	return fmt.Sprintf("%s_%s_FullMethodName", service.GoName, method.GoName)
}

func (serviceGenerateHelper) genServiceName(g *protogen.GeneratedFile, service *protogen.Service) {
	g.P("const (")
	g.P(service.GoName, `ServiceName = "`, service.Desc.FullName(), `"`)
	g.P(")")
	g.P()
}

func (serviceGenerateHelper) genFullMethods(g *protogen.GeneratedFile, service *protogen.Service) {
	g.P("const (")
	for _, method := range service.Methods {
		fmSymbol := helper.formatFullMethodSymbol(service, method)
		fmName := fmt.Sprintf("/%s/%s", service.Desc.FullName(), method.Desc.Name())
		g.P(fmSymbol, ` = "`, fmName, `"`)
	}
	g.P(")")
	g.P()
}

func (serviceGenerateHelper) generateServerRouterHandler(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service) {
	serverRouterType := service.GoName + "ServerRouter"
	serverType := service.GoName + "Server"
	// Server Unimplemented struct for forward compatibility.
	g.P("// ", serverRouterType, " is the default server router handler for ", service.GoName, " service.")
	g.P("type ", serverRouterType, "Handler interface {")
	g.P("    ", serverType)
	g.P("}")
	g.P()
}

func (serviceGenerateHelper) generateServerRouterMiddleware(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service) {
	serverRouterType := service.GoName + "ServerRouter"
	// Server Unimplemented struct for forward compatibility.
	g.P("// ", serverRouterType, " is the default server router middleware for ", service.GoName, " service.")
	g.P("type ", serverRouterType, "Middleware interface {")
	g.P("    ", serverRouterType, "MiddlewareInvoke(ctx ", serverRouterType, "MiddlewareContext) error")
	g.P("}")
	g.P()
}

func (serviceGenerateHelper) generateServerRouterMiddlewareContext(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service) {
	serverRouterType := service.GoName + "ServerRouter"
	serverType := service.GoName + "Server"
	g.P("// ", serverRouterType, " is the default server router middleware context for ", service.GoName, " service.")
	g.P("type ", serverRouterType, "MiddlewareContext interface {")
	g.P("    Handler() ", serverType)
	g.P("    Next()")
	g.P("    Wait() error")
	g.P("}")
	g.P()

	g.P("type ", unexport(serverRouterType), "MiddlewareContext struct {")
	g.P("	 handler ", serverType)
	g.P("    invoke func() (resp interface{}, err error)")
	g.P("    ctx ", contextPackage.Ident("Context"))
	g.P("	 fullMethod string")
	g.P("	 in interface{}")
	g.P("	 out interface{}")
	g.P("	 err error")
	g.P("	 method string")
	g.P("	 c chan struct{}")
	g.P("}")
	g.P("func (m *", unexport(serverRouterType), "MiddlewareContext) Handler() ", serverType, " {")
	g.P("    return m.handler")
	g.P("}")
	g.P("func (m *", unexport(serverRouterType), "MiddlewareContext) Next() {")
	g.P("    defer func(){")
	g.P("        if e := recover(); e != nil {")
	g.P("            m.err = e.(error)")
	g.P("        }")
	g.P("    }()")
	g.P("    m.out, m.err = m.invoke()")
	g.P("    if m.c != nil {")
	g.P("        m.c <- struct{}{}")
	g.P("    }")
	g.P("}")
	g.P("func (m *", unexport(serverRouterType), "MiddlewareContext) Wait() error {")
	g.P("    if m.c == nil {")
	g.P("        m.c = make(chan struct{}, 1)")
	g.P("    }")
	g.P("    select {")
	g.P("    case <-m.c:")
	g.P("        return nil")
	g.P("    case <-m.ctx.Done():")
	g.P("        m.err = m.ctx.Err()")
	g.P("        return m.err")
	g.P("    }")
	g.P("}")
	g.P()

}

func (serviceGenerateHelper) generateServerRouterInterface(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service) {
	serverType := service.GoName + "ServerRouter"
	// Server Unimplemented struct for forward compatibility.
	g.P("// ", serverType, " is the default server router for ", service.GoName, " service.")
	g.P("type ", serverType, " interface {")
	g.P("RegisterMiddleware(middle ", serverType, "Middleware)")
	g.P("RegisterHandler(key string, handler ", serverType, "Handler)")
	g.P("UnregisterHandler(key string, handler ", serverType, "Handler)")
	g.P("}")
	g.P()
}

func (serviceGenerateHelper) generateServerRouterType(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service) {
	serverRouterType := service.GoName + "ServerRouter"
	serverType := service.GoName + "Server"
	// Server Unimplemented struct for forward compatibility.
	g.P("// ", unexport(serverRouterType), " is the default server router for ", service.GoName, " service.")
	g.P("type ", unexport(serverRouterType), " struct {")
	g.P("    Unimplemented", serverType)
	g.P("    mu ", syncPackage.Ident("Mutex"))
	g.P("    handlers ", syncPackage.Ident("Map"))
	g.P("    middleware ", serverRouterType, "Middleware")
	g.P("}")
	g.P()
	g.P("func (svr *", unexport(serverRouterType), ")RegisterMiddleware(middleware ", serverRouterType, "Middleware){")
	g.P("	svr.middleware = middleware")
	g.P("}")
	g.P()
	g.P("func (svr *", unexport(serverRouterType), ")RegisterHandler(key string, handler ", serverRouterType, "Handler){")
	g.P("	svr.handlers.Store(key, handler)")
	g.P("}")
	g.P()
	g.P("func (svr *", unexport(serverRouterType), ")UnregisterHandler(key string, handler ", serverRouterType, "Handler){")
	g.P("	svr.handlers.Delete(key)")
	g.P("}")
	g.P()
	for _, method := range service.Methods {
		nilArg := ""
		if !method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
			nilArg = "nil,"
		}
		g.P("func (svr* ", unexport(serverRouterType), ") ", serverSignature(g, method), "{")

		if !method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
			fmSymbol := helper.formatFullMethodSymbol(service, method)
			g.P("	var kv metadata.MD")
			g.P("   var ok bool")
			g.P("	if kv, ok = ", metaPackage.Ident("FromIncomingContext"), "(ctx); !ok {")
			g.P("		if kv, ok = ", metaPackage.Ident("FromOutgoingContext"), "(ctx); !ok {")
			g.P("       	return nil, ", errorsPackage.Ident("ErrServerRouterMetaNotFound"))
			g.P("		}")
			g.P("	}")
			g.P("	if keys, ok := kv[", giraPackage.Ident("GRPC_PATH_KEY"), "]; !ok {")
			g.P("       return nil, ", errorsPackage.Ident("ErrServerRouterKeyNotFound"))
			g.P("	} else if len(keys) <= 0 {")
			g.P("       return nil, ", errorsPackage.Ident("ErrServerRouterKeyNotFound"))
			g.P("	} else if v, ok := svr.handlers.Load(keys[0]); !ok {")
			g.P("       return nil, ", errorsPackage.Ident("ErrServerRouterHandlerNotRegist"))
			g.P("	} else if handler, ok := v.(", serverType, "); !ok {")
			g.P("       return nil, ", errorsPackage.Ident("ErrServerRouterHandlerNotImplement"))
			g.P("	} else {")
			g.P("		if middleware := svr.middleware; middleware == nil {")
			g.P("	        return handler.", method.GoName, "(ctx, in)")
			g.P("		} else {")
			g.P("			r := &", unexport(serverRouterType), "MiddlewareContext{")
			g.P("               fullMethod: ", fmSymbol, ",")
			g.P("               method:	    \"", method.GoName, "\",")
			g.P("				ctx:        ctx,")
			g.P("				in: in,")
			g.P("				handler: handler,")
			g.P("				invoke:    func() (resp interface{}, err error) {")
			g.P("					return handler.", method.GoName, "(ctx, in)")
			g.P("				},")
			g.P("			}")
			g.P("			if err := middleware.", serverRouterType, "MiddlewareInvoke(r); err != nil {")
			g.P("			    return nil, err")
			g.P("			} ")
			g.P("			if r.out == nil && r.err == nil {")
			g.P("			    return nil, ", errorsPackage.Ident("ErrServerRouterHandlerNotImplement"))
			g.P("			} else if r.out == nil && r.err != nil {")
			g.P("			    return nil, r.err")
			g.P("			} else {")
			g.P("			    return r.out.(*", g.QualifiedGoIdent(method.Output.GoIdent), "), r.err")
			g.P("			}")
			g.P("       }")
			g.P("   }")
		} else {
			g.P("return ", nilArg, statusPackage.Ident("Errorf"), "(", codesPackage.Ident("Unimplemented"), `, "method `, method.GoName, ` not implemented")`)
		}
		g.P("}")
	}
	g.P()
}

func (serviceGenerateHelper) generateServerFunctions(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service, serverType string, serviceDescVar string) {
	// Server handler implementations.
	handlerNames := make([]string, 0, len(service.Methods))
	for _, method := range service.Methods {
		hname := genServerMethod(gen, file, g, method, func(hname string) string {
			return hname
		})
		handlerNames = append(handlerNames, hname)
	}
	genServiceDesc(file, g, serviceDescVar, serverType, service, handlerNames)
}

func (serviceGenerateHelper) formatHandlerFuncName(service *protogen.Service, hname string) string {
	return hname
}

var helper serviceGenerateHelperInterface = serviceGenerateHelper{}

// FileDescriptorProto.package field number
const fileDescriptorProtoPackageFieldNumber = 2

// FileDescriptorProto.syntax field number
const fileDescriptorProtoSyntaxFieldNumber = 12

// generateFile generates a _gserver.pb.go file containing gRPC service definitions.
func generateFile(gen *protogen.Plugin, file *protogen.File) *protogen.GeneratedFile {
	if len(file.Services) == 0 {
		return nil
	}
	filename := file.GeneratedFilenamePrefix + "_gserver.pb.go"
	g := gen.NewGeneratedFile(filename, file.GoImportPath)
	// Attach all comments associated with the syntax field.
	genLeadingComments(g, file.Desc.SourceLocations().ByPath(protoreflect.SourcePath{fileDescriptorProtoSyntaxFieldNumber}))
	g.P("// Code generated by protoc-gen-go-gserver. DO NOT EDIT.")
	g.P("// versions:")
	g.P("// - protoc-gen-go-gserver v", version)
	g.P("// - protoc             ", protocVersion(gen))
	if file.Proto.GetOptions().GetDeprecated() {
		g.P("// ", file.Desc.Path(), " is a deprecated file.")
	} else {
		g.P("// source: ", file.Desc.Path())
	}
	g.P()
	// Attach all comments associated with the package field.
	genLeadingComments(g, file.Desc.SourceLocations().ByPath(protoreflect.SourcePath{fileDescriptorProtoPackageFieldNumber}))
	g.P("package ", file.GoPackageName)
	g.P()
	generateFileContent(gen, file, g)
	return g
}

func protocVersion(gen *protogen.Plugin) string {
	v := gen.Request.GetCompilerVersion()
	if v == nil {
		return "(unknown)"
	}
	var suffix string
	if s := v.GetSuffix(); s != "" {
		suffix = "-" + s
	}
	return fmt.Sprintf("v%d.%d.%d%s", v.GetMajor(), v.GetMinor(), v.GetPatch(), suffix)
}

// generateFileContent generates the gRPC service definitions, excluding the package statement.
func generateFileContent(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile) {
	if len(file.Services) == 0 {
		return
	}

	g.P("// This is a compile-time assertion to ensure that this generated file")
	g.P("// is compatible with the grpc package it is being compiled against.")
	g.P("// Requires gRPC-Go v1.32.0 or later.")
	g.P("const _ = ", grpcPackage.Ident("SupportPackageIsVersion7")) // When changing, update version number above.
	g.P()

	for _, service := range file.Services {
		genService(gen, file, g, service)
	}
}

func genService(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service) {

	// Server interface.
	serverType := service.GoName + "ServerRouter"
	// Server Unimplemented struct for forward compatibility.
	helper.generateServerRouterHandler(gen, file, g, service)
	helper.generateServerRouterMiddleware(gen, file, g, service)
	helper.generateServerRouterMiddlewareContext(gen, file, g, service)
	helper.generateServerRouterInterface(gen, file, g, service)
	helper.generateServerRouterType(gen, file, g, service)

	// Server registration.
	if service.Desc.Options().(*descriptorpb.ServiceOptions).GetDeprecated() {
		g.P(deprecationComment)
	}
	serviceDescVar := service.GoName + "_ServiceRouterDesc"
	g.P("func Register", service.GoName, "ServerAsRouter(s ", grpcPackage.Ident("ServiceRegistrar"), " ,handler ", serverType, "Handler) ", serverType, " {")
	g.P("svr := &", unexport(serverType), "{}")
	g.P("s.RegisterService(&", serviceDescVar, `, svr)`)
	g.P("return svr}")
	g.P()

	helper.generateServerFunctions(gen, file, g, service, serverType, serviceDescVar)
}

func serverSignature(g *protogen.GeneratedFile, method *protogen.Method) string {
	var reqArgs []string
	ret := "error"
	if !method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
		reqArgs = append(reqArgs, "ctx "+g.QualifiedGoIdent(contextPackage.Ident("Context")))
		ret = "(*" + g.QualifiedGoIdent(method.Output.GoIdent) + ", error)"
	}
	if !method.Desc.IsStreamingClient() {
		reqArgs = append(reqArgs, "in *"+g.QualifiedGoIdent(method.Input.GoIdent))
	}
	if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
		reqArgs = append(reqArgs, "s "+method.Parent.GoName+"_"+method.GoName+"Server")
	}
	return method.GoName + "(" + strings.Join(reqArgs, ", ") + ") " + ret
}

func genServiceDesc(file *protogen.File, g *protogen.GeneratedFile, serviceDescVar string, serverType string, service *protogen.Service, handlerNames []string) {
	// Service descriptor.
	g.P("// ", serviceDescVar, " is the ", grpcPackage.Ident("ServiceDesc"), " for ", service.GoName, " service.")
	g.P("// It's only intended for direct use with ", grpcPackage.Ident("RegisterService"), ",")
	g.P("// and not to be introspected or modified (even as a copy)")
	g.P("var ", serviceDescVar, " = ", grpcPackage.Ident("ServiceDesc"), " {")
	g.P("ServiceName: ", strconv.Quote(string(service.Desc.FullName())), ",")
	g.P("HandlerType: (*", serverType, ")(nil),")
	g.P("Methods: []", grpcPackage.Ident("MethodDesc"), "{")
	for i, method := range service.Methods {
		if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
			continue
		}
		g.P("{")
		g.P("MethodName: ", strconv.Quote(string(method.Desc.Name())), ",")
		g.P("Handler: ", handlerNames[i], ",")
		g.P("},")
	}
	g.P("},")
	g.P("Streams: []", grpcPackage.Ident("StreamDesc"), "{")
	for i, method := range service.Methods {
		if !method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
			continue
		}
		g.P("{")
		g.P("StreamName: ", strconv.Quote(string(method.Desc.Name())), ",")
		g.P("Handler: ", handlerNames[i], ",")
		if method.Desc.IsStreamingServer() {
			g.P("ServerStreams: true,")
		}
		if method.Desc.IsStreamingClient() {
			g.P("ClientStreams: true,")
		}
		g.P("},")
	}
	g.P("},")
	g.P("Metadata: \"", file.Desc.Path(), "\",")
	g.P("}")
	g.P()
}

func genServerMethod(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, method *protogen.Method, hnameFuncNameFormatter func(string) string) string {
	service := method.Parent
	hname := fmt.Sprintf("_%s_%s_RouterHandler", service.GoName, method.GoName)

	if !method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
		g.P("func ", hnameFuncNameFormatter(hname), "(srv interface{}, ctx ", contextPackage.Ident("Context"), ", dec func(interface{}) error, interceptor ", grpcPackage.Ident("UnaryServerInterceptor"), ") (interface{}, error) {")
		g.P("in := new(", method.Input.GoIdent, ")")
		g.P("if err := dec(in); err != nil { return nil, err }")
		g.P("if interceptor == nil { return srv.(", service.GoName, "Server).", method.GoName, "(ctx, in) }")
		g.P("info := &", grpcPackage.Ident("UnaryServerInfo"), "{")
		g.P("Server: srv,")
		fmSymbol := helper.formatFullMethodSymbol(service, method)
		g.P("FullMethod: ", fmSymbol, ",")
		g.P("}")
		g.P("handler := func(ctx ", contextPackage.Ident("Context"), ", req interface{}) (interface{}, error) {")
		g.P("return srv.(", service.GoName, "Server).", method.GoName, "(ctx, req.(*", method.Input.GoIdent, "))")
		g.P("}")
		g.P("return interceptor(ctx, in, info, handler)")
		g.P("}")
		g.P()
		return hname
	}
	streamType := unexport(service.GoName) + method.GoName + "ServerRouter"
	g.P("func ", hnameFuncNameFormatter(hname), "(srv interface{}, stream ", grpcPackage.Ident("ServerStream"), ") error {")
	if !method.Desc.IsStreamingClient() {
		g.P("m := new(", method.Input.GoIdent, ")")
		g.P("if err := stream.RecvMsg(m); err != nil { return err }")
		g.P("return srv.(", service.GoName, "Server).", method.GoName, "(m, &", streamType, "{stream})")
	} else {
		g.P("return srv.(", service.GoName, "Server).", method.GoName, "(&", streamType, "{stream})")
	}
	g.P("}")
	g.P()

	genSend := method.Desc.IsStreamingServer()
	genSendAndClose := !method.Desc.IsStreamingServer()
	genRecv := method.Desc.IsStreamingClient()

	// Stream auxiliary types and methods.
	g.P("type ", service.GoName, "_", method.GoName, "ServerRouter interface {")
	if genSend {
		g.P("Send(*", method.Output.GoIdent, ") error")
	}
	if genSendAndClose {
		g.P("SendAndClose(*", method.Output.GoIdent, ") error")
	}
	if genRecv {
		g.P("Recv() (*", method.Input.GoIdent, ", error)")
	}
	g.P(grpcPackage.Ident("ServerStream"))
	g.P("}")
	g.P()

	g.P("type ", streamType, " struct {")
	g.P(grpcPackage.Ident("ServerStream"))
	g.P("}")
	g.P()

	if genSend {
		g.P("func (x *", streamType, ") Send(m *", method.Output.GoIdent, ") error {")
		g.P("return x.ServerStream.SendMsg(m)")
		g.P("}")
		g.P()
	}
	if genSendAndClose {
		g.P("func (x *", streamType, ") SendAndClose(m *", method.Output.GoIdent, ") error {")
		g.P("return x.ServerStream.SendMsg(m)")
		g.P("}")
		g.P()
	}
	if genRecv {
		g.P("func (x *", streamType, ") Recv() (*", method.Input.GoIdent, ", error) {")
		g.P("m := new(", method.Input.GoIdent, ")")
		g.P("if err := x.ServerStream.RecvMsg(m); err != nil { return nil, err }")
		g.P("return m, nil")
		g.P("}")
		g.P()
	}

	return hname
}

func genLeadingComments(g *protogen.GeneratedFile, loc protoreflect.SourceLocation) {
	for _, s := range loc.LeadingDetachedComments {
		g.P(protogen.Comments(s))
		g.P()
	}
	if s := loc.LeadingComments; s != "" {
		g.P(protogen.Comments(s))
		g.P()
	}
}

const deprecationComment = "// Deprecated: Do not use."

func unexport(s string) string { return strings.ToLower(s[:1]) + s[1:] }
