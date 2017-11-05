# protoc-gen-go
A protoc-gen-go clone registering a "serialized API" plugin.

## Requirements

`protoc` should be installed to be able to compile `.proto` files

## Installation

`go get -u github.com/lleveque/protoc-gen-go` should be enough.

## Usage

To compile your model and use the grpcserial plugin, run the following command :
```
protoc --go_out=plugins=grpcserial:`pwd` test.proto
```

This should generate a file named `test.pb.go` containing a Go implementation of your model and a stub proposition to start implementing your API.   
The grpcserial plugin takes grpc service description (here the Greet service) and outputs some stubs including deserialization of your parameters and serialization of your return values.   
With the test.proto file, the generated stubs look like this :

```go
package your_package // TODO change to your project package name

import "github.com/golang/protobuf/proto"
import pb "greeting" // TODO change to the Go package in which your .pb.go has been generated

// TODO change packagePath value to match your package full import path
//go:generate goprotopy --packagePath=your_org/your_name/your_package $GOFILE

// Hello returns a greeting to a person with an age,
// and whether this person had previously been seen or not
// input is a serialized protobuf object of type HelloRequest
// output is a serialized protobuf object of type HelloResponse
// @protopy
func Hello(input []byte) (output []byte, err error) {
    helloRequest := new(pb.HelloRequest)
    err = proto.Unmarshal(input, helloRequest)
    if err != nil {
        return
    }

    // TODO : implement Hello(helloRequest *pb.HelloRequest) (*pb.HelloResponse, error)
    // helloResponse, err := yourHelloImplementation(helloRequest)

    helloResponse := new(pb.HelloResponse)
    output, err = proto.Marshal(helloResponse)
    return
}

// Goodbye returns a byebye greeting to anyone
// input is a serialized protobuf object of type GoodbyeRequest
// output is a serialized protobuf object of type GoodbyeResponse
// @protopy
func Goodbye(input []byte) (output []byte, err error) {
    goodbyeRequest := new(pb.GoodbyeRequest)
    err = proto.Unmarshal(input, goodbyeRequest)
    if err != nil {
        return
    }

    // TODO : implement Goodbye(goodbyeRequest *pb.GoodbyeRequest) (*pb.GoodbyeResponse, error)
    // goodbyeResponse, err := yourGoodbyeImplementation(goodbyeRequest)

    goodbyeResponse := new(pb.GoodbyeResponse)
    output, err = proto.Marshal(goodbyeResponse)
    return
}
```

## Going further

The stubs are annotated with the `@protopy` comment, that enables the straightforward use of the [goprotopy](https://github.com/lleveque/goprotopy) sister tool to generate Python bindings for your serialized API.
