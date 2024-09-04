package entity

type HelmRepository struct {
	K8sConfig
	Name                  string `json:"name" gorm:"not null;unique"`
	Url                   string `json:"url" gorm:"not null;unique"`
	Username              string `json:"username"`
	Password              string `json:"password"`
	IsActive              bool   `json:"is_active" gorm:"type:boolean;default:true"`
	CertFile              string `json:"cert_file"`
	KeyFile               string `json:"key_file"`
	CAFile                string `json:"ca_file"`
	InsecureSkipTlsVerify bool   `json:"insecure_skip_tls_verify"`
}

type HelmChartInfo struct {
	K8sConfig
	Namespace       string `json:"namespace"`
	ReleaseName     string `json:"release_name"`
	ChartName       string `json:"chart_name"`
	ValuesYaml      string `json:"values_yaml"`
	CreateNamespace bool   `json:"create_namespace"`
}
