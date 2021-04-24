package coordinator

import (
	"html/template"
	"log"
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
)

const pingServerTemp = "https://%s.%s/echo"
const devPingServer = "http://localhost:9000/echo"

//func (c *Server) RelayPacket(u *BrowserClient, packet cws.WSPacket, req func(w *WorkerClient, p cws.WSPacket) cws.WSPacket) cws.WSPacket {
//	packet.SessionID = u.SessionID
//	wc, ok := c.workerClients[u.Worker.Id]
//	if !ok {
//		return cws.EmptyPacket
//	}
//	return req(wc, packet)
//}

func index(conf coordinator.Config) http.Handler {
	tpl, err := template.ParseFiles("./web/index.html")
	if err != nil {
		log.Fatal(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// return 404 on unknown
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		// render index page with some tpl values
		if err = tpl.Execute(w, conf.Coordinator.Analytics); err != nil {
			log.Fatal(err)
		}
	})
}

func static(dir string) http.Handler {
	return http.StripPrefix("/static/", http.FileServer(http.Dir(dir)))
}
