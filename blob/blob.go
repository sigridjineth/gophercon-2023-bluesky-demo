package blob

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	cid "github.com/ipfs/go-cid"
)

type Blob struct {
	Cid         cid.Cid    `json:"-"`
	Size        int        `json:"size"`
	ContentType string     `json:"contentType"`
	Data        []byte     `json:"-"`
	Source      BlobSource `json:"source"`
}

type BlobSource struct {
	Pds string `json:"pds"`
	Did string `json:"did"`
	Url string `json:"url"`
}

func RetrieveBlob(dir string, did string, cidStr string) (Blob, error) {
	blob := Blob{}

	c, err := cid.Decode(cidStr)
	if err != nil {
		return Blob{}, errors.New("invalid cid")
	}
	blob.Cid = c
	if blob.Cid.String() != cidStr {
		return blob, errors.New("invalid cid, not equal")
	}

	if err := blob.FileLoad(dir); err == nil {
		return blob, nil
	}

	// find repository location
	// TODO: remove dependency on ATScan
	resp, err := http.Get(fmt.Sprintf("https://api.atscan.net/%v", did))
	if err != nil {
		return blob, err
	}
	defer resp.Body.Close()
	ds, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return blob, err
	}
	var dat map[string]interface{}
	if err := json.Unmarshal(ds, &dat); err != nil {
		return blob, err
	}
	pd, ok := dat["pds"].([]interface{})
	if !ok || len(pd) == 0 {
		return blob, errors.New("repository location not found")
	}
	pds, ok := pd[0].(string)
	if !ok {
		return blob, errors.New("invalid repository location")
	}
	// update did if differ from resolved (when using handle, for example)
	if did != dat["did"].(string) {
		did = dat["did"].(string)
	}

	// get from PDS
	url := fmt.Sprintf("%v/xrpc/com.atproto.sync.getBlob?did=%v&cid=%v", pds, did, cidStr)
	r, err := http.Get(url)
	if err != nil {
		return blob, err
	}
	defer r.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return blob, fmt.Errorf("PDS return code: %v", resp.StatusCode)
	}

	body, _ := ioutil.ReadAll(r.Body)
	ct := r.Header.Get("Content-Type")

	// check if its not error (in json)
	if strings.Contains(ct, "application/json") {
		var dat map[string]interface{}
		if err := json.Unmarshal(body, &dat); err != nil {
			return blob, err
		}
		if dat["error"].(string) != "" {
			return blob, errors.New(dat["error"].(string))
		}
	}

	bsum, err := blob.Cid.Prefix().Sum(body)
	if err != nil {
		return blob, err
	}
	if !blob.Cid.Equals(bsum) {
		fmt.Printf("Hash of file is different than cid!\n %v != %v\n", bsum.String(), blob.Cid.String())
		return blob, errors.New("Bad hash")
	}

	blob.Data = body
	blob.ContentType = ct
	blob.Size = len(body)
	blob.Source = BlobSource{
		Pds: pds,
		Did: did,
		Url: url,
	}

	blob.FileSave(dir)

	return blob, nil
}

func filePathBase(dir string) string {
	return fmt.Sprintf("%v/blobs", dir)
}

func (b Blob) FilePath(dir string) string {
	return fmt.Sprintf("%v/%v", filePathBase(dir), b.Cid.String())
}

func (b *Blob) FileSave(dir string) error {
	bp := filePathBase(dir)
	if _, err := os.Stat(bp); err != nil {
		os.MkdirAll(bp, 0700)
	}
	path := b.FilePath(dir)

	// write index
	index, _ := json.MarshalIndent(b, "", "  ")
	if err := ioutil.WriteFile(path+".json", index, 0644); err != nil {
		log.Println("Error: ", err)
		return err
	}
	// write blob
	if err := ioutil.WriteFile(path+".blob", b.Data, 0644); err != nil {
		log.Println("Error: ", err)
		return err
	}
	return nil
}

func (b *Blob) FileLoad(dir string) error {
	path := b.FilePath(dir)
	indexFn := path + ".json"
	blobFn := path + ".blob"

	if _, err := os.Stat(indexFn); err != nil {
		return err
	}
	index, err := os.ReadFile(indexFn)
	if err != nil {
		return err
	}
	json.Unmarshal(index, &b)

	if _, err := os.Stat(blobFn); err != nil {
		return err
	}
	data, err := os.ReadFile(blobFn)
	if err != nil {
		return err
	}
	if len(data) != b.Size {
		log.Printf("Size mismatch: %v", b.Cid.String())
		return errors.New("Size mismatch")
	}

	b.Data = data
	return nil
}
