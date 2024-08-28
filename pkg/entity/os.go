package entity

type DiskConf struct {
	Host   Host
	Device string
	LVS
}

type LVS struct {
	LVName string
	VGName string
	Size   string
}
