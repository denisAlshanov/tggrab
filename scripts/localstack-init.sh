#!/bin/bash

# LocalStack S3 initialization script
# This script creates the S3 bucket needed for the application

echo "Initializing LocalStack S3..."

# Wait for LocalStack to be ready
until curl -s http://localhost:4566/_localstack/health | grep -q '"s3": "available"'; do
  echo "Waiting for LocalStack S3 to be ready..."
  sleep 2
done

# Create S3 bucket
awslocal s3 mb s3://telegram-media-bucket --region us-east-1

# Set bucket policy to allow access
awslocal s3api put-bucket-policy --bucket telegram-media-bucket --policy '{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PublicReadGetObject",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::telegram-media-bucket/*"
    }
  ]
}'

echo "S3 bucket 'telegram-media-bucket' created successfully"