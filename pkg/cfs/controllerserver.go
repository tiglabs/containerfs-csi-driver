/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cfs

import (
	"strconv"
	"net/http"
	"io/ioutil"
	"encoding/json"

	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/golang/glog"
	"github.com/kubernetes-csi/drivers/pkg/csi-common"
	"golang.org/x/net/context"
	"k8s.io/kubernetes/pkg/volume/util"
)

var defaultMaster string


type CFSMasterLeader struct {
    LeaderAddr string
}

type controllerServer struct {
	*csicommon.DefaultControllerServer
}

func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {

	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		glog.V(3).Infof("invalid create cfs volume req: %v", req)
		return nil, err
	}

	// Volume Size - Default is 1 GiB
	volSizeBytes := int64(1 * 1024 * 1024 * 1024)
	if req.GetCapacityRange() != nil {
		volSizeBytes = int64(req.GetCapacityRange().GetRequiredBytes())
	}
	volSizeGB := int(util.RoundUpSize(volSizeBytes, 1024*1024*1024))

	//Volume Name
	volName := req.GetName()

	cfsMasterHost1 := req.GetParameters()["cfsMaster1"]
	cfsMasterHost2 := req.GetParameters()["cfsMaster2"]
	cfsMasterHost3 := req.GetParameters()["cfsMaster3"]
	defaultMaster = cfsMasterHost1

	var cfsMasterLeader CFSMasterLeader
	getClusterUrl := "http://" + cfsMasterHost1 + "/admin/getCluster"
	r0, err := http.Get(getClusterUrl)
	if err != nil {
        glog.Errorf("Get cfs Cluster info failed, error:%v",err)
        return nil, err
    }
    defer r0.Body.Close()
    body0, err := ioutil.ReadAll(r0.Body)
	json.Unmarshal(body0, &cfsMasterLeader)

	cfsMasterLeaderHost := cfsMasterLeader.LeaderAddr
	glog.V(1).Infof("CFS Master Leader Host is:%v",cfsMasterLeaderHost)

	CreateVolUrl := "http://" + cfsMasterLeaderHost + "/admin/createVol?name=" + volName + "&replicas=3&type=extent"
	r1, err := http.Get(CreateVolUrl)
	if err != nil {
		glog.Errorf("CreateVol cfs failed, error:%v",err)
		return nil, err
	}
	defer r1.Body.Close()
	body1, err := ioutil.ReadAll(r1.Body)
	if err != nil {
		glog.Errorf("CreateVol cfs get response failed, error:%v",err)
		return nil, err
    }
 	glog.V(1).Infof("Create cfs volume:%v response body: %v success",volName, string(body1))

	var count int
	if volSizeGB%120 == 0 {
		count = volSizeGB / 120
	} else {
		count = volSizeGB / 120 + 1
		volSizeGB = count * 120
	}

	num := strconv.Itoa(count)
	CreateVolDataPartionUrl := "http://" + cfsMasterLeaderHost + "/dataPartition/create?count=" + num + "&name=" + volName + "&type=extent"
	r2, err := http.Get(CreateVolDataPartionUrl)
    if err != nil {
        glog.Errorf("CreateVol DataPartion failed, error:%v",err)
        return nil, err
    }

	defer r2.Body.Close()
    body2, err := ioutil.ReadAll(r2.Body)
    if err != nil {
        glog.Errorf("CreateVol cfs get DataPartion response failed, error:%v",err)
        return nil, err
    }
    glog.V(1).Infof("Create cfs volume:%v DataPartion response body: %v success",volName, string(body2))

	glog.V(1).Infof("succesfully created cfs volume with name:%v and size: %v", volName, volSizeGB)

	resp := &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			Id: volName,
			Attributes: map[string]string{
				"cfsvolname":  volName,
				"cfsMaster1":  cfsMasterHost1,
				"cfsMaster2":  cfsMasterHost2,
				"cfsMaster3":  cfsMasterHost3,
			},
		},
	}
	return resp, nil
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		glog.V(3).Infof("invalid delete volume req: %v", req)
		return nil, err
	}
	volumeId := req.VolumeId

	var cfsMasterLeader CFSMasterLeader
    getClusterUrl := "http://" + defaultMaster + "/admin/getCluster"
    r, err := http.Get(getClusterUrl)
    if err != nil {
        glog.Errorf("Get cfs Cluster info failed, error:%v",err)
        return nil, err
    }
    defer r.Body.Close()
    body, err := ioutil.ReadAll(r.Body)
    json.Unmarshal(body, &cfsMasterLeader)

    cfsMasterLeaderHost := cfsMasterLeader.LeaderAddr
    glog.V(1).Infof("CFS Master Leader Host is:%v", cfsMasterLeaderHost)

	DeleteVolUrl := "http://" + cfsMasterLeaderHost + "/vol/delete?name=" + volumeId
    r1, err := http.Get(DeleteVolUrl)
    if err != nil {
        glog.Errorf("DeleteVol cfs failed, error:%v",err)
        return nil, err
    }
    defer r1.Body.Close()

	glog.V(1).Infof("Delete cfs volume :%s deleted successs", volumeId)
	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	for _, cap := range req.VolumeCapabilities {
		if cap.GetAccessMode().GetMode() != csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER {
			return &csi.ValidateVolumeCapabilitiesResponse{Supported: false, Message: ""}, nil
		}
	}
	return &csi.ValidateVolumeCapabilitiesResponse{Supported: true, Message: ""}, nil
}
