package controller

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	torrentsv1alpha1 "vitoru.fun/torrents/api/v1alpha1"
)

var _ = Describe("TorrentRequest Controller", func() {
	const (
		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		server *ghttp.Server
	)

	BeforeEach(func() {
		server = ghttp.NewServer()
	})

	AfterEach(func() {
		server.Close()
	})

	Context("When creating a valid TorrentRequest", func() {
		It("Should find a torrent and complete successfully", func() {
			ctx := context.Background()

			// 1. Setup Mock Indexer Response
			server.AppendHandlers(
				func(w http.ResponseWriter, req *http.Request) {
					defer GinkgoRecover()
					Expect(req.Method).To(Equal("GET"))
					Expect(req.URL.Path).To(Equal("/search"))
					Expect(req.URL.Query().Get("q")).To(Equal("ubuntu"))
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`
						<html>
							<table>
								<tr class="result">
									<td class="title">Ubuntu 22.04 ISO</td>
									<td><a class="dl" href="magnet:?xt=urn:btih:validmagnetlink">Download</a></td>
									<td>2.5 GB</td>
									<td>100</td>
									<td>10</td>
								</tr>
							</table>
						</html>
					`))
				},
			)

			// 2. Create the Indexer CR pointing to our mock server
			indexer := &torrentsv1alpha1.Indexer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mock-indexer",
					Namespace: "default",
				},
				Spec: torrentsv1alpha1.IndexerSpec{
					Links: []string{server.URL()},
					Caps: torrentsv1alpha1.Caps{
						Modes: torrentsv1alpha1.Modes{
							Search: []string{"q"},
						},
					},
					Search: &torrentsv1alpha1.Search{
						Rows: torrentsv1alpha1.RowsBlock{
							Selector: "tr.result",
						},
						Fields: torrentsv1alpha1.FieldsBlock{
							"title":    torrentsv1alpha1.SelectorBlock{Selector: ".title"},
							"download": torrentsv1alpha1.SelectorBlock{Selector: ".dl", Attribute: "href"},
							"seeders":  torrentsv1alpha1.SelectorBlock{Selector: "td:nth-child(4)"},
						},
						Paths: []torrentsv1alpha1.SearchPathBlock{
							{Path: "/search?q={{ .Keywords }}"},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, indexer)).To(Succeed())

			// 2.1 Set Indexer to Healthy (since IndexerController is not running)
			indexer.Status.Healthy = true
			Expect(k8sClient.Status().Update(ctx, indexer)).To(Succeed())

			// 3. Create the TorrentRequest
			tr := &torrentsv1alpha1.TorrentRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ubuntu-req",
					Namespace: "default",
				},
				Spec: torrentsv1alpha1.TorrentRequestSpec{
					Keywords: "ubuntu",
				},
			}
			Expect(k8sClient.Create(ctx, tr)).To(Succeed())

			// 4. Verify Status becomes Completed
			trLookupKey := types.NamespacedName{Name: "test-ubuntu-req", Namespace: "default"}
			createdTR := &torrentsv1alpha1.TorrentRequest{}

			Eventually(func() string {
				err := k8sClient.Get(ctx, trLookupKey, createdTR)
				if err != nil {
					return ""
				}
				return createdTR.Status.State
			}, timeout, interval).Should(Equal("Completed"))

			// 5. Verify Conditions
			Expect(createdTR.Status.Conditions).To(ContainElement(
				SatisfyAll(
					HaveField("Type", "Ready"),
					HaveField("Status", metav1.ConditionTrue),
					HaveField("Reason", "Found"),
				),
			))

			// 6. Verify Torrent CR was created
			Expect(createdTR.Status.FoundTorrent).NotTo(BeEmpty())

			torrentLookupKey := types.NamespacedName{Name: createdTR.Status.FoundTorrent, Namespace: "default"}
			createdTorrent := &torrentsv1alpha1.Torrent{}
			Eventually(func() error {
				return k8sClient.Get(ctx, torrentLookupKey, createdTorrent)
			}, timeout, interval).Should(Succeed())

			Expect(createdTorrent.Spec.Title).To(Equal("Ubuntu 22.04 ISO"))
			Expect(createdTorrent.Spec.Magnet).To(Equal("magnet:?xt=urn:btih:validmagnetlink"))
		})
	})
})
