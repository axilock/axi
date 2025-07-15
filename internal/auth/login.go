package auth

import (
	"fmt"

	"github.com/axilock/axi/internal/context"
	pb "github.com/axilock/axilock-protos/client"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

func Login(conn *grpc.ClientConn, backendUrl string) (string, error) {
	uuid := uuid.New().String()
	client := pb.NewSessionServiceClient(conn)
	request := pb.CreateAuthSessionRequest{InitToken: uuid}

	authErrChan := make(chan error, 1)
	apiKeyChan := make(chan string, 1)
	go func() {
		ctx, cancel := context.WithAuthRequestTimeout(context.Background())
		defer cancel()
		response, err := client.CreateAuthSession(ctx, &request)
		if err != nil {
			authErrChan <- err
		}
		authErrChan <- nil
		apiKeyChan <- response.GetCliAuthToken()
	}()

	link := fmt.Sprintf("%s/client/login/?clitoken=%s",
		backendUrl,
		uuid,
	)
	fmt.Println("Opening browser @ " + link)
	if err := openbrowser(link); err != nil {
		fmt.Println("Error opening browser. Please open the following URL in your browser:")
		fmt.Println(link)
	}

	err := <-authErrChan
	if err != nil {
		return "", err
	}

	fmt.Println("Authentication successful!")
	return <-apiKeyChan, nil
}

func timeout(cancel context.CancelFunc) {
	fmt.Println("Timed out while trying to authenticate")
	fmt.Println("Please try again in sometime")
	cancel()
}
