// Copyright 2015 Keybase Inc. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

// +build windows

package libdokan

import (
	"github.com/keybase/kbfs/dokan"
	"github.com/keybase/kbfs/libkbfs"
)

// File represents KBFS files.
type File struct {
	FSO
}

func newFile(folder *Folder, node libkbfs.Node, name string, parent libkbfs.Node) *File {
	f := &File{FSO{
		name:   name,
		parent: parent,
		folder: folder,
		node:   node,
	}}
	f.refcount.Increase()
	return f
}

// GetFileInformation for dokan.
func (f *File) GetFileInformation(*dokan.FileInfo) (a *dokan.Stat, err error) {
	ctx := NewContextWithOpID(f.folder.fs)
	f.folder.fs.log.CDebugf(ctx, "File GetFileInformation node=%v start", f.node)
	defer func() { f.folder.fs.reportErr(ctx, err) }()

	a, err = eiToStat(f.folder.fs.config.KBFSOps().Stat(ctx, f.node))
	if a != nil {
		f.folder.fs.log.CDebugf(ctx, "File GetFileInformation node=%v => %v", f.node, *a)
	} else {
		f.folder.fs.log.CDebugf(ctx, "File GetFileInformation node=%v => Error %T %v", f.node, err, err)
	}
	return a, err
}

// CanDeleteFile - return just nil
// TODO check for permissions here.
func (*File) CanDeleteFile(*dokan.FileInfo) error {
	return nil
}

// Cleanup - for dokan, remember to handle deletions.
func (f *File) Cleanup(fi *dokan.FileInfo) {
	var err error
	ctx := NewContextWithOpID(f.folder.fs)

	f.folder.fs.log.CDebugf(ctx, "Cleanup %v", *f)
	if fi != nil && fi.DeleteOnClose() {
		f.folder.fs.log.CDebugf(ctx, "Removing file in cleanup %s", f.name)
		defer func() { f.folder.fs.reportErr(ctx, err) }()

		err = f.folder.fs.config.KBFSOps().RemoveEntry(ctx, f.parent, f.name)
	}

	if f.refcount.Decrease() {
		f.folder.fs.log.CDebugf(ctx, "Forgetting file node")
		f.folder.forgetNode(f.node)
		// TODO this should not be needed in future.
		f.folder.fs.config.KBFSOps().Sync(ctx, f.node)
	}
}

// FlushFileBuffers performs a (f)sync.
func (f *File) FlushFileBuffers(*dokan.FileInfo) (err error) {
	ctx := NewContextWithOpID(f.folder.fs)
	f.folder.fs.log.CDebugf(ctx, "File FlushFileBuffers")
	defer func() { f.folder.fs.reportErr(ctx, err) }()

	err = f.folder.fs.config.KBFSOps().Sync(ctx, f.node)
	return err
}

// ReadFile for dokan reads.
func (f *File) ReadFile(fi *dokan.FileInfo, bs []byte, offset int64) (n int, err error) {
	ctx := NewContextWithOpID(f.folder.fs)
	f.folder.fs.log.CDebugf(ctx, "File Read")
	defer func() { f.folder.fs.reportErr(ctx, err) }()

	var nlarge int64
	nlarge, err = f.folder.fs.config.KBFSOps().Read(ctx, f.node, bs, offset)

	// This is safe since length of slices always fits into an int
	return int(nlarge), err
}

// WriteFile for dokan writes.
func (f *File) WriteFile(fi *dokan.FileInfo, bs []byte, offset int64) (n int, err error) {
	ctx := NewContextWithOpID(f.folder.fs)
	f.folder.fs.log.CDebugf(ctx, "File Write sz=%d ", len(bs))
	defer func() { f.folder.fs.reportErr(ctx, err) }()

	if offset == -1 {
		ei, err := f.folder.fs.config.KBFSOps().Stat(ctx, f.node)
		if err != nil {
			return 0, err
		}
		offset = int64(ei.Size)
	}

	err = f.folder.fs.config.KBFSOps().Write(
		ctx, f.node, bs, offset)
	return len(bs), err
}

// SetEndOfFile for dokan (f)truncates.
func (f *File) SetEndOfFile(fi *dokan.FileInfo, length int64) (err error) {
	ctx := NewContextWithOpID(f.folder.fs)
	f.folder.fs.log.CDebugf(ctx, "File SetFileTime")
	defer func() { f.folder.fs.reportErr(ctx, err) }()

	return f.folder.fs.config.KBFSOps().Truncate(ctx, f.node, uint64(length))
}
