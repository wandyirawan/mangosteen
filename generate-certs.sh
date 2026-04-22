#!/bin/bash
# Generate RSA keys for JWT signing

mkdir -p certs

echo "Generating RSA key pair..."
openssl genrsa -out certs/private.pem 2048
openssl rsa -in certs/private.pem -pubout -out certs/public.pem

echo ""
echo "Keys generated:"
echo "  - certs/private.pem (keep secret!)"
echo "  - certs/public.pem"
echo ""
echo "Add to .env:"
echo "JWT_PRIVATE_KEY=\$(awk 'NF {sub(/\r/, ""); printf "%s\\n",\$0}' certs/private.pem)"
echo "JWT_PUBLIC_KEY=\$(awk 'NF {sub(/\r/, ""); printf "%s\\n",\$0}' certs/public.pem)"