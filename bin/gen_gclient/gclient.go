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
	errorsPackage  = protogen.GoImportPath("github.com/Lyndon-Zhang/gira/errors")
	facadePackage  = protogen.GoImportPath("github.com/Lyndon-Zhang/gira/facade")
	optionsPackage = protogen.GoImportPath("github.com/Lyndon-Zhang/gira/options/service_options")
	metaPackage    = protogen.GoImportPath("google.golang.org/grpc/metadata")
	timePackage    = protogen.GoImportPath("time")
)

var pbPackage protogen.GoImportPath

func PbPackageIdent(s string) interface{} {
	if pbPackage == "" {
		return s
	} else {
		return pbPackage.Ident(s)
	}
}

type serviceGenerateHelperInterface interface {
	formatFullMethodSymbol(service *protogen.Service, method *protogen.Method) string
	genServiceName(g *protogen.GeneratedFile, service *protogen.Service)
	genFullMethods(g *protogen.GeneratedFile, service *protogen.Service)
	generateClientsStruct(g *protogen.GeneratedFile, clientsName string, clientName string)
	generateClientsUnicastStruct(g *protogen.GeneratedFile, clientsUnicastName string, clientsName string)
	generateClientsMulticastStruct(g *protogen.GeneratedFile, clientsMulticastName string, clientsName string)
	generateNewClientsDefinitions(g *protogen.GeneratedFile, service *protogen.Service, clientName string)
	generateUnimplementedServerType(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service)
	generateServerFunctions(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service, serverType string, serviceDescVar string)
	formatHandlerFuncName(service *protogen.Service, hname string) string
}

type serviceGenerateHelper struct{}

func (serviceGenerateHelper) formatFullMethodSymbol(service *protogen.Service, method *protogen.Method) string {
	return fmt.Sprintf("%s_%s_FullMethodName", service.GoName, method.GoName)
}

func (serviceGenerateHelper) genServiceName(g *protogen.GeneratedFile, service *protogen.Service) {
	g.P("const (")
	g.P(service.GoName, `ServerName = "`, service.Desc.FullName(), `"`)
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

func (serviceGenerateHelper) generateClientsStruct(g *protogen.GeneratedFile, clientsName string, clientName string) {
	g.P("type ", unexport(clientsName), " struct {")
	// g.P("cc ", grpcPackage.Ident("ClientConnInterface"))
	g.P("mu  ", syncPackage.Ident("Mutex"))
	g.P("clientPool  map[string]*sync.Pool")
	g.P("serviceName string")
	g.P("}")
	g.P()
}

func (serviceGenerateHelper) generateClientsUnicastStruct(g *protogen.GeneratedFile, clientsUnicastName string, clientsName string) {
	g.P("type ", unexport(clientsUnicastName), " struct {")
	// g.P("cc ", grpcPackage.Ident("ClientConnInterface"))
	g.P("timeout 		int64")
	g.P("peer 			*gira.Peer")
	g.P("peerFullName   string")
	g.P("serviceName 	string")
	g.P("address 		string")
	g.P("userId 		string")
	g.P("local			bool")
	g.P("client 		*" + unexport(clientsName))
	g.P("headers		", metaPackage.Ident("MD"))
	g.P("}")
	g.P()
}

func (serviceGenerateHelper) generateClientsMulticastStruct(g *protogen.GeneratedFile, clientsMulticastName string, clientsName string) {
	g.P("type ", unexport(clientsMulticastName), " struct {")
	// g.P("cc ", grpcPackage.Ident("ClientConnInterface"))
	g.P("timeout 		int64")
	g.P("count 			int")
	g.P("serviceName 	string")
	g.P("regex 			string")
	g.P("prefix			bool")
	g.P("local			bool")
	g.P("client 		*" + unexport(clientsName))
	g.P("headers		", metaPackage.Ident("MD"))
	g.P("}")
	g.P()
}

func (serviceGenerateHelper) generateNewClientsDefinitions(g *protogen.GeneratedFile, service *protogen.Service, clientName string) {
	g.P("return &", unexport(clientName), "{")
	g.P("	serviceName: ", service.GoName, "ServerName,")
	g.P("	clientPool:  make(map[string]*sync.Pool, 0),")
	g.P("}")
}

func (serviceGenerateHelper) generateUnimplementedServerType(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service) {
	serverType := service.GoName + "Server"
	mustOrShould := "must"
	if !*requireUnimplemented {
		mustOrShould = "should"
	}
	// Server Unimplemented struct for forward compatibility.
	g.P("// Unimplemented", serverType, " ", mustOrShould, " be embedded to have forward compatible implementations.")
	g.P("type Unimplemented", serverType, " struct {")
	g.P("}")
	g.P()
	for _, method := range service.Methods {
		nilArg := ""
		if !method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
			nilArg = "nil,"
		}
		g.P("func (Unimplemented", serverType, ") ", serverSignature(g, method), "{")
		g.P("return ", nilArg, statusPackage.Ident("Errorf"), "(", codesPackage.Ident("Unimplemented"), `, "method `, method.GoName, ` not implemented")`)
		g.P("}")
	}
	if *requireUnimplemented {
		g.P("func (Unimplemented", serverType, ") mustEmbedUnimplemented", serverType, "() {}")
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

// generateFile generates a _gclient.pb.go file containing gRPC service definitions.
func generateFile(gen *protogen.Plugin, file *protogen.File, opts Options) *protogen.GeneratedFile {
	if len(file.Services) == 0 {
		return nil
	}
	filename := file.GeneratedFilenamePrefix + "_gclient.pb.go"

	g := gen.NewGeneratedFile(filename, file.GoImportPath)
	// Attach all comments associated with the syntax field.
	genLeadingComments(g, file.Desc.SourceLocations().ByPath(protoreflect.SourcePath{fileDescriptorProtoSyntaxFieldNumber}))
	g.P("// Code generated by protoc-gen-go-gclient. DO NOT EDIT.")
	g.P("// versions:")
	g.P("// - protoc-gen-go-gclient v", version)
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
	pbPackage = protogen.GoImportPath(*opts.ProtoAndGrpcPackage)
	if *opts.ProtoAndGrpcPackage != "" {
		for _, service := range file.Services {
			for _, method := range service.Methods {
				method.Output.GoIdent.GoImportPath = pbPackage
				method.Input.GoIdent.GoImportPath = pbPackage
			}
		}
	}
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

	// gen response multicast result
	responseDict := make(map[string]*protogen.Message)
	responseArr := make([]*protogen.Message, 0)
	for _, service := range file.Services {
		for _, method := range service.Methods {
			name := g.QualifiedGoIdent(method.Output.GoIdent)
			if _, ok := responseDict[name]; !ok {
				responseDict[name] = method.Output
				responseArr = append(responseArr, method.Output)
			}
		}
	}
	for _, response := range responseArr {
		structName := fmt.Sprintf("%s_MulticastResult", response.GoIdent.GoName)
		g.P("type ", structName, " struct {")
		g.P("	errors 			[]error")
		g.P("	peerCount 		int")
		g.P("	successPeers	[]*gira.Peer")
		g.P("	errorPeers		[]*gira.Peer")
		g.P("	responses		[]*", g.QualifiedGoIdent(response.GoIdent))
		g.P("}")

		g.P("func (r *", structName, ") Error() error {")
		g.P("	if len(r.errors) <= 0 {return nil}")
		g.P("	return r.errors[0]")
		g.P("}")
		g.P("func (r *", structName, ") Response(index int) *", g.QualifiedGoIdent(response.GoIdent), " {")
		g.P("	if index < 0 || index >= len(r.responses) {return nil}")
		g.P("	return r.responses[index]")
		g.P("}")
		g.P("func (r *", structName, ") SuccessPeer(index int) *gira.Peer {")
		g.P("	if index < 0 || index >= len(r.successPeers) {return nil}")
		g.P("	return r.successPeers[index]")
		g.P("}")
		g.P("func (r *", structName, ") ErrorPeer(index int) *gira.Peer {")
		g.P("	if index < 0 || index >= len(r.errorPeers) {return nil}")
		g.P("	return r.errorPeers[index]")
		g.P("}")
		g.P("func (r *", structName, ") PeerCount() int {")
		g.P("	return r.peerCount")
		g.P("}")
		g.P("func (r *", structName, ") SuccessCount() int {")
		g.P("	return len(r.successPeers)")
		g.P("}")
		g.P("func (r *", structName, ") ErrorCount() int {")
		g.P("	return len(r.errorPeers)")
		g.P("}")
		g.P("func (r *", structName, ") Errors(index int) error {")
		g.P("	if index < 0 || index >= len(r.errors) {return nil}")
		g.P("	return r.errors[index]")
		g.P("}")
	}
	for _, service := range file.Services {
		genService(gen, file, g, service)
	}
}

func genService(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service) {
	// Full methods constants.
	// helper.genFullMethods(g, service)
	helper.genServiceName(g, service)

	// Client interface.
	clientName := service.GoName + "Client"
	clientsName := service.GoName + "Clients"
	clientsMulticastName := service.GoName + "ClientsMulticast"
	clientsUnicastName := service.GoName + "ClientsUnicast"

	g.P("// ", clientName, " is the client API for ", service.GoName, " service.")
	g.P("//")
	g.P("// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.")

	if service.Desc.Options().(*descriptorpb.ServiceOptions).GetDeprecated() {
		g.P("//")
		g.P(deprecationComment)
	}
	g.Annotate(clientsName, service.Location)
	// clients interface
	g.P("type ", clientsName, " interface {")
	g.P("    WithServiceName(serviceName string) " + clientsName)
	g.P("	 // 发送到一个节点")
	g.P("    Unicast() " + clientsUnicastName)
	g.P("    Multicast(count int) " + clientsMulticastName)
	g.P("	 // 广播给所有节点")
	g.P("    Broadcast() " + clientsMulticastName)
	g.P()
	for _, method := range service.Methods {
		g.Annotate(clientsName+"."+method.GoName, method.Location)
		if method.Desc.Options().(*descriptorpb.MethodOptions).GetDeprecated() {
			g.P(deprecationComment)
		}
		g.P(method.Comments.Leading,
			clientsSignature(g, method))
	}
	g.P("}")
	g.P()

	// clients multicast interface
	g.P("type ", clientsMulticastName, " interface {")
	g.P("    WhereRegex(regex string) " + clientsMulticastName)
	g.P("    WherePrefix(prefix bool) " + clientsMulticastName)
	g.P("    Local() " + clientsMulticastName)
	for _, method := range service.Methods {
		g.Annotate(clientsMulticastName+"."+method.GoName, method.Location)
		if method.Desc.Options().(*descriptorpb.MethodOptions).GetDeprecated() {
			g.P(deprecationComment)
		}
		g.P(method.Comments.Leading,
			clientsMulticastSignature(g, method))
	}
	g.P("}")
	g.P()

	// clients unicast interface
	g.P("type ", clientsUnicastName, " interface {")
	g.P("    Where(serviceName string) " + clientsUnicastName)
	g.P("    WherePeer(peer *gira.Peer) " + clientsUnicastName)
	g.P("    WherePeerFullName(appFullName string) " + clientsUnicastName)
	g.P("    WhereAddress(address string) " + clientsUnicastName)
	g.P("    WhereUser(userId string) " + clientsUnicastName)
	g.P("    Local() " + clientsUnicastName)
	g.P()
	for _, method := range service.Methods {
		g.Annotate(clientsUnicastName+"."+method.GoName, method.Location)
		if method.Desc.Options().(*descriptorpb.MethodOptions).GetDeprecated() {
			g.P(deprecationComment)
		}
		g.P(method.Comments.Leading,
			clientsUnicastSignature(g, method))
	}
	g.P("}")
	g.P()

	// Clients structure.
	helper.generateClientsStruct(g, clientsName, clientName)

	// NewClients factory.
	if service.Desc.Options().(*descriptorpb.ServiceOptions).GetDeprecated() {
		g.P(deprecationComment)
	}
	g.P("func New", clientsName, " () ", clientsName, " {")
	helper.generateNewClientsDefinitions(g, service, clientsName)
	g.P("}")
	g.P()
	g.P("var Default", clientsName, " = New", clientsName, "()")

	// func getClient
	g.P("func (c *", unexport(service.GoName), "Clients) getClient(address string) (", PbPackageIdent(clientName), ", error) {")
	g.P("c.mu.Lock()")
	g.P("var pool *sync.Pool")
	g.P("var ok bool")
	g.P("if pool, ok = c.clientPool[address]; !ok {")
	g.P("	pool = &sync.Pool{")
	g.P("		New: func() any {")
	g.P("		conn, err := grpc.Dial(address, grpc.WithInsecure())")
	g.P("	if err != nil {")
	g.P("			return err")
	g.P("	}")
	g.P("	client := ", PbPackageIdent("New"+clientName), "(conn)")
	g.P("	return client")
	g.P("},")
	g.P("}")
	g.P("c.clientPool[address] = pool")
	g.P("c.mu.Unlock()")
	g.P("} else {")
	g.P("	c.mu.Unlock()")
	g.P("}")
	g.P("if v := pool.Get(); v == nil {")
	g.P("	return nil, ", errorsPackage.Ident("ErrGrpcClientPoolNil"))
	g.P("} else if err, ok := v.(error); ok {")
	g.P("	return nil, err")
	g.P("} else {")
	g.P("	return v.(", PbPackageIdent(clientName), "), nil")
	g.P("}")
	g.P("}")
	g.P()

	// func putClient
	g.P("func (c *", unexport(service.GoName), "Clients) putClient(address string, client ", PbPackageIdent(clientName), ") {")
	g.P("c.mu.Lock()")
	g.P("var pool *sync.Pool")
	g.P("var ok bool")
	g.P("if pool, ok = c.clientPool[address]; ok {")
	g.P("	pool.Put(client)")
	g.P("}")
	g.P("c.mu.Unlock()")
	g.P("}")
	g.P()

	// func Where
	g.P("func (c *", unexport(service.GoName), "Clients) WithServiceName(serviceName string) ", clientsName, " {")
	g.P("    c.serviceName = serviceName")
	g.P("    return c")
	g.P("}")
	g.P()

	// func Unicast
	g.P("func (c *", unexport(service.GoName), "Clients) Unicast() ", clientsUnicastName, " {")
	g.P("   headers := make(map[string]string)")
	g.P("	u := &" + unexport(clientsUnicastName) + "{")
	g.P("		timeout: 5,")
	g.P("       headers: ", metaPackage.Ident("New"), "(headers),")
	g.P("		client: c,")
	g.P("	}")
	g.P("	return u")
	g.P("}")
	g.P()

	// func Multicast
	g.P("func (c *", unexport(service.GoName), "Clients) Multicast(count int) ", clientsMulticastName, " {")
	g.P("   headers := make(map[string]string)")
	g.P("	u := &" + unexport(clientsMulticastName) + "{")
	g.P("		timeout: 5,")
	g.P("		count: count,")
	g.P("       headers: ", metaPackage.Ident("New"), "(headers),")
	g.P("		serviceName: ", fmtPackage.Ident("Sprintf"), "(\"%s/\", c.serviceName),")
	g.P("		client: c,")
	g.P("	}")
	g.P("	return u")
	g.P("}")
	g.P()

	// func Broadcast
	g.P("func (c *", unexport(service.GoName), "Clients) Broadcast() ", clientsMulticastName, " {")
	g.P("	u := &" + unexport(clientsMulticastName) + "{")
	g.P("		count: -1,")
	g.P("		serviceName: ", fmtPackage.Ident("Sprintf"), "(\"%s/\", c.serviceName),")
	g.P("		client: c,")
	g.P("	}")
	g.P("	return u")
	g.P("}")
	g.P()
	var methodIndex, streamIndex int
	// Client method implementations.
	for _, method := range service.Methods {
		if !method.Desc.IsStreamingServer() && !method.Desc.IsStreamingClient() {
			// Unary RPC method
			genClientsMethod(gen, file, g, method, methodIndex)
			methodIndex++
		} else {
			// Streaming RPC method
			genClientsMethod(gen, file, g, method, streamIndex)
			streamIndex++
		}
	}

	// ClientsUnicast structure.
	helper.generateClientsUnicastStruct(g, clientsUnicastName, clientsName)
	// func Local
	g.P("func (c *", unexport(service.GoName), "ClientsUnicast) Local() ", clientsUnicastName, " {")
	g.P("   c.local = true") // ", fmtPackage.Ident("Sprintf"), "(\"%s/%s\", c.client.serviceName, serviceName)")
	g.P("	return c")
	g.P("}")
	g.P()
	// func Where
	g.P("func (c *", unexport(service.GoName), "ClientsUnicast) Where(serviceName string) ", clientsUnicastName, " {")
	g.P("   c.serviceName = serviceName") // ", fmtPackage.Ident("Sprintf"), "(\"%s/%s\", c.client.serviceName, serviceName)")
	g.P("	return c")
	g.P("}")
	g.P()
	// func WherePeer
	g.P("func (c *", unexport(service.GoName), "ClientsUnicast) WherePeer(peer *gira.Peer) ", clientsUnicastName, " {")
	g.P("	c.peer = peer")
	g.P("	return c")
	g.P("}")
	g.P()
	// func WherePeerFullName
	g.P("func (c *", unexport(service.GoName), "ClientsUnicast) WherePeerFullName(peerFullName string) ", clientsUnicastName, " {")
	g.P("	c.peerFullName = peerFullName")
	g.P("	return c")
	g.P("}")
	g.P()
	// func WhereAddress
	g.P("func (c *", unexport(service.GoName), "ClientsUnicast) WhereAddress(address string) ", clientsUnicastName, " {")
	g.P("	c.address = address")
	g.P("	return c")
	g.P("}")
	g.P()
	// func WhereUser
	g.P("func (c *", unexport(service.GoName), "ClientsUnicast) WhereUser(userId string) ", clientsUnicastName, " {")
	g.P("	c.userId = userId")
	g.P("	c.headers.Set(", giraPackage.Ident("GRPC_PATH_KEY"), ", userId)")
	g.P("	return c")
	g.P("}")
	g.P()
	methodIndex = 0
	streamIndex = 0
	for _, method := range service.Methods {
		if !method.Desc.IsStreamingServer() && !method.Desc.IsStreamingClient() {
			// Unary RPC method
			genClientsUnicastMethod(gen, file, g, method, methodIndex)
			methodIndex++
		} else {
			// Streaming RPC method
			genClientsUnicastMethod(gen, file, g, method, streamIndex)
			streamIndex++
		}
	}

	// ClientsMulticast structure.
	helper.generateClientsMulticastStruct(g, clientsMulticastName, clientsName)
	// multicast result
	for _, method := range service.Methods {
		if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
			g.P("type ", method.Parent.GoName+"_"+method.GoName+"Client_MulticastResult", " struct {")
			g.P("	errors 			[]error")
			g.P("	peerCount 		int")
			g.P("	successPeers	[]*gira.Peer")
			g.P("	errorPeers		[]*gira.Peer")
			g.P("	responses		[]", method.Parent.GoName+"_"+method.GoName+"Client")
			g.P("}")
			g.P("func (r *", method.Parent.GoName+"_"+method.GoName+"Client_MulticastResult", ") Error() error {")
			g.P("	if len(r.errors) <= 0 {return nil}")
			g.P("	return r.errors[0]")
			g.P("}")
			g.P("func (r *", method.Parent.GoName+"_"+method.GoName+"Client_MulticastResult", ") Response(index int) ", method.Parent.GoName+"_"+method.GoName+"Client", " {")
			g.P("	if index < 0 || index >= len(r.responses) {return nil}")
			g.P("	return r.responses[index]")
			g.P("}")
			g.P("func (r *", method.Parent.GoName+"_"+method.GoName+"Client_MulticastResult", ") SuccessPeer(index int) *gira.Peer {")
			g.P("	if index < 0 || index >= len(r.successPeers) {return nil}")
			g.P("	return r.successPeers[index]")
			g.P("}")
			g.P("func (r *", method.Parent.GoName+"_"+method.GoName+"Client_MulticastResult", ") ErrorPeer(index int) *gira.Peer {")
			g.P("	if index < 0 || index >= len(r.errorPeers) {return nil}")
			g.P("	return r.errorPeers[index]")
			g.P("}")
			g.P("func (r *", method.Parent.GoName+"_"+method.GoName+"Client_MulticastResult", ") PeerCount() int {")
			g.P("	return r.peerCount")
			g.P("}")
			g.P("func (r *", method.Parent.GoName+"_"+method.GoName+"Client_MulticastResult", ") SuccessCount() int {")
			g.P("	return len(r.successPeers)")
			g.P("}")
			g.P("func (r *", method.Parent.GoName+"_"+method.GoName+"Client_MulticastResult", ") ErrorCount() int {")
			g.P("	return len(r.errorPeers)")
			g.P("}")
			g.P("func (r *", method.Parent.GoName+"_"+method.GoName+"Client_MulticastResult", ") Errors(index int) error {")
			g.P("	if index < 0 || index >= len(r.errors) {return nil}")
			g.P("	return r.errors[index]")
			g.P("}")
		}
	}
	// func Local
	g.P("func (c *", unexport(service.GoName), "ClientsMulticast) Local() ", clientsMulticastName, " {")
	g.P("   c.local = true")
	g.P("	return c")
	g.P("}")
	g.P()
	// func WhereRegex
	g.P("func (c *", unexport(service.GoName), "ClientsMulticast) WhereRegex(regex string) ", clientsMulticastName, " {")
	g.P("	c.regex = regex")
	g.P("	return c")
	g.P("}")
	g.P()

	// func WherePrefix
	g.P("func (c *", unexport(service.GoName), "ClientsMulticast) WherePrefix(prefix bool) ", clientsMulticastName, " {")
	g.P("	c.prefix = prefix")
	g.P("	return c")
	g.P("}")
	g.P()
	methodIndex = 0
	streamIndex = 0
	for _, method := range service.Methods {
		if !method.Desc.IsStreamingServer() && !method.Desc.IsStreamingClient() {
			// Unary RPC method
			genClientsMulticastMethod(gen, file, g, method, methodIndex)
			methodIndex++
		} else {
			// Streaming RPC method
			genClientsMulticastMethod(gen, file, g, method, streamIndex)
			streamIndex++
		}
	}
	return
	mustOrShould := "must"
	if !*requireUnimplemented {
		mustOrShould = "should"
	}

	// Server interface.
	serverType := service.GoName + "Server"
	g.P("// ", serverType, " is the server API for ", service.GoName, " service.")
	g.P("// All implementations ", mustOrShould, " embed Unimplemented", serverType)
	g.P("// for forward compatibility")
	if service.Desc.Options().(*descriptorpb.ServiceOptions).GetDeprecated() {
		g.P("//")
		g.P(deprecationComment)
	}
	g.Annotate(serverType, service.Location)
	g.P("type ", serverType, " interface {")
	for _, method := range service.Methods {
		g.Annotate(serverType+"."+method.GoName, method.Location)
		if method.Desc.Options().(*descriptorpb.MethodOptions).GetDeprecated() {
			g.P(deprecationComment)
		}
		g.P(method.Comments.Leading,
			serverSignature(g, method))
	}
	if *requireUnimplemented {
		g.P("mustEmbedUnimplemented", serverType, "()")
	}
	g.P("}")
	g.P()

	// Server Unimplemented struct for forward compatibility.
	helper.generateUnimplementedServerType(gen, file, g, service)

	// Unsafe Server interface to opt-out of forward compatibility.
	g.P("// Unsafe", serverType, " may be embedded to opt out of forward compatibility for this service.")
	g.P("// Use of this interface is not recommended, as added methods to ", serverType, " will")
	g.P("// result in compilation errors.")
	g.P("type Unsafe", serverType, " interface {")
	g.P("mustEmbedUnimplemented", serverType, "()")
	g.P("}")

	// Server registration.
	if service.Desc.Options().(*descriptorpb.ServiceOptions).GetDeprecated() {
		g.P(deprecationComment)
	}
	serviceDescVar := service.GoName + "_ServiceDesc"
	g.P("func Register", service.GoName, "Server(s ", grpcPackage.Ident("ServiceRegistrar"), ", srv ", serverType, ") {")
	g.P("s.RegisterService(&", serviceDescVar, `, srv)`)
	g.P("}")
	g.P()

	helper.generateServerFunctions(gen, file, g, service, serverType, serviceDescVar)
}

func clientsSignature(g *protogen.GeneratedFile, method *protogen.Method) string {
	s := method.GoName + "(ctx " + g.QualifiedGoIdent(contextPackage.Ident("Context"))
	s += ", address string"
	if !method.Desc.IsStreamingClient() {
		s += ", in *" + g.QualifiedGoIdent(method.Input.GoIdent)
	}
	s += ", opts ..." + g.QualifiedGoIdent(grpcPackage.Ident("CallOption")) + ") ("
	if !method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
		s += "*" + g.QualifiedGoIdent(method.Output.GoIdent)
	} else {
		s += method.Parent.GoName + "_" + method.GoName + "Client"
	}
	s += ", error)"
	return s
}

func clientsMulticastSignature(g *protogen.GeneratedFile, method *protogen.Method) string {
	s := method.GoName + "(ctx " + g.QualifiedGoIdent(contextPackage.Ident("Context"))
	if !method.Desc.IsStreamingClient() {
		s += ", in *" + g.QualifiedGoIdent(method.Input.GoIdent)
	}
	s += ", opts ..." + g.QualifiedGoIdent(grpcPackage.Ident("CallOption")) + ") ("
	if !method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
		s += "*" + method.Output.GoIdent.GoName + "_MulticastResult"
	} else {
		s += "*" + method.Parent.GoName + "_" + method.GoName + "Client_MulticastResult"
	}
	s += ", error)"
	return s
}

func clientsUnicastSignature(g *protogen.GeneratedFile, method *protogen.Method) string {
	s := method.GoName + "(ctx " + g.QualifiedGoIdent(contextPackage.Ident("Context"))
	if !method.Desc.IsStreamingClient() {
		s += ", in *" + g.QualifiedGoIdent(method.Input.GoIdent)
	}
	s += ", opts ..." + g.QualifiedGoIdent(grpcPackage.Ident("CallOption")) + ") ("
	if !method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
		s += "*" + g.QualifiedGoIdent(method.Output.GoIdent)
	} else {
		s += method.Parent.GoName + "_" + method.GoName + "Client"
	}
	s += ", error)"
	return s
}

func genClientsMethod(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, method *protogen.Method, index int) {
	service := method.Parent

	if method.Desc.Options().(*descriptorpb.MethodOptions).GetDeprecated() {
		g.P(deprecationComment)
	}
	g.P("func (c *", unexport(service.GoName), "Clients) ", clientsSignature(g, method), "{")
	if !method.Desc.IsStreamingServer() && !method.Desc.IsStreamingClient() {
		g.P("client, err := c.getClient(address)")
		g.P("if err != nil { return nil, err }")
		g.P("defer c.putClient(address, client)")
		g.P(`out, err := client.`, method.Desc.Name(), `(ctx, in, opts...)`)
		g.P("if err != nil { return nil, err }")
		g.P("return out, nil")
		g.P("}")
		g.P()
		return
	} else if method.Desc.IsStreamingServer() && !method.Desc.IsStreamingClient() {
		g.P("client, err := c.getClient(address)")
		g.P("if err != nil { return nil, err }")
		g.P("defer c.putClient(address, client)")
		g.P(`out, err := client.`, method.Desc.Name(), `(ctx, in, opts...)`)
		g.P("if err != nil { return nil, err }")
		g.P("return out, nil")
		g.P("}")
		g.P()
		return
	} else {
		g.P("client, err := c.getClient(address)")
		g.P("if err != nil { return nil, err }")
		g.P("defer c.putClient(address, client)")
		g.P(`out, err := client.`, method.Desc.Name(), `(ctx, opts...)`)
		g.P("if err != nil { return nil, err }")
		g.P("return out, nil")
		g.P("}")
		g.P()
		return
	}
}

func genClientsUnicastMethod(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, method *protogen.Method, index int) {
	service := method.Parent

	if method.Desc.Options().(*descriptorpb.MethodOptions).GetDeprecated() {
		g.P(deprecationComment)
	}
	g.P("func (c *", unexport(service.GoName), "ClientsUnicast) ", clientsUnicastSignature(g, method), "{")

	g.P("    if c.local {")
	if !method.Desc.IsStreamingServer() && !method.Desc.IsStreamingClient() {
		g.P("    cancelCtx, cancelFunc := ", contextPackage.Ident("WithTimeout"), "(ctx, ", timePackage.Ident("Second"), "*", timePackage.Ident("Duration"), "(c.timeout))")
		g.P("    defer cancelFunc()")
		g.P("    if c.headers.Len() > 0 {")
		g.P("        cancelCtx = metadata.NewOutgoingContext(cancelCtx, c.headers)")
		g.P("    }")
		g.P("    if s, ok := ", facadePackage.Ident("WhereIsServer"), "(c.client.serviceName); !ok {")
		g.P("	     return nil, ", errorsPackage.Ident("ErrServerNotFound"))
		g.P(" 	 } else if svr, ok := s.(", PbPackageIdent(service.GoName), "Server); !ok {")
		g.P("	     return nil, ", errorsPackage.Ident("ErrServerNotFound"))
		g.P(" 	 } else {")
		g.P(" 	     return svr.", method.GoName, "(cancelCtx, in)")
		g.P(" 	 } ")
		g.P()
	} else if method.Desc.IsStreamingServer() && !method.Desc.IsStreamingClient() {
		g.P("return nil, ", statusPackage.Ident("Errorf"), "(", codesPackage.Ident("Unimplemented"), `, "method `, method.GoName, ` not implemented")`)
		g.P()
	} else {
		g.P("return nil, ", statusPackage.Ident("Errorf"), "(", codesPackage.Ident("Unimplemented"), `, "method `, method.GoName, ` not implemented")`)
		g.P()
	}
	g.P("    } else {")
	g.P("var address string")
	g.P("if len(c.address) > 0 {")
	g.P("	address = c.address")
	g.P("} else if len(c.peerFullName) > 0 {")
	g.P("	if peer, err := ", facadePackage.Ident("WhereIsPeer"), "(c.peerFullName); err != nil {")
	g.P("       return nil, err")
	g.P("   } else if ", facadePackage.Ident("IsEnableResolver()"), " {")
	g.P("	    address = peer.Url")
	g.P("   } else {")
	g.P("	    address = peer.Address")
	g.P("   }")
	g.P("} else if c.peer != nil && ", facadePackage.Ident("IsEnableResolver()"), " {")
	g.P("	address = c.peer.Url")
	g.P("} else if c.peer != nil {")
	g.P("	address = c.peer.Address")
	g.P("} else if len(c.serviceName) > 0 {")
	g.P("	if peers, err := ", facadePackage.Ident("WhereIsServiceName"), "(c.serviceName); err != nil {")
	g.P("		return nil, err")
	g.P("	} else if len(peers) < 1 {")
	g.P("		return nil, ", errorsPackage.Ident("ErrPeerNotFound"))
	g.P("	} else if ", facadePackage.Ident("IsEnableResolver()"), " {")
	g.P("		address = peers[0].Url")
	g.P("	} else  {")
	g.P("		address = peers[0].Address")
	g.P("	}")
	g.P("} else if len(c.userId) > 0 {")
	g.P("	if peer, err := ", facadePackage.Ident("WhereIsUser"), "(c.userId); err != nil {")
	g.P("		return nil, err")
	g.P("	} else if ", facadePackage.Ident("IsEnableResolver()"), " {")
	g.P("		address = peer.Url")
	g.P("	} else {")
	g.P("		address = peer.Address")
	g.P("	}")
	g.P("}")
	g.P("if len(address) <= 0 {")
	g.P("	return nil,", errorsPackage.Ident("ErrPeerNotFound"))
	g.P("}")
	if !method.Desc.IsStreamingServer() && !method.Desc.IsStreamingClient() {
		g.P("client, err := c.client.getClient(address)")
		g.P("if err != nil { return nil, err }")
		g.P("defer c.client.putClient(address, client)")
		g.P("if c.headers.Len() > 0 {")
		g.P("    ctx = metadata.NewOutgoingContext(ctx, c.headers)")
		g.P("}")
		g.P(`out, err := client.`, method.Desc.Name(), `(ctx, in, opts...)`)
		g.P("if err != nil { return nil, err }")
		g.P("return out, nil")
		g.P("}")
		g.P()
	} else if method.Desc.IsStreamingServer() && !method.Desc.IsStreamingClient() {
		g.P("client, err := c.client.getClient(address)")
		g.P("if err != nil { return nil, err }")
		g.P("defer c.client.putClient(address, client)")
		g.P("if c.headers.Len() > 0 {")
		g.P("    ctx = metadata.NewOutgoingContext(ctx, c.headers)")
		g.P("}")
		g.P(`out, err := client.`, method.Desc.Name(), `(ctx, in, opts...)`)
		g.P("if err != nil { return nil, err }")
		g.P("return out, nil")
		g.P("}")
		g.P()
	} else {
		g.P("client, err := c.client.getClient(address)")
		g.P("if err != nil { return nil, err }")
		g.P("defer c.client.putClient(address, client)")
		g.P("if c.headers.Len() > 0 {")
		g.P("    ctx = metadata.NewOutgoingContext(ctx, c.headers)")
		g.P("}")
		g.P(`out, err := client.`, method.Desc.Name(), `(ctx, opts...)`)
		g.P("if err != nil { return nil, err }")
		g.P("return out, nil")
		g.P("}")
		g.P()
	}
	g.P("}")
}

func genClientsMulticastMethod(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, method *protogen.Method, index int) {
	service := method.Parent

	if method.Desc.Options().(*descriptorpb.MethodOptions).GetDeprecated() {
		g.P(deprecationComment)
	}
	g.P("func (c *", unexport(service.GoName), "ClientsMulticast) ", clientsMulticastSignature(g, method), "{")

	g.P("    if c.local {")

	if !method.Desc.IsStreamingServer() && !method.Desc.IsStreamingClient() {
		g.P("    if s, ok := ", facadePackage.Ident("WhereIsServer"), "(c.client.serviceName); !ok {")
		g.P("	     return nil, ", errorsPackage.Ident("ErrServerNotFound"))
		g.P(" 	 } else if svr, ok := s.(", PbPackageIdent(service.GoName), "Server); ok {")
		g.P("		result := &", method.Output.GoIdent.GoName, "_MulticastResult{}")
		g.P("		cancelCtx, cancelFunc := ", contextPackage.Ident("WithTimeout"), "(ctx, ", timePackage.Ident("Second"), "*", timePackage.Ident("Duration"), "(c.timeout))")
		g.P("		defer cancelFunc()")
		g.P("		if c.headers.Len() > 0 {")
		g.P("			cancelCtx = metadata.NewOutgoingContext(cancelCtx, c.headers)")
		g.P("		}")
		g.P("		if resp , err := svr.", method.GoName, "(cancelCtx, in); err != nil {")
		g.P("			return nil, err")
		g.P("		} else {")
		g.P("			result.responses = append(result.responses, resp)")
		g.P("		} ")
		g.P("		return result, nil")
		g.P("	 } else {")
		g.P("	     return nil, ", errorsPackage.Ident("ErrServerNotFound"))
		g.P("	 }")
	} else if method.Desc.IsStreamingServer() && !method.Desc.IsStreamingClient() {
		g.P("return nil, ", statusPackage.Ident("Errorf"), "(", codesPackage.Ident("Unimplemented"), `, "method `, method.GoName, ` not implemented")`)
		g.P()
	} else {
		g.P("return nil, ", statusPackage.Ident("Errorf"), "(", codesPackage.Ident("Unimplemented"), `, "method `, method.GoName, ` not implemented")`)
		g.P()
	}

	g.P("    } else {")

	g.P("var peers []*gira.Peer")
	g.P("var whereOpts []", optionsPackage.Ident("WhereOption"))
	g.P("// 多播")
	g.P("whereOpts = append(whereOpts, ", optionsPackage.Ident("WithWhereCatalogOption"), "())")
	g.P("if c.count > 0 {whereOpts = append(whereOpts, ", optionsPackage.Ident("WithWhereMaxCountOption"), "(c.count))}")
	g.P("serviceName := c.serviceName")
	g.P("if len(c.regex) > 0 {")
	g.P("    serviceName = fmt.Sprintf(\"%s%s\", c.serviceName, c.regex)")
	g.P("    whereOpts = append(whereOpts, ", optionsPackage.Ident("WithWhereRegexOption"), "())")
	g.P("}")
	g.P("if c.prefix {")
	g.P("    whereOpts = append(whereOpts, ", optionsPackage.Ident("WithWherePrefixOption"), "())")
	g.P("}")
	g.P("peers, err := ", facadePackage.Ident("WhereIsServiceName"), "(serviceName, whereOpts...)")
	g.P("if err != nil {")
	g.P("	return nil, err")
	g.P("}")
	if !method.Desc.IsStreamingServer() && !method.Desc.IsStreamingClient() {
		g.P("result := &", method.Output.GoIdent.GoName, "_MulticastResult{}")
	} else if method.Desc.IsStreamingServer() && !method.Desc.IsStreamingClient() {
		g.P("result := &", method.Parent.GoName, "_", method.GoName, "Client_MulticastResult{}")
	} else {
		g.P("result := &", method.Parent.GoName, "_", method.GoName, "Client_MulticastResult{}")
	}
	g.P("result.peerCount = len(peers)")
	if !method.Desc.IsStreamingServer() && !method.Desc.IsStreamingClient() {
		g.P("for _, peer := range peers {")
		g.P("	var address string")
		g.P("	if ", facadePackage.Ident("IsEnableResolver()"), "{")
		g.P("		address = peer.Url")
		g.P("	} else {")
		g.P("		address = peer.Address")
		g.P("	}")
		g.P("	client, err := c.client.getClient(address)")
		g.P("	if err != nil { ")
		g.P("		result.errors = append(result.errors, err)")
		g.P("		result.errorPeers = append(result.errorPeers, peer)")
		g.P("		continue")
		g.P("	}")
		g.P(`	out, err := client.`, method.Desc.Name(), `(ctx, in, opts...)`)
		g.P("	if err != nil { ")
		g.P("		result.errors = append(result.errors, err)")
		g.P("		result.errorPeers = append(result.errorPeers, peer)")
		g.P("		c.client.putClient(address, client)")
		g.P("		continue")
		g.P("	}")
		g.P("	c.client.putClient(address, client)")
		g.P("	result.responses = append(result.responses, out)")
		g.P("	result.successPeers = append(result.successPeers, peer)")
		g.P("}")
		g.P("return result, nil")
		g.P("}")
		g.P()
	} else if method.Desc.IsStreamingServer() && !method.Desc.IsStreamingClient() {
		g.P("for _, peer := range peers {")
		g.P("	var address string")
		g.P("	if ", facadePackage.Ident("IsEnableResolver()"), "{")
		g.P("		address = peer.Url")
		g.P("	} else {")
		g.P("		address = peer.Address")
		g.P("	}")
		g.P("	client, err := c.client.getClient(address)")
		g.P("	if err != nil { ")
		g.P("		result.errors = append(result.errors, err)")
		g.P("		result.errorPeers = append(result.errorPeers, peer)")
		g.P("		continue")
		g.P("	}")
		g.P(`	out, err := client.`, method.Desc.Name(), `(ctx, in, opts...)`)
		g.P("	if err != nil { ")
		g.P("		result.errors = append(result.errors, err)")
		g.P("		result.errorPeers = append(result.errorPeers, peer)")
		g.P("		c.client.putClient(address, client)")
		g.P("		continue")
		g.P("	}")
		g.P("	result.responses = append(result.responses, out)")
		g.P("	result.successPeers = append(result.successPeers, peer)")
		g.P("}")
		g.P("return result, nil")
		g.P("}")
		g.P()
	} else {
		g.P("for _, peer := range peers {")
		g.P("	var address string")
		g.P("	if ", facadePackage.Ident("IsEnableResolver()"), "{")
		g.P("		address = peer.Url")
		g.P("	} else {")
		g.P("		address = peer.Address")
		g.P("	}")
		g.P("	client, err := c.client.getClient(address)")
		g.P("	if err != nil { ")
		g.P("		result.errors = append(result.errors, err)")
		g.P("		result.errorPeers = append(result.errorPeers, peer)")
		g.P("		continue")
		g.P("	}")
		g.P(`	out, err := client.`, method.Desc.Name(), `(ctx, opts...)`)
		g.P("	if err != nil { ")
		g.P("		result.errors = append(result.errors, err)")
		g.P("		result.errorPeers = append(result.errorPeers, peer)")
		g.P("		c.client.putClient(address, client)")
		g.P("		continue")
		g.P("	}")
		g.P("	result.responses = append(result.responses, out)")
		g.P("	result.successPeers = append(result.successPeers, peer)")
		g.P("}")
		g.P("return result, nil")
		g.P("}")
		g.P()
	}
	g.P("}")
}

func serverSignature(g *protogen.GeneratedFile, method *protogen.Method) string {
	var reqArgs []string
	ret := "error"
	if !method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
		reqArgs = append(reqArgs, g.QualifiedGoIdent(contextPackage.Ident("Context")))
		ret = "(*" + g.QualifiedGoIdent(method.Output.GoIdent) + ", error)"
	}
	if !method.Desc.IsStreamingClient() {
		reqArgs = append(reqArgs, "*"+g.QualifiedGoIdent(method.Input.GoIdent))
	}
	if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
		reqArgs = append(reqArgs, method.Parent.GoName+"_"+method.GoName+"Server")
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
	hname := fmt.Sprintf("_%s_%s_Handler", service.GoName, method.GoName)

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
	streamType := unexport(service.GoName) + method.GoName + "Server"
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
	g.P("type ", service.GoName, "_", method.GoName, "Server interface {")
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
