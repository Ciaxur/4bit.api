#!/usr/bin/env sh


# This script servers as an entrypoint for a docker container.
# Lets get all of the variables sorted out.
SCRIPT_REL_DIR=$(dirname $0)
SCRIPT_DIR=$(realpath $SCRIPT_REL_DIR)
PROJ_ROOT=$(realpath $SCRIPT_REL_DIR/..)
BUILD_DIR=$PROJ_ROOT/build
CERTS_DIR=$PROJ_ROOT/certs


# Obtain the CA & server name name from the env, removing the extention if
# present.
# Default to a cert name if not present.
CA_CRT_NAME=${CA_CRT_NAME:-4bitCA}
CA_CRT_NAME=${CA_CRT_NAME%.*}
SERVER_CRT_NAME=${SERVER_CRT_NAME:-localhost}
SERVER_CRT_NAME=${SERVER_CRT_NAME%.*}

# Install required alpine packages.
apk add git

# First, we would build the server from a clean slate.
echo "Clearing out previous build(s)."
cd $PROJ_ROOT
rm -rf $BUILD_DIR
sh $SCRIPT_DIR/build.sh

# Then, check if certs were generated already, if so, use those certs.
if [ ! -d "$CERTS_DIR" ] || [ `ls -1 $CERTS_DIR | wc -l | tr -d " "` == "0" ]; then
  echo "Generating server and client certificates."
  mkdir -p $CERTS_DIR

  echo "Cloning cerstrap from https://github.com/square/certstrap, which is used to aid in cert generation."
  CERTSRAP_VERSION="v1.2.0"
  git clone https://github.com/square/certstrap
  git checkout $CERTSRAP_VERSION
  cd certstrap
  go build
  mv ./certstrap /usr/bin/
  cd -

  # Generate a CA to be used for signing the server cert and other client certs.
  echo "Generating a CA certificate, $CA_CRT_NAME, without a passphrase."
  certstrap init --common-name $CA_CRT_NAME --passphrase ""

  # Generate the server key pairs (certs & keys).
  echo "Generating server key pairs, $SERVER_CRT_NAME, without a passphrase."
  certstrap request-cert --domain $SERVER_CRT_NAME --key-bits 4096 --passphrase ""

  # Sign the generated server key pair.
  certstrap sign $SERVER_CRT_NAME --CA $CA_CRT_NAME --passphrase ""
  echo "WARNING: Change the passphrase separately for extra security."

  # Notify the user that they will need to manually generate and sign a client
  # certificate using the generated CA.
  echo "NOTE: The user is responsible for manually generating and signing "
  echo "client key pairs using the generated CA, $CA_CRT_NAME."

  # Finally, clean up the cloned certstrap repository and move certs to an
  # appropriate location, since certstrap generates certs under the "out"
  # direcotory.
  echo "Cleaning up. Moving generated certificates to an appropriate location "
  echo "and removing unnecessary junk."
  mv ./out/* $CERTS_DIR/
  rm -rf ./cerstrap
  go clean -cache -modcache -i -r
fi

# Start the server!
BIN_VERSION=$(git rev-parse HEAD | head -c 8)
BIN_NAME="4bit-api-$BIN_VERSION"
$BUILD_DIR/$BIN_NAME server \
  --caCrt $CERTS_DIR/$CA_CRT_NAME.crt \
  --caCrl $CERTS_DIR/$CA_CRT_NAME.crl \
  --srvCrt $CERTS_DIR/$SERVER_CRT_NAME.crt \
  --srvKey $CERTS_DIR/$SERVER_CRT_NAME.key