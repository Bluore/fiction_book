package service

// calcCreatorRevenue 计算创作者收益：按用户实付虚拟货币折半并向上取整。
func calcCreatorRevenue(paid int64) int64 {
	if paid <= 0 {
		return 0
	}
	return paid/2 + paid%2
}
