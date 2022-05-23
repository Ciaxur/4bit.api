#!/usr/bin/env bash
# This script is intended to generate a CA cert along with a server cert and
# client cert.

# Verify that certstrap is installed and readily available to be invoked.
which certstrap 2>&1 1>/dev/null
WHICH_CERTSTRAP_EXIT=$?
if [ $WHICH_CERTSTRAP_EXIT == 1 ]; then
  echo "certstrap is required to run this script."
  echo "Please install from https://github.com/square/certstrap."
  exit 1
fi

# Store the relative path to the script directory in order to create & move the
# generated files into the project root.
SCRIPTS_DIR=`dirname $0`
CERTS_DIR=$SCRIPTS_DIR/../certs

# Parse positional arguments.
POSITIONAL=()
while [[ $# -gt 0 ]]; do
  key="$1"

  case $key in
    --ca)
      CA_NAME="$2"
      shift # past argument
      shift # past value
      ;;
    --server)
      SERVER_DOMAIN="$2"
      shift # past argument
      shift # past value
      ;;
    --client)
      CLIENT_DOMAIN="$2"
      shift # past argument
      shift # past value
      ;;
    *)    # unknown option
      POSITIONAL+=("$1") # save it in an array for later
      shift # past argument
      ;;
  esac
done

# Check required flags where populated.
BORKED=0
if [ "$CA_NAME" == "" ]; then
  echo "Missing required CA name flag."
  BORKED=1
fi
if [ "$SERVER_DOMAIN" == "" ]; then
  echo "Missing required server domain name flag."
  BORKED=1
fi
if [ "$CLIENT_DOMAIN" == "" ]; then
  echo "Missing required client domain name flag."
  BORKED=1
fi
if [ "$BORKED" == "1" ]; then
  echo "
USAGE:
  generate_certs.sh [FLAGS]
EXAMPLE:
  generate_certs.sh --ca YeetusCA --server localhost --client client
"
  exit 1
fi

echo "Generating certificates and keys..."
# Using certstrap, create a CA that will "vouche" for certificates.
certstrap init --common-name $CA_NAME

# Generate the server and client key pairs (certs & keys).
# NOTE: If your server has multiple domains and/or ips (use --ip), append the
# various entires separated by commas.
certstrap request-cert --domain $SERVER_DOMAIN --key-bits 4096
certstrap request-cert --domain $CLIENT_DOMAIN --key-bits 4096

# Have the CA sign those suckers for verification.
certstrap sign $SERVER_DOMAIN --CA $CA_NAME
certstrap sign $CLIENT_DOMAIN --CA $CA_NAME

echo "Moving generated files into the 'certs' directory."
mkdir -p $CERTS_DIR
mv ./out/* $CERTS_DIR/
rm -r ./out
