package util

/*
 排序id以便处理
*/
func GetOrderedIds(popId1, popId2 uint) (uint,uint) {
	if popId1 < popId2 {
		return popId1, popId2
	} else {
		return popId2, popId1
	}
}