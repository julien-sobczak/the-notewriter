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
		FreezeOn(t, "2023-01-01 12:30:00")
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
    - oid: "0000000000000000000000000000000000000001"
      kind: file
      ctime: 2023-01-01T12:30:00Z
      desc: file "go.md" [0000000000000000000000000000000000000001]
      data: eJyUk9tu1DAQhu/zFMPuTTc05+3S5gYQh0UCqaiqhERVZZ141rE28US20wOUd0dOwnaFREVzNfPPry+e8Zgkz2EW/9+XzDzT9CIHQV7Hqt1WNlgMhDTLsmWWniYn2TI9q/CkZGdnrOKnWXKalPGqWi63Wcy2nsaGWXmDRcds7UBhy71buZONVLsBvNWkbNEya1Hn8OABAFgmTD5EgbMwa7Use4uT+FjeW6y0DeawJs/UpG3xmJfE73N4eDn4504Zozlc4BY1qmrwNUwJ+CSNJX0/Ojbzekw3U/7GUK+du7a2M3kUoQpdKx1yyULSInJZtKbiqNMkNGtbqUThyD0TuJg4V+PPro/+YASFHG8iTlUEs7mgSNBsAbfMAEcjhUIO5T1cUInawlqjNNiiPnYKfJU7PAamOHxGBZc1tZ0hBczCmkg0CFJBGsevQm/f9seGmbpimu/b/kJimsq3mlnghAZsjeD7k6EhQb4PGjuNBpV9PbqDIBiDt+D7groate+Ho/TiylGvj8KodeMxrktzIxaP57g8f3+ewztS0y2YseK7ATlURQo+9Jo6PBzVVAmxjxawmWtspeKogzROsyBeBelq4w13XjRSYQ4rz8gfmEPi1czUz9jc1soWcxi5SRAnl0maZ3Eex9+9sqFyv6DPelLpbNrbdqBbvLNRy/SO062aKgfbDj9/TeLfK09aCqlYcyDtMZVGZpEXzP7j+H3Hnza4kd49ZfgdAAD//51JNjw=
    - oid: "0000000000000000000000000000000000000003"
      kind: note
      ctime: 2023-01-01T12:30:00Z
      desc: 'note "Reference: Golang History" [0000000000000000000000000000000000000003]'
      data: eJzckT9v2zwQh3d+ioM8JAEiixKtROL0bn6BLkWQqUWhUOSJIizzBIpOaqAfvpDjP+1QI0C3cjvq0XO/O5IzEhL+sSMSNg07K8FSGrDDgF5jamlQ3qa9myKFPRuV3nRuwOagLoQQK1FUeSlWRa2xbFVdK20qkVd5yx/0atUJrjp2+eWjafKEbZw3Es5RWHRxQAk3T6cbCetDOvj/Pd0NG8jb5sglCZt6CvFU/86ygIOK7hWbUcV+Hnq5NezNbdzg/EbCjaXFlUYqxuDaXcRJMgCAiXZh5voYx0lmGfrl7BrROLWkYLO5ytbU3I6BbFDbrfO2mZ07ZfHu4IjKHm3zScHSL8XpAS7QGbh8HJxHCRXT5CP6KOFHeiAWC/jjLOxAvCyOjpdj/d9fTnT0fH1v9u32pLG0NPiaGdIZJAtLmaXkDt7UBAYnZz0aaPfwRC2GCOuAbsIthvv5Bj67Dd6D8gY+oYfnnrbjRB5UhDWRHRCch4Lzx+VpAU2vpl4CdporXT7qUhRKd0pXpq0VNzm2pha1KPPVgy4rzloy+/PS/rWVBFQRTaOihIIXIuV5yvPnvJCCS86/sN1orgPOG/x+DfgZAAD//+/jXxg=
    - oid: "0000000000000000000000000000000000000004"
      kind: link
      ctime: 2023-01-01T12:30:00Z
      desc: link "https://go.dev/doc/" [0000000000000000000000000000000000000004]
      data: eJyMzTFuwzAMheGdpxC8J6ZEpbB5gV6gUxeDkWhHqGIZLhPk+EWLzkHe/H94rWR2Hb622MEm6WsuVac/GIgoUhj8iWIYk57OMo6S8kB+8Gd8SzHOhDLD2uyfvPpFHexaxcpdp03swm5px2sG04exe29V1gVue2V3Mdu+ue+Xdsx673NLPVixquy6DpY2rXLVXw5pVzHNkxi7gIEO6A/oP3xgQkb8hNuWnwdlzfp4FvwEAAD//4SQWbc=
    - oid: "0000000000000000000000000000000000000005"
      kind: note
      ctime: 2023-01-01T12:30:00Z
      desc: 'note "Flashcard: Golang Logo" [0000000000000000000000000000000000000005]'
      data: eJy8kcGO0zAQhu9+iqE9FCK568RJ2/iCuMCFIxISCEXjeOJY9dpR7C4g8fCoabvLAVZ7Yk4z/3ye+UeOzihYiZdFs2LJn6wCG/ngMY09zobb6DFY7qONbML+ODhP3TK3klLWsjqUjayrtqdGY9tibw6yPJRa7Pq6HqTAgT09eamVcsWOLhgFjz5YdtmTgs37m6Lgw2INPkYbN8zHYLsrtFqxNMY53+o/QDaTx+weqJswj+dbt/eGfXdH5104KtjYuP7XCsx5dvqUKSkGAJDRXrNzcLCRPUlL6V0gBeWe9TFkClnBL75012v4+xa2tD+PmMFESpBHgqK4AudPKAqYaZopUchvLzTn/JK8g6KwcRppLortRXr19Tz12+vt3T0Zh+nOxm16sG9ujroR06igkU2/17oVpRbaCNoPLeq2MXjY7xpDO6pEO9Q4MB3Nz8cr/ofNmTCT6TArqEQluSi5KD+VlZJCCfGFnSbzPOCCoR/PAb8DAAD///FO5Wg=
    - oid: "0000000000000000000000000000000000000006"
      kind: flashcard
      ctime: 2023-01-01T12:30:00Z
      desc: flashcard "Golang Logo" [0000000000000000000000000000000000000006]
      data: eJyUjUFr6zAQhO/6Fft8ek8gR7KckOjy6KmXHguFlmI20lo2USwjKaGH/vhiU+gtpHtaZuabiaMzUMn7blexGe2pHwN1K9horVvd7NVWt83B0vaIhwNat9dqr45yZ9u21xJ79oPcu6UqNsXyS2hbsUQBy3ilbsYyGPCxPjuWw8Uvv+gD5sFicsLHgJMXIfrI8hBT6cpYAhl4XA14WoyCPhsGACDAR9anOBUDLwMWcJEylIGA829iqeIcEs2JMk3lPzuiPRn4FGvDA3Du4zxQ4rxmq/TnbVl5/1tvzuRGzBsf63z1/5hNhIVch8VAIxstpBJSPavGaGmkfGWX2d0OjJOjj1uBrwAAAP//pRSJcQ==
    - oid: "0000000000000000000000000000000000000007"
      kind: note
      ctime: 2023-01-01T12:30:00Z
      desc: 'note "TODO: Conferences" [0000000000000000000000000000000000000007]'
      data: eJykj0+L2zAQxe/6FINzcFvwRn8Sx9K1LT3uZU8txStLY1vEKxl53D/QD1+SbkihZQnsnDT6Pb33lII3UPDb5lCwZVoHA0OqKPlUuRR7zBgdLmy27tiHCduzpVRK7ZRsxF7tpHa476zW1vlGiUZ0vHa7Xa+47dn1ya0tRMGOIXoDpwqMAk1ooHy4/3Bv4P21UMmmFIf2mRcFW8aU6bL/JWQZJ0vhG7azpfH0ubsnz76HY5hCPBooh7T5j7slyqFbCRfDAADIDs+n01QwJHa9Oq9TiGhANsylSBjJwK/qTDcb+CeAnck7+PIpzSNmlyJ8XHOa8eubkWhezHY7XMgdrtu38LjJ+BSix1xJLlXF60rWj5ewdrTLaMAJ7bw7dN4Kx3XPa8FlLbTSTS06FDUerN73B8W65H8aKF9foGQuoyX0rSUDf4CouHgQ0ihuOP/M1tm/LDiZ/nhJ8DsAAP//0SbORQ==
    - oid: "0000000000000000000000000000000000000008"
      kind: reminder
      ctime: 2023-01-01T12:30:00Z
      desc: 'reminder #reminder-2023-06-26 [0000000000000000000000000000000000000008]'
      data: eJyUjsFOhDAURff9igYX6IKhtAxC98YfcKUx5E37gEagTXmY+XwzGDPqYjLT3F3Pue96ZzVPxHWvTlgA89G5EdtNlEqpUsm62KtSNgb3B2gaMLZWRV0cRGXKslMCOnZWrr1VJGz2dKP0mLCII5D7xDYADZr3fjdZZnEx0QVyftY8fXv2YcBo/Myf1ugDvt8PRGHRed7//OxwzR9SRtBrnt5FnNxsMWZSSJWJKpNVykZYqA0YOx8ntC2Q5qfd2ZYXIfSWVzbj8T93rvnFmYhAf4CtqJBafQNrsJeB08bjJeArAAD//8TBhIY=
blobs:
    - oid: "0000000000000000000000000000000000000002"
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
