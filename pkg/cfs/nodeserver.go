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
	"fmt"
	"os"
	"os/exec"
	"path"
	"encoding/json"

	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/kubernetes/pkg/util/mount"
	"k8s.io/kubernetes/pkg/volume/util"

	"github.com/kubernetes-csi/drivers/pkg/csi-common"
)

type nodeServer struct {
	*csicommon.DefaultNodeServer
}

func WriteBytes(filePath string, b []byte) (int, error) {
    os.MkdirAll(path.Dir(filePath), os.ModePerm)
    fw, err := os.Create(filePath)
    if err != nil {
        return 0, err
    }
    defer fw.Close()
    return fw.Write(b)
}

func (ns *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	targetPath := req.GetTargetPath()
	notMnt, err := mount.New("").IsLikelyNotMountPoint(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(targetPath, 0750); err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			notMnt = true
		} else {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	if !notMnt {
		return &csi.NodePublishVolumeResponse{}, nil
	}

	mo := req.GetVolumeCapability().GetMount().GetMountFlags()
	if req.GetReadonly() {
		mo = append(mo, "ro")
	}

	master1 := req.GetVolumeAttributes()["cfsMaster1"]
	master2 := req.GetVolumeAttributes()["cfsMaster2"]
	master3 := req.GetVolumeAttributes()["cfsMaster3"]
    volName := req.GetVolumeAttributes()["cfsvolname"]
    cfgpath := "/etc/cfs/fuse.json"

	master := master1 +"," + master2 + "," + master3

	cfgmap := make(map[string]interface{})
	cfgmap["mountpoint"] = targetPath
    cfgmap["volname"] = volName
    cfgmap["master"] = master
    cfgmap["logpath"] = "/export/Logs/cfs/client/"

	cfgstr, err := json.MarshalIndent(cfgmap, "", "      ")
	if err != nil {
        fmt.Printf("cfs client cfg map to json err:%v \n",err)
	    return &csi.NodePublishVolumeResponse{}, err
    }

	WriteBytes(cfgpath, cfgstr)

    cmd := exec.Command("cfs-client", "-c", cfgpath)
	if err = cmd.Run(); err != nil {
		fmt.Printf("cfs mount volume:%v err:%v\n", volName, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	fmt.Printf("cfs mount volume:%v finished\n", volName)
	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	targetPath := req.GetTargetPath()
	notMnt, err := mount.New("").IsLikelyNotMountPoint(targetPath)

	if err != nil {
		if os.IsNotExist(err) {
			return nil, status.Error(codes.NotFound, "Targetpath not found")
		} else {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	if notMnt {
		return nil, status.Error(codes.NotFound, "Volume not mounted")
	}

	err = util.UnmountPath(req.GetTargetPath(), mount.New(""))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return &csi.NodeStageVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return &csi.NodeUnstageVolumeResponse{}, nil
}
