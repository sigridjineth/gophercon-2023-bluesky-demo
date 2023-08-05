# zip-default:
  #    GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o gophercon-demo-connect ./lambda/connect.go
  #    zip gophercon-demo-connect.zip gophercon-demo-connect

zip-connect:
	GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o gophercon-demo-connect ./lambda/connect.go
	zip gophercon-demo-connect.zip gophercon-demo-connect

zip-default:
	GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o gophercon-demo-default ./lambda/default.go
	zip gophercon-demo-default.zip gophercon-demo-default

zip-disconnect:
	GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o gophercon-demo-disconnect ./lambda/disconnect.go
	zip gophercon-demo-disconnect.zip gophercon-demo-disconnect

deploy:
	# login to aws
	aws lambda create-function --function-name myFunction \
    --runtime provided.al2 --handler bootstrap \
    --architectures arm64 \
    --role arn:aws:iam::111122223333:role/lambda-ex \
    --zip-file fileb://myFunction.zip