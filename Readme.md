# Go Load Balancer

This is a simple Go load balancer that distributes incoming HTTP requests across multiple backend servers. It uses a round-robin algorithm to select the next server to handle each request.

## Usage

To use the load balancer, you need to provide a list of backend servers as a comma-separated list of URLs. For example:

```bash
./bin/lb.exe --servers=http://localhost:3031,http://localhost:3032,http://localhost:3033,http://localhost:3034
```

You can also specify a port for the load balancer to listen on:

```bash
./bin/lb.exe --servers=http://localhost:3031,http://localhost:3032,http://localhost:3033,http://localhost:3034 --port=3000
```

## Building

To build the load balancer, you need to have Go installed on your system. Then, you can use the following command:

```bash
make build
```

This will compile the load balancer and create an executable file named `lb.exe` in the `bin` directory.

## Running

To run the load balancer, you can use the following command:

```bash
make run --servers=http://localhost:3031,http://localhost:3032,http://localhost:3033,http://localhost:3034 --port=3000
```

This will start the load balancer with the specified backend servers and port.

## Cleaning

To clean up the build artifacts, you can use the following command:

```bash
make clean
```

This will remove the `lb.exe` executable file from the `bin` directory.

## Reference Links

- [Refence Article](https://kasvith.me/posts/lets-create-a-simple-lb-go/)


Feel free to customize this README.md file to better suit your project's needs.