linode
======

go package for interacting with the Linode API. Automatically batches requests.

[GoDoc](http://godoc.org/github.com/awilliams/linode)

As of now, it only supports two API methods: 

 * [linode.list()](https://www.linode.com/api/linode/linode.list)
 * [linode.ip.list()](https://www.linode.com/api/linode/linode.ip.list)

## Usage

```go
apiKey := "secretAPIKey"
client := linode.NewClient(apiKey)

linodes, err := client.LinodeList()
ips, err := client.LinodeIPList([]int{1,2,3})
```
