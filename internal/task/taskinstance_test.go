package task

import (
	"os"
	"strings"
	"taskd/dao"
	"testing"
)

func TestTaskInstance_Compile(t *testing.T) {
	bs2, err := os.ReadFile("./testdata/template_train_standalone.yaml")
	if err != nil {
		panic(err)
	}

	expectHcaYaml, err := os.ReadFile("./testdata/expect_train_standalone.yaml")
	if err != nil {
		panic(err)
	}

	tests := []struct {
		name    string
		ti      *TaskInstance
		want    string
		wantErr bool
	}{
		{
			name: "hca",
			ti: &TaskInstance{
				TaskRec: dao.TaskRec{
					TaskRuntimeRec: dao.TaskRuntimeRec{},
					TaskObjRec: dao.TaskObjRec{
						UUID:      "uuid0",
						Namespace: "for-test-namespace",
						Name:      "for-test-name",
						Pool:      "h100-mem-2t",
						Args:      "",
						Extra:     `{"adjustable": false,"conda": "/home/jovyan/work/zjz/env/zjz","createTime": "2024-10-31 17:39:13","createdBy": "zhangzhaojin40458","executeImage": "registry.ai.sangfor.com/ai-sangfor/code-studio:torch-2.1-cu121-v1.0.0","id": 16956,"inputDatasets": null,"master": {"envs": null,"resources": {"cpu": "112","memory": "983040Mi","nvidia.com/gpu": "8"}},"masterCommand": "sleep infinity","masterEnv": null,"masterNum": 1,"masterResourceCpu": 112,"masterResourceGpu": 8,"masterResourceHca": 0,"masterResourceMemory": 983040,"maxRetries": 0,"maxTimes": 0,"monitorUrl": null,"mounts": [{"mountPath": "/home/jovyan","server": null,"serverPath": "dfdfd","type": "pvc","volumeName": "pvc-8dd98f6a-c30f-4d59-a017-23e60309b06f"},{"mountPath": "/mnt/project/aip","server": null,"serverPath": "/mnt/project/aip","type": "hostpath","volumeName": "hostpath-fd621f42-d0d0-4fe6-98e1-bf6b41252e33"}],"name": "kaip-qwen25-72-f-te72","namespace": "zhangzhaojin40458","notebook": "tgy","noticeFlag": null,"outputDatasets": null,"ownerName": "zhangzhaojin40458","poolName": "h100-mem-2t","projectName": "aip","quotas": [{"pool_name": "h100-mem-2t","res_name": "a800","res_num": 16}],"rawName": "qwen25-72-full-jkz","schedule": null,"status": "scheduled","tags": "{\"gpu\":\"a800\"}","timeout": null,"trainMode": "distributed","type": "pytorch","uuid": "f7dcdf4c-0a76-4967-a3df-a307d6740555","volumeMount": "(pvc)dfdfd:/home/jovyan,(hostpath)/mnt/project/aip:/mnt/project/aip","wechatRobot": null,"worker": {"envs": null,"resources": {"cpu": "112","memory": "983040Mi","nvidia.com/gpu": "8"}},"workerCommand": "sleep infinity","workerEnv": null,"workerNum": 1,"workerResourceCpu": 112,"workerResourceGpu": 8,"workerResourceHca": 0,"workerResourceMemory": 983040}`,
					},
				},
				template: &dao.TemplateRec{
					Schema: string(bs2),
				},
			},
			want:    string(expectHcaYaml),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ti.Compile()
			if (err != nil) != tt.wantErr {
				t.Errorf("TaskInstance.Compile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			os.WriteFile("out.yaml", []byte(got), 0644)
			if strings.TrimSpace(got) != strings.TrimSpace(tt.want) {
				t.Errorf("TaskInstance.Compile() = %v, want %v", got, tt.want)
			}
			os.Remove("out.yaml")
		})
	}
}
