package dev

//
//import (
//	"fmt"
//	"time"
//	"log"
//	"github.com/libvirt/libvirt-go-xml"
//	libvirt "github.com/libvirt/libvirt-go"
//	"io"
//	"encoding/xml"
//)
//
//const (
//	poolName = "default"
//	volumeName = "extra-worker"
//	baseVolumeID = "/var/lib/libvirt/images/coreos_base"
//	size = 17706254336
//	)
//
//// WaitSleepInterval time
//var WaitSleepInterval = 1 * time.Second
//
//// WaitTimeout time
//var WaitTimeout = 5 * time.Minute
//
//// waitForSuccess wait for success and timeout after 5 minutes.
//func waitForSuccess(errorMessage string, f func() error) error {
//	start := time.Now()
//	for {
//		err := f()
//		if err == nil {
//			return nil
//		}
//		log.Printf("[DEBUG] %s. Re-trying.\n", err)
//
//		time.Sleep(WaitSleepInterval)
//		if time.Since(start) > WaitTimeout {
//			return fmt.Errorf("%s: %s", errorMessage, err)
//		}
//	}
//}
//
//func newDefVolume() libvirtxml.StorageVolume {
//	return libvirtxml.StorageVolume{
//		Target: &libvirtxml.StorageVolumeTarget{
//			Format: &libvirtxml.StorageVolumeTargetFormat{
//				Type: "qcow2",
//			},
//			Permissions: &libvirtxml.StorageVolumeTargetPermissions{
//				Mode: "644",
//			},
//		},
//		Capacity: &libvirtxml.StorageVolumeSize{
//			Unit:  "bytes",
//			Value: 1,
//		},
//	}
//}
//
//// network transparent image
//type image interface {
//	Size() (uint64, error)
//	Import(func(io.Reader) error, libvirtxml.StorageVolume) error
//	String() string
//}
//
//
//func newDefBackingStoreFromLibvirt(baseVolume *libvirt.StorageVol) (libvirtxml.StorageVolumeBackingStore, error) {
//	baseVolumeDef, err := newDefVolumeFromLibvirt(baseVolume)
//	if err != nil {
//		return libvirtxml.StorageVolumeBackingStore{}, fmt.Errorf("could not get volume: %s", err)
//	}
//	baseVolPath, err := baseVolume.GetPath()
//	if err != nil {
//		return libvirtxml.StorageVolumeBackingStore{}, fmt.Errorf("could not get base image path: %s", err)
//	}
//	backingStoreDef := libvirtxml.StorageVolumeBackingStore{
//		Path: baseVolPath,
//		Format: &libvirtxml.StorageVolumeTargetFormat{
//			Type: baseVolumeDef.Target.Format.Type,
//		},
//	}
//	return backingStoreDef, nil
//}
//
//func newDefVolumeFromLibvirt(volume *libvirt.StorageVol) (libvirtxml.StorageVolume, error) {
//	name, err := volume.GetName()
//	if err != nil {
//		return libvirtxml.StorageVolume{}, fmt.Errorf("could not get name for volume: %s", err)
//	}
//	volumeDefXML, err := volume.GetXMLDesc(0)
//	if err != nil {
//		return libvirtxml.StorageVolume{}, fmt.Errorf("could not get XML description for volume %s: %s", name, err)
//	}
//	volumeDef, err := newDefVolumeFromXML(volumeDefXML)
//	if err != nil {
//		return libvirtxml.StorageVolume{}, fmt.Errorf("could not get a volume definition from XML for %s: %s", volumeDef.Name, err)
//	}
//	return volumeDef, nil
//}
//
//// Creates a volume definition from a XML
//func newDefVolumeFromXML(s string) (libvirtxml.StorageVolume, error) {
//	var volumeDef libvirtxml.StorageVolume
//	err := xml.Unmarshal([]byte(s), &volumeDef)
//	if err != nil {
//		return libvirtxml.StorageVolume{}, err
//	}
//	return volumeDef, nil
//}
//
//func resourceLibvirtVolumeCreate() error {
//	config := &Config{
//		URI: uri,
//	}
//	client, err := config.Client(); if err != nil {
//		return fmt.Errorf("Failed to build libvirt client: %s", err)
//	}
//
//	poolName := poolName
//
//	//client.poolMutexKV.Lock(poolName)
//	//defer client.poolMutexKV.Unlock(poolName)
//
//	pool, err := client.libvirt.LookupStoragePoolByName(poolName)
//	if err != nil {
//		return fmt.Errorf("can't find storage pool '%s'", poolName)
//	}
//	defer pool.Free()
//
//	// Refresh the pool of the volume so that libvirt knows it is
//	// not longer in use.
//	waitForSuccess("error refreshing pool for volume", func() error {
//		return pool.Refresh(0)
//	})
//
//	// Check whether the storage volume already exists. Its name needs to be
//	// unique.
//	if _, err := pool.LookupStorageVolByName(volumeName); err == nil {
//		return fmt.Errorf("storage volume '%s' already exists", volumeName)
//	}
//
//	volumeDef := newDefVolume()
//	volumeDef.Name = volumeName
//
//	volumeFormat := "qcow2"
//	//if _, ok := d.GetOk("format"); ok {
//	//	volumeFormat = d.Get("format").(string)
//	//}
//	volumeDef.Target.Format.Type = volumeFormat
//
//	var (
//		//img    image
//		volume *libvirt.StorageVol
//	)
//
//	// an source image was given, this mean we can't choose size
//	//if source, ok := d.GetOk("source"); ok {
//	//	// source and size conflict
//	//	if _, ok := d.GetOk("size"); ok {
//	//		return fmt.Errorf("'size' can't be specified when also 'source' is given (the size will be set to the size of the source image")
//	//	}
//	//	if _, ok := d.GetOk("base_volume_id"); ok {
//	//		return fmt.Errorf("'base_volume_id' can't be specified when also 'source' is given")
//	//	}
//	//
//	//	if _, ok := d.GetOk("base_volume_name"); ok {
//	//		return fmt.Errorf("'base_volume_name' can't be specified when also 'source' is given")
//	//	}
//	//
//	//	if img, err = newImage(source.(string)); err != nil {
//	//		return err
//	//	}
//	//
//	//	// update the image in the description, even if the file has not changed
//	//	size, err := img.Size()
//	//	if err != nil {
//	//		return err
//	//	}
//	//	log.Printf("Image %s image is: %d bytes", img, size)
//	//	volumeDef.Capacity.Unit = "B"
//	//	volumeDef.Capacity.Value = size
//	//} else {
//	//	_, noSize := d.GetOk("size")
//	//	_, noBaseVol := d.GetOk("base_volume_id")
//	//
//	//	if noSize && noBaseVol {
//	//		return fmt.Errorf("'size' needs to be specified if no 'source' or 'base_volume_id' is given")
//	//	}
//	//	volumeDef.Capacity.Value = uint64(d.Get("size").(int))
//	//}
//
//	if baseVolumeID != "" {
//		//if _, ok := d.GetOk("size"); ok {
//		//	return fmt.Errorf("'size' can't be specified when also 'base_volume_id' is given (the size will be set to the size of the backing image")
//		//}
//		//
//		//if _, ok := d.GetOk("base_volume_name"); ok {
//		//	return fmt.Errorf("'base_volume_name' can't be specified when also 'base_volume_id' is given")
//		//}
//
//		volume = nil
//		volumeDef.Capacity.Value = uint64(size)
//		baseVolume, err := client.libvirt.LookupStorageVolByKey(baseVolumeID)
//		if err != nil {
//			return fmt.Errorf("Can't retrieve volume %s", baseVolumeID)
//		}
//		backingStoreDef, err := newDefBackingStoreFromLibvirt(baseVolume)
//		if err != nil {
//			return fmt.Errorf("Could not retrieve backing store %s", baseVolumeID)
//		}
//		volumeDef.BackingStore = &backingStoreDef
//	}
//
//	//if baseVolumeName, ok := d.GetOk("base_volume_name"); ok {
//	//	if _, ok := d.GetOk("size"); ok {
//	//		return fmt.Errorf("'size' can't be specified when also 'base_volume_name' is given (the size will be set to the size of the backing image")
//	//	}
//	//
//	//	volume = nil
//	//	baseVolumePool := pool
//	//	if _, ok := d.GetOk("base_volume_pool"); ok {
//	//		baseVolumePoolName := d.Get("base_volume_pool").(string)
//	//		baseVolumePool, err = client.libvirt.LookupStoragePoolByName(baseVolumePoolName)
//	//		if err != nil {
//	//			return fmt.Errorf("can't find storage pool '%s'", baseVolumePoolName)
//	//		}
//	//		defer baseVolumePool.Free()
//	//	}
//	//	baseVolume, err := baseVolumePool.LookupStorageVolByName(baseVolumeName.(string))
//	//	if err != nil {
//	//		return fmt.Errorf("Can't retrieve volume %s", baseVolumeName.(string))
//	//	}
//	//	backingStoreDef, err := newDefBackingStoreFromLibvirt(baseVolume)
//	//	if err != nil {
//	//		return fmt.Errorf("Could not retrieve backing store %s", baseVolumeName.(string))
//	//	}
//	//	volumeDef.BackingStore = &backingStoreDef
//	//}
//
//	if volume == nil {
//		volumeDefXML, err := xml.Marshal(volumeDef)
//		if err != nil {
//			return fmt.Errorf("Error serializing libvirt volume: %s", err)
//		}
//
//		// create the volume
//		v, err := pool.StorageVolCreateXML(string(volumeDefXML), 0)
//		if err != nil {
//			return fmt.Errorf("Error creating libvirt volume: %s", err)
//		}
//		volume = v
//		defer volume.Free()
//	}
//
//	// we use the key as the id
//	key, err := volume.GetKey()
//	if err != nil {
//		return fmt.Errorf("Error retrieving volume key: %s", err)
//	}
//	//d.SetId(key)
//	//
//	//// make sure we record the id even if the rest of this gets interrupted
//	//d.Partial(true)
//	//d.Set("id", key)
//	//d.SetPartial("id")
//	//d.Partial(false)
//
//	log.Printf("[INFO] Volume ID: %s", key)
//
//	// upload source if present
//	//if _, ok := d.GetOk("source"); ok {
//	//	err = img.Import(newCopier(client.libvirt, volume, volumeDef.Capacity.Value), volumeDef)
//	//	if err != nil {
//	//		return fmt.Errorf("Error while uploading source %s: %s", img.String(), err)
//	//	}
//	//}
//
//	return nil
//}
