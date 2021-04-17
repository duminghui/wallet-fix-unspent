module wallet-fix-unspent

go 1.16

require (
	github.com/duminghui/go-rpcclient v0.0.0-20210413134428-bd1d56db7aad
	github.com/sirupsen/logrus v1.8.1
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/sys v0.0.0-20210412220455-f1c623a9e750 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

replace github.com/duminghui/go-rpcclient => ../go-rpcclient
