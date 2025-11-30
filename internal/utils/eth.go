package utils

import "math/big"

func WeiToEthString(wei *big.Int) string {
	if wei == nil {
		return "0"
	}

	// 转换为 big.Float
	fwei := new(big.Float).SetInt(wei)
	// 除以 10^18
	ethValue := new(big.Float).Quo(fwei, big.NewFloat(1e18))
	// 格式化为字符串，保留 18 位小数
	return ethValue.Text('f', 18)
}
