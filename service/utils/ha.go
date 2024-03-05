package utils

import "encoding/json"

type RedisMasterInfo struct {
	Name              string `json:"name"`
	Ip                string `json:"ip"`
	Port              string `json:"port"`
	Runid             string `json:"runid"`
	RoleReported      string `json:"role-reported"`
	NumSlaves         string `json:"num-slaves"`
	NumOtherSentinels string `json:"num-other-sentinels"`
	Quorum            string `json:"quorum"`
}

type RedisReplicaInfo struct {
	Name             string `json:"name"`
	Ip               string `json:"ip"`
	Port             string `json:"port"`
	Runid            string `json:"runid"`
	RoleReported     string `json:"role-reported"`
	MasterHost       string `json:"master-host"`
	MasterPort       string `json:"master-port"`
	MasterLinkStatus string `json:"master-link-status"`
}

func NewRedisMasterInfo(b []byte) (RedisMasterInfo, error) {
	var redisMasterInfo RedisMasterInfo
	err := json.Unmarshal(b, &redisMasterInfo)
	if err != nil {
		return redisMasterInfo, err
	}

	return redisMasterInfo, nil
}

func NewRedisReplicaInfo(b []byte) ([]RedisReplicaInfo, error) {
	var redisReplicaInfo []RedisReplicaInfo
	err := json.Unmarshal(b, &redisReplicaInfo)
	if err != nil {
		return redisReplicaInfo, err
	}

	return redisReplicaInfo, nil
}

func GetIndexOfInstanceMatchingIpAndRunId(replicas []RedisReplicaInfo, ip, runId string) int {
	index := -1
	for i, instance := range replicas {
		if instance.Ip == ip && instance.Runid == runId {
			index = i
		}
	}

	return index
}
