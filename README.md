# 4bit.api
The 4bit project is a personal api meant to be invoked by various trusted local
network devices for monitoring and reporting their status.

# Usage
## Certificates
For getting started, certificates can be generated using the
[generate_certs.sh](scripts/generate_certs.sh) script, which will generate a
Certificate Authority(CA), one client, and one server certificate. The latter
two will be signed by the CA. This script uses [certstrap](https://github.com/square/certstrap)
to help generate those certificates.

The following is an example for invoking the script,
```sh
scripts/generate_certs.sh \
  --ca 4bitCA \         # Arbitrary Certificate Authority name.
  --server localhost \  # Arbitrary Server name.
  --client client1      # Arbitrary Client name.
```

## Server
There are a couple of ways to start running the server binary. Within a
docker container and without.

### Using Docker
There's already a [docker-compose.yaml](docker-compose.yaml) file, which
includes a template for the basic environment variables. The environment
variable names depend on what the generated certificate names were set to.

After modifying the environment variables to to the appropriate values, running
the following start a server listening on localhost:3000,
```sh
docker-compose up
...
4bit_api_server  | 2022/05/29 18:58:10 Listening on 0.0.0.0:3000.
```

### On Host Machine
Running the server on a host machine is pretty simple. After generating the
required certificates, build the server binary by running the [build.sh](scripts/build.sh)
script. This will build the binary under the `build` directory.

After the latter two are complete, run the following with respect certificate
names chosen and the binary name built,
```sh
build/SERVER_BIN_NAME \
  server \
  --caCrtDir certs/cas \
  --caCrl certs/4bitCA.crl \
  --srvCrt certs/localhost.crt \
  --srvKey certs/localhost.key \
  --postgres_host localhost \
  --postgres_port 5432 \
  --host 0.0.0.0
```