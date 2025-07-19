package main

func main() {
	ConnectDB()
	StartServer()
	log.Printf("App ready to recieve requests")
}
