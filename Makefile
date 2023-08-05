zip:
	GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o gophercon-demo main.go
	zip gophercon-demo.zip gophercon-demo

deploy:
	# login to aws
	aws lambda create-function --function-name myFunction \
    --runtime provided.al2 --handler bootstrap \
    --architectures arm64 \
    --role arn:aws:iam::111122223333:role/lambda-ex \
    --zip-file fileb://myFunction.zip