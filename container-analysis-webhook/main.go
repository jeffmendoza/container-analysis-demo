// Copyright 2017 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	grafeas "github.com/Grafeas/client-go/v1alpha1"

	docker "github.com/docker/distribution/manifest/schema2"
	googleAuth "golang.org/x/oauth2/google"

	"k8s.io/api/admission/v1alpha1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	grafeasUrl  string
	tlsCertFile string
	tlsKeyFile  string
	sevThresh   string
	c           *http.Client
)

var (
	authScope = "https://www.googleapis.com/auth/cloud-platform"
)

func main() {
	flag.StringVar(&tlsCertFile, "tls-cert", "/etc/admission-controller/tls/cert.pem", "TLS certificate file.")
	flag.StringVar(&tlsKeyFile, "tls-key", "/etc/admission-controller/tls/key.pem", "TLS key file.")
	flag.StringVar(&sevThresh, "sev-thresh", "HIGH", "Severity Threshold: LOW, MEDIUM, HIGH, or CRITICAL")
	flag.Parse()

	ctx := context.Background()
	var err error
	c, err = googleAuth.DefaultClient(ctx, authScope)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", admissionReviewHandler)
	s := http.Server{
		Addr: ":443",
		TLSConfig: &tls.Config{
			ClientAuth: tls.NoClientCert,
		},
	}
	log.Fatal(s.ListenAndServeTLS(tlsCertFile, tlsKeyFile))
}

func admissionReviewHandler(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ar := v1alpha1.AdmissionReview{}
	if err := json.Unmarshal(data, &ar); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	pod := v1.Pod{}
	if err := json.Unmarshal(ar.Spec.Object.Raw, &pod); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	admissionReviewStatus := v1alpha1.AdmissionReviewStatus{Allowed: false}

	for _, container := range pod.Spec.Containers {
		admit, err := checkAdmit(container.Image)
		if !admit {
			log.Println(err)
			admissionReviewStatus.Allowed = false
			admissionReviewStatus.Result = &metav1.Status{
				Reason: metav1.StatusReasonInvalid,
				Details: &metav1.StatusDetails{
					Causes: []metav1.StatusCause{
						{Message: err.Error()},
					},
				},
			}
			goto done
		}

		log.Printf("No vulns found for image: %s", container.Image)
		admissionReviewStatus.Allowed = true
	}

done:
	ar = v1alpha1.AdmissionReview{
		Status: admissionReviewStatus,
	}

	data, err = json.Marshal(ar)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func checkAdmit(image string) (bool, error) {
	image, err := getDigest(image)
	if err != nil {
		return false, err
	}

	occs, err := getOccurrences(image)
	if err != nil {
		return false, err
	}

	occs = filterOccurrences(occs)

	if len(occs) > 0 {
		return false, fmt.Errorf("Found %v occurrences with severity >= %v for image %v", len(occs), sevThresh, image)
	}

	return true, nil
}

func getDigest(image string) (string, error) {

	// Trim prefix if there to ease parsing, add back at end
	if strings.HasPrefix(image, "https://") {
		image = strings.TrimPrefix(image, "https://")
	}

	if sp := strings.Split(image, "@"); len(sp) == 2 {
		if !strings.HasPrefix(sp[1], "sha256:") {
			return "", fmt.Errorf("Invalid Digest %s Digest should be in form sha256:<sha>", sp[1])
		}
		return fmt.Sprintf("https://%s", image), nil
	}

	sp := strings.Split(image, ":")
	if len(sp) == 1 {
		sp = append(sp, "latest")
	}
	if len(sp) != 2 {
		return "", fmt.Errorf("Malformed image/tag %s too many :", image)
	}
	pth := strings.Split(sp[0], "/")
	if len(pth) != 3 {
		return "", fmt.Errorf("Malformed image %s should be gcr.io/<project>/<name>", sp[0])
	}
	path := fmt.Sprintf("v2/%s/%s/manifests/%s", pth[1], pth[2], sp[1])
	u := &url.URL{
		Scheme: "https",
		Host:   pth[0],
		Path:   path,
	}

	r, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return "", err
	}

	r.Header.Set("Accept", docker.MediaTypeManifest)

	rsp, err := c.Do(r)
	if err != nil {
		return "", err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		return "", fmt.Errorf("non 200 status code: %d", rsp.StatusCode)
	}

	digest := rsp.Header.Get("docker-content-digest")

	return fmt.Sprintf("https://%s@%s", sp[0], digest), nil
}

func getOccurrences(image string) ([]grafeas.Occurrence, error) {
	filter := fmt.Sprintf("kind=\"PACKAGE_VULNERABILITY\" AND resourceUrl=\"%s\"", image)
	sp := strings.Split(strings.TrimPrefix(image, "https://"), "/")
	if len(sp) < 3 {
		return nil, fmt.Errorf("Malformed image %s should be gcr.io/<project>/<name>", image)
	}
	imgProject := sp[1]

	path := fmt.Sprintf("v1alpha1/projects/%s/occurrences", imgProject)

	u := &url.URL{
		Scheme: "https",
		Host:   "containeranalysis.googleapis.com",
		Path:   path,
	}
	q := &url.Values{}
	q.Set("pageSize", "1000") // Just do one page
	q.Set("filter", filter)
	// if token != "" {
	// 	q.Set("pageToken", token)
	// }
	u.RawQuery = q.Encode()

	resp, err := c.Get(u.String())
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("%v", string(data))
		return nil, fmt.Errorf("non 200 status code: %d", resp.StatusCode)
	}

	oResp := grafeas.ListOccurrencesResponse{}
	if err := json.Unmarshal(data, &oResp); err != nil {
		return nil, err
	}

	return oResp.Occurrences, nil
}

var sevOrder = [...]string{"LOW", "MEDIUM", "HIGH", "CRITICAL"}

func sevGE(val, comp string) bool {
	if val == "" {
		return false
	}
	for _, sev := range sevOrder {
		if comp == sev {
			return true
		}
		if val == sev {
			return false
		}
	}
	return false
}

func filterOccurrences(occs []grafeas.Occurrence) []grafeas.Occurrence {
	new := make([]grafeas.Occurrence, 0)
	for _, o := range occs {
		if sevGE(o.VulnerabilityDetails.Severity, sevThresh) {
			new = append(new, o)
		}
	}
	return new
}
