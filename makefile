# 定义变量
PROTOC = protoc
PROTO_DIR = .
# 输出目录设置为.proto文件所在的目录
GO_OUT_DIR = .

# 查找所有.proto文件
PROTO_FILES = $(shell find $(PROTO_DIR) -name '*.proto')

# 生成的Go文件
GO_FILES = $(PROTO_FILES:.proto=.pb.go)
GRPC_GO_FILES = $(PROTO_FILES:.proto=_grpc.pb.go)

build:
	mkdir -p bin
	go build -o bin ./cmd/...

# 编译pb文件生成桩代码
pb: $(GO_FILES) $(GRPC_GO_FILES)

# 规则：如何从.proto文件生成.pb.go文件
%.pb.go: %.proto
	$(PROTOC) --proto_path=$(PROTO_DIR) --go_out=$(GO_OUT_DIR) --go_opt=paths=source_relative $<

# 规则：如何从.proto文件生成_grpc.pb.go文件
%_grpc.pb.go: %.proto
	$(PROTOC) --proto_path=$(PROTO_DIR) --go-grpc_out=$(GO_OUT_DIR) --go-grpc_opt=paths=source_relative $<

# 清理生成的文件
clean:
	rm -f $(GO_FILES) $(GRPC_GO_FILES)
	rm -f certs/*

# 生成临时使用的自签名证书
cert:
	mkdir -p certs
	openssl genrsa -out certs/server.key 2048
	openssl req -new -x509 -days 3650 -key certs/server.key -out certs/server.crt -subj "/C=CN/ST=Guangdong/L=Shenzhen/O=toolkits/OU=toolkits/CN=*.toolkits.win" -addext "subjectAltName = DNS:*.toolkits.win"
