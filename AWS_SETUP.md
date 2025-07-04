# AWS S3 Setup Guide

## Prerequisites

1. AWS Account
2. S3 Bucket created
3. IAM User with S3 permissions

## Step 1: Create S3 Bucket

1. Go to AWS Console → S3
2. Click "Create bucket"
3. Enter bucket name (e.g., `telegram-media-bucket-yourname`)
4. Select region (e.g., `us-east-1`)
5. Keep other settings as default
6. Click "Create bucket"

## Step 2: Create IAM User

1. Go to AWS Console → IAM → Users
2. Click "Add users"
3. Enter username (e.g., `telegram-grabber`)
4. Select "Access key - Programmatic access"
5. Click "Next: Permissions"

## Step 3: Set Permissions

1. Click "Attach existing policies directly"
2. Search for and select `AmazonS3FullAccess`
   
   OR create a custom policy:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": [
           "s3:GetObject",
           "s3:PutObject",
           "s3:DeleteObject",
           "s3:ListBucket"
         ],
         "Resource": [
           "arn:aws:s3:::your-bucket-name/*",
           "arn:aws:s3:::your-bucket-name"
         ]
       }
     ]
   }
   ```

## Step 4: Get Credentials

1. Complete user creation
2. Save the Access Key ID and Secret Access Key
3. **IMPORTANT**: You won't be able to see the secret key again!

## Step 5: Configure the Application

1. Copy `.env.example` to `.env`:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` with your real AWS credentials:
   ```env
   AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
   AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
   S3_BUCKET_NAME=telegram-media-bucket-yourname
   AWS_REGION=us-east-1
   ```

## Step 6: Run with Real S3

```bash
# Use the real S3 docker-compose file
docker-compose -f docker-compose-real-s3.yml up --build
```

## Troubleshooting

### Invalid Access Key Error
- Double-check your credentials in `.env`
- Ensure no extra spaces or quotes
- Try generating new access keys in AWS IAM

### Access Denied Error
- Check IAM user permissions
- Ensure bucket name is correct
- Verify region is correct

### Bucket Not Found
- Ensure bucket exists
- Check bucket name spelling
- Verify region matches

## Cost Considerations

Using real S3 will incur costs:
- Storage: ~$0.023 per GB per month
- Requests: ~$0.0004 per 1,000 requests
- Data transfer: ~$0.09 per GB (outbound)

## Security Best Practices

1. **Never commit `.env` file** to git
2. Use IAM roles in production (not access keys)
3. Enable S3 bucket versioning
4. Set up lifecycle policies to delete old files
5. Enable CloudTrail for audit logging
6. Use bucket policies to restrict access