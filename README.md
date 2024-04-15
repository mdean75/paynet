## Paynet

### Usage

#### Client

For client mode, specify backend ip and port, number of client connections to establish and number of concurrent workers per client connection.
Default parameters are 127.0.0.1 on port 3000 using a single client connection with 10 workers for 10 seconds.

The client seems to be most stable using no more than 5 workers for each connection. This means that it is better and more stable to scale by number of connections while keeping num workers set to 5.
Higher than 5 workers even with a single client connection inconsistently causes dropped requests.

```shell
./paynet -m client -i <server ip> -p 3000 -c 10 -w 5 -d 10
```
#### Server
Server mode has fewer parameters and only uses server ip and port. 
The tcp listener is started on the specified ip and port. 
```shell
./paynet -m server -i <server ip> -p 3000
```
