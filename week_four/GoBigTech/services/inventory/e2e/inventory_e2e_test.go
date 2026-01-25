package e2e_test

import (
	"context"
	"fmt"
	"time"

	inventorypb "github.com/bulbahal/GoBigTech/services/inventory/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ = Describe("Inventory gRPC E2E", func() {
	var (
		ctx context.Context

		network tc.Network
		mongoC  tc.Container
		invC    tc.Container

		grpcAddr string
		conn     *grpc.ClientConn
		client   inventorypb.InventoryServiceClient
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	AfterEach(func() {
		if conn != nil {
			_ = conn.Close()
		}
		if invC != nil {
			_ = invC.Terminate(ctx)
		}
		if mongoC != nil {
			_ = mongoC.Terminate(ctx)
		}
		if network != nil {
			_ = network.Remove(ctx)
		}
	})

	It("reserves stock and then returns updated availability", func() {
		var err error

		// 1) network
		network, err = tc.GenericNetwork(ctx, tc.GenericNetworkRequest{
			NetworkRequest: tc.NetworkRequest{Name: "inv-e2e-net", CheckDuplicate: true},
		})
		Expect(err).NotTo(HaveOccurred())

		// 2) mongo container
		mongoC, err = tc.GenericContainer(ctx, tc.GenericContainerRequest{
			ContainerRequest: tc.ContainerRequest{
				Image:        "mongo:7",
				ExposedPorts: []string{"27017/tcp"},
				Networks:     []string{"inv-e2e-net"},
				NetworkAliases: map[string][]string{
					"inv-e2e-net": {"mongo"},
				},
				WaitingFor: wait.ForLog("Waiting for connections"),
			},
			Started: true,
		})
		Expect(err).NotTo(HaveOccurred())

		invC, err = tc.GenericContainer(ctx, tc.GenericContainerRequest{
			ContainerRequest: tc.ContainerRequest{
				FromDockerfile: tc.FromDockerfile{
					Context:    "../../services",
					Dockerfile: "inventory/Dockerfile",
				},
				Env: map[string]string{
					"GRPC_ADDR": "0.0.0.0:50051",
					"MONGO_URI": "mongodb://mongo:27017",
					"MONGO_DB":  "appdb",
				},
				ExposedPorts: []string{"50051/tcp"},
				Networks:     []string{"inv-e2e-net"},
				NetworkAliases: map[string][]string{
					"inv-e2e-net": {"inventory"},
				},
				WaitingFor: wait.ForListeningPort("50051/tcp").WithStartupTimeout(40 * time.Second),
			},
			Started: true,
		})
		Expect(err).NotTo(HaveOccurred())

		// 4) dial inventory from host
		host, err := invC.Host(ctx)
		Expect(err).NotTo(HaveOccurred())
		mapped, err := invC.MappedPort(ctx, "50051")
		Expect(err).NotTo(HaveOccurred())
		grpcAddr = fmt.Sprintf("%s:%s", host, mapped.Port())

		conn, err = grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		Expect(err).NotTo(HaveOccurred())
		client = inventorypb.NewInventoryServiceClient(conn)

		productID := "p1"

		before, err := client.GetStock(ctx, &inventorypb.GetStockRequest{ProductId: productID})
		Expect(err).NotTo(HaveOccurred())

		_, err = client.ReserveStock(ctx, &inventorypb.ReserveStockRequest{ProductId: productID, Quantity: 1})
		Expect(err).NotTo(HaveOccurred())

		after, err := client.GetStock(ctx, &inventorypb.GetStockRequest{ProductId: productID})
		Expect(err).NotTo(HaveOccurred())

		Expect(after.Available).To(Equal(before.Available - 1))
	})
})
