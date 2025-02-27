package core

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestObjectData(t *testing.T) {
	SetUpRepositoryFromTempDir(t)

	WriteFileFromRelativePath(t, "project.md", ""+
		"# Project\n"+
		"\n"+
		"## TODO: Backlog\n"+
		"\n"+
		"[ ] Test `ObjectData`\n")
	parsedFile := ParseFileFromRelativePath(t, "project.md")

	dummyPackFile := DummyPackFile()

	fileSrc, err := NewFile(dummyPackFile, parsedFile)
	require.NoError(t, err)
	dataSrc, err := NewObjectData(fileSrc)
	require.NoError(t, err)

	// Marshall YAML
	txt, err := yaml.Marshal(dataSrc)
	require.NoError(t, err)
	reBase64 := regexp.MustCompile(`^[A-Za-z0-9+=/]*$`)
	assert.Regexp(t, reBase64, strings.TrimSpace(string(txt)))

	// Unmarshall YAML
	var dataDest ObjectData
	err = yaml.Unmarshal(txt, &dataDest)
	require.NoError(t, err)

	// Unmarshall
	fileDest := new(File)
	err = dataDest.Unmarshal(fileDest)
	require.NoError(t, err)
	assert.Equal(t, "Project", fileDest.Title.String())
}

func TestPackFile(t *testing.T) {

	t.Run("NewPackFileFromParsedFile", func(t *testing.T) {
		oid.UseSequence(t)
		FreezeNow(t)
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		parsedFile := ParseFileFromRelativePath(t, "go.md")

		packFile, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)

		// Pack must have been saved to disk
		assert.FileExists(t, packFile.ObjectPath())

		// A blob must have been created for the original file
		assert.Len(t, packFile.BlobRefs, 1)
		for _, blob := range packFile.BlobRefs {
			assert.FileExists(t, blob.ObjectPath())
		}
		// The same blob can be found searching by mimetype
		blob := packFile.FindFirstBlobWithMimeType("text/markdown")
		assert.NotNil(t, blob)
	})

	t.Run("NewPackFileFromParsedFile_DryRun", func(t *testing.T) {
		oid.UseSequence(t)
		FreezeNow(t)
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")
		CurrentConfig().DryRun = true

		parsedFile := ParseFileFromRelativePath(t, "go.md")

		packFile, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)

		// Pack must have been saved to disk
		assert.NoFileExists(t, packFile.ObjectPath())
		// No blobs have been generated
		assert.Empty(t, packFile.BlobRefs)
	})

	t.Run("NewPackFileFromParsedMedia", func(t *testing.T) {
		oid.UseSequence(t)
		FreezeNow(t)
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		parsedFile := ParseFileFromRelativePath(t, "go.md")
		require.Len(t, parsedFile.Medias, 1)
		parsedMedia := parsedFile.Medias[0]
		packFile, err := NewPackFileFromParsedMedia(parsedMedia)
		require.NoError(t, err)

		// Pack must have been save to disk
		assert.FileExists(t, packFile.ObjectPath())

		// Blob must have been created
		assert.GreaterOrEqual(t, len(packFile.BlobRefs), 1)
		for _, blob := range packFile.BlobRefs {
			assert.FileExists(t, blob.ObjectPath())
		}
	})

	t.Run("NewPackFileFromParsedMedia_DryRun", func(t *testing.T) {
		oid.UseSequence(t)
		FreezeNow(t)
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")
		CurrentConfig().DryRun = true

		parsedFile := ParseFileFromRelativePath(t, "go.md")
		require.Len(t, parsedFile.Medias, 1)
		parsedMedia := parsedFile.Medias[0]
		packFile, err := NewPackFileFromParsedMedia(parsedMedia)
		require.NoError(t, err)

		// Pack must have been save to disk
		assert.NoFileExists(t, packFile.ObjectPath())
		// No blobs must have been generated
		assert.Empty(t, packFile.BlobRefs)
	})

	t.Run("LoadPackFileFromPath", func(t *testing.T) {
		oid.UseSequence(t)
		FreezeNow(t)
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		parsedFile := ParseFileFromRelativePath(t, "go.md")

		packFileOriginal, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)

		packFileRead, err := LoadPackFileFromPath(packFileOriginal.ObjectPath())
		require.NoError(t, err)

		// reread pack file must be identical
		assert.Equal(t, packFileOriginal.OID, packFileRead.OID)
		assert.Equal(t, len(packFileOriginal.PackObjects), len(packFileRead.PackObjects))
		assert.Equal(t, len(packFileOriginal.BlobRefs), len(packFileRead.BlobRefs))
	})

	t.Run("LoadPackFileFromPath", func(t *testing.T) {
		oid.UseSequence(t)
		FreezeNow(t)
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		parsedFile := ParseFileFromRelativePath(t, "go.md")

		packFile, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)

		ref := packFile.Ref()
		assert.Equal(t, PackFileRef{
			RelativePath: "go.md",
			OID:          packFile.OID,
			CTime:        packFile.CTime,
		}, ref)
	})

	t.Run("YAML", func(t *testing.T) {
		oid.UseSequence(t)
		FreezeOn(t, "2023-01-01 12:30")
		SetUpRepositoryFromGoldenDirNamed(t, "TestMinimal")

		parsedFile := ParseFileFromRelativePath(t, "go.md")

		packFileSrc, err := NewPackFileFromParsedFile(parsedFile)
		require.NoError(t, err)

		// Marshmall YAML
		buf := new(bytes.Buffer)
		err = packFileSrc.Write(buf)
		require.NoError(t, err)
		cYAML := buf.String()
		assert.Equal(t, strings.TrimSpace(`
oid: 23334328153429ce5ba99acd83181b06c44f30af
file_relative_path: go.md
file_mtime: 2023-01-01T12:30:00Z
file_size: 1
ctime: 2023-01-01T12:30:00Z
objects:
    - oid: "1000000000000000000000000000000000000000"
      kind: file
      ctime: 2023-01-01T12:30:00Z
      desc: file "go.md" [1000000000000000000000000000000000000000]
      data: eJyUk9tu1DAQhu/zFMPuTTc05+3S5gYQh0UCqaiqhERVZZ141rE28US20wOUd0dOwnaFRFVyNfPPry+e8Zgkz2GWxM/7Zp5pepGDIK9j1W4rGywGQppl2TJLT5OTbJmeVXhSsrMzVvHTLDlNynhVLZfbLGZbT2PDrLzBomO2dqCw5d6t3MlGqt0A3mpStmiZtahzePAAACwTJh+iwFmYtVqWvcVJfCzvLVbaBnNYk2dq0rZ4zEvi9zk8vBz8c6eM0RwucIsaVTX4GqYEfJLGkr4fHZt5PaabKX9jqNfOXVvbmTyKUIWulQ65ZCFpEbksWlNx1GkSmrWtVKJw5J4JXEycq/Fn10d/MIJCjjcRpyqC2VxQJGi2gFtmgKORQiGH8h4uqERtYa1RGmxRHzsFvsodHgNTHD6jgsua2s6QAmZhTSQaBKkgjeNXobdv+2PDTF0xzfdtfyExTeVbzSxwQgO2RvD9ydCQIN8HjZ1Gg8q+Ht1BEIzBW/B9QV2N2vfDUXpx5ajXR2HUuvEY16W5EYvHc1yevz/P4R2p6RbMWPHdgByqIgUfek0dHo5qqoTYRwvYzDW2UnHUQRqnWRCvgnS18YY7LxqpMIeVZ+QPzCHxambq/9jc1soWcxi5SRAnl0maZ3Eex9+9sqFyv6Djk0qf/aTGrW0HusU7G7VM7zjdqqlysO3w89ck/r3ypKWQijUH0h5TaWQWecHsP47fd/xpgxvp3VOG3wEAAP//nb42PA==
    - oid: "3000000000000000000000000000000000000000"
      kind: note
      ctime: 2023-01-01T12:30:00Z
      desc: 'note "Reference: Golang History" [3000000000000000000000000000000000000000]'
      data: eJzckT9v2zwQh3d+ioM8JAEiixKtROL0bn6BLkWQqUWhUOSJIizzBIpOaqAfvpDjP+1QI0C3cjvq0XO/O5IzEhLBP3YSNg07K8FSGrDDgF5jamlQ3qa9myKFPRuV3nRuwOagLoQQK1FUeSlWRa2xbFVdK20qkVd5yx/0atUJrjp2+SXJP5xm47yRcI7CoosDSrh5Ot1IWB/Swf/v6W7YQN42Ry5J2NRTiKf6d5YFHFR0r9iMKvbz0MutYW9u4wbnNxJuLC2uNFIxBtfuIk6SAQBMtAsz18c4TjLL0C9n14jGqSUFm81Vtqbmdgxkg9punbfN7Nwpi3cHR1T2aJtPCpZ+KU4PcIHOwOXj4DxKqJgmH9FHCT/SA7FYwB9nYQfiZXF0vBzr//5yoqPn63uzb7cnjaWlwdfMkM4gWVjKLCV38KYmMDg569FAu4cnajFEWAd0E24x3M838Nlt8B6UN/AJPTz3tB0n8qAirInsgOA8FJw/Lk8LaHo19RKw01zp8lGXolC6U7oyba24ybE1tahFma8edFlx1pLZn5f2r60koIpoGhUlFLwQKc9Tnj/nhRRccv6F7UZzHXDe4PdrwM8AAAD///B/Xxg=
    - oid: "4000000000000000000000000000000000000000"
      kind: link
      ctime: 2023-01-01T12:30:00Z
      desc: link "https://go.dev/doc/" [4000000000000000000000000000000000000000]
      data: eJyMzUFuwyAQheE9p0DsEw8Mqey5QC/QVTfWBMYOKjHInUQ5ftWq6yhv/X96rWSyLsJrc6Zz+lpKlfkPBkSMGEZ/whimJKczTxOnPKIf/RneUowLAi9ma/pPHL78tUtlLXeZO+uF7NqO12xUHkr2vVXeVnPbK9mLav+mYVjbMct9yC0NRotWIeucWdu88VV+uUm7sEqeWckGCHgAfwD/4QMhEMCnufX8PChblsez4CcAAP//haFZtw==
    - oid: "5000000000000000000000000000000000000000"
      kind: note
      ctime: 2023-01-01T12:30:00Z
      desc: 'note "Flashcard: Golang Logo" [5000000000000000000000000000000000000000]'
      data: eJy8kcGO0zAQhu9+iqE9FCK568RJ2/iCuMCFIxISCEXjeOJY9dpR7C4g8fCoabvLAVZ7Yk4z/3ye+UeOzihYNeJlsWLJn6wCG/ngMY09zobb6DFY7qONbML+ODhP3TK3klLWsjqUjayrtqdGY9tibw6yPJRa7Pq6HqTAgT09WZUvtnJ0wSh49MGyy54UbN7fFAUfFmvwMdq4YT4G212h1YqlMc75Vv8Bspk8ZvdA3YR5PN+6vTfsuzs678JRwcbG9b9WYM6z06dMSTEAgIz2mp2Dg43sSVpK7wIpKPesjyFTyAp+8aW7XsPft7Cl/XnEDCZSgjwSFMUVOH9CUcBM00yJQn57oTnnl+QdFIWN00hzUWwv0quv56nfXm/v7sk4THc2btODfXNz1I2YRgWNbPq91q0otdBG0H5oUbeNwcN+1xjaUSXaocaB6Wh+Pl7xP2zOhJlMh1lBJSrJRclF+amslBRKiC/sNJnnARcM/XgO+B0AAP//8jjlaA==
    - oid: "6000000000000000000000000000000000000000"
      kind: flashcard
      ctime: 2023-01-01T12:30:00Z
      desc: flashcard "Golang Logo" [6000000000000000000000000000000000000000]
      data: eJyMjTFrwzAQhXf9imumViBHspyQaCmdunQsFFqKuUhn2USxjKSEDv3xxaalW8hNx3vvey8OzsBqK2+7FZvQHrshULuAtda60fVObXRT7y1tDrjfo3U7rXbqILe2aTotsWP/yErdvDXG8gdtboYSBSzDhdoJS2/Ax+rkWA5nP/+iC5h7i8kJHwOOXoToI8t9TKUtQwlk4Hkx4GU2CvpsGACAAB9Zl+JYDLz1WMBFylB6As5/ibmKc0g0Jco0lkd2QHs08C2Whifg3Mepp8R5xRbp7mNe+byv1idyA+a1j1W++AdmE2Eh12IxUMtaC6mEVK+qNloaKd/ZeXLXA8Po6Ota4CcAAP//puiJcQ==
    - oid: "7000000000000000000000000000000000000000"
      kind: note
      ctime: 2023-01-01T12:30:00Z
      desc: 'note "TODO: Conferences" [7000000000000000000000000000000000000000]'
      data: eJykj0+L2zAQxe/6FINzcFvwRn8Sx9K1LT3uZU8txStLY1vEKxl53D/QD1+SbkihZQnsnDT6Pb33lII3UBz4bVOwZVoHA0OqKPlUuRR7zBgdLmy27tiHCduzpVRK7ZRsxF7tpHa476zW1vlGiUZ0vHa7Xa+47dn1SSFubnEM0Rs4VWAUaEID5cP9h3sD76+FSjalOLTPvCjYMqZMl/0vIcs4WQrfsJ0tjafP3T159j0cwxTi0UA5pM1/3C1RDt1KuBgGAEB2eD6dpoIhsevVeZ1CRAOyYS5FwkgGflVnutnAPwHsTN7Bl09pHjG7FOHjmtOMX9+MRPNittvhQu5w3b6Fx03GpxA95kpyqSpeV7J+vIS1o11GA05o592h81Y4rnteCy5roZVuatGhqPFg9b4/KNYl/9NA+foCJXMZLaFvLRn4A0TFxYOQRnHD+We2zv5lwcn0x0uC3wEAAP//0l7ORQ==
    - oid: "8000000000000000000000000000000000000000"
      kind: reminder
      ctime: 2023-01-01T12:30:00Z
      desc: 'reminder #reminder-2023-06-26 [8000000000000000000000000000000000000000]'
      data: eJyMjs9OhDAQxu99igYP6IGlf1iE3o0v4EljSLcdoBHapgxmH9+I2ax62DD5bvP7fTPBWUWzhu2bjERtPno3QbeJQkpZSdHwo6xEa+B40m2rjW0kb/iJ1aaqesl0T65Kxnff8gEv0uNuKcGk0X1CFzWOig7hMFtiYTHJRXTBK5q/PYc4QjLB06c1hQjv9yNiXFRZDpfNAdbyISeoB0XzuwSz8xZSIZiQBasLUedk0gt2EVIf0gy206goY4wXW14YU1teiYfzf+5a84szCTT+AbYiLpT8AdZobwPfP55vAV8BAAD//8cxhIY=
blobs:
    - oid: "2000000000000000000000000000000000000000"
      mime: text/markdown
      attributes: {}
      tags:
        - original
        - markdown
`), strings.TrimSpace(cYAML))

		// Unmarshall YAML
		packFileDest := new(PackFile)
		err = packFileDest.Read(buf)
		require.NoError(t, err)
		require.Equal(t, oid.MustParse("23334328153429ce5ba99acd83181b06c44f30af"), packFileDest.OID)
		require.Len(t, packFileDest.PackObjects, 7)

		// Unmarshall the note
		noteDest := new(Note)
		err = packFileDest.PackObjects[0].Data.Unmarshal(noteDest)
		require.NoError(t, err)
		assert.Equal(t, "Go", noteDest.Title.String())

		// Unmarshall the note
		flashcardDest := new(Flashcard)
		err = packFileDest.PackObjects[1].Data.Unmarshal(flashcardDest)
		require.NoError(t, err)
		assert.Equal(t, "Golang History", flashcardDest.ShortTitle.String())

		// Unmarshall a single object by OID
		noteCopy := new(Note)
		err = packFileDest.UnmarshallObject(packFileDest.PackObjects[0].OID, noteCopy)
		require.NoError(t, err)
		require.EqualValues(t, noteDest, noteCopy)
	})

}
