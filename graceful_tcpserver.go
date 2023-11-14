//TCP сервер с функционалом безопасной остановки
//при получении сигнала остановки ждёт окончания обработки всех открытых подключений

package gracefultcpserver

import (
	"net"
	"sync"
)

// тип Безопасный TCP сервер
type GracefulTCPServer struct {
	//tcp listener
	listener net.Listener
	//канал сигнала остановки сервера
	quit_channel chan interface{}
	//wait group сервера для контроля завершения всех дочерних горутин
	wg sync.WaitGroup
	//обработчик подключения
	handleConnection func(net.Conn)
}

// создание и запуск экземпляра безопасного TCP сервера
// addr      адрес листенера
// handler   обработчик подключений
// возвращает ссылку на запущенный экземпляр сервера и nil ошибку в случае удачного запуска
// возвращает nil и ошибку в случае ошибки
func RunGracefulTCPServer(addr string, handler func(net.Conn)) (*GracefulTCPServer, error) {
	//создаём экземпляр сервера
	gs := new(GracefulTCPServer)
	//создаём канал сигнала остановки сервера
	gs.quit_channel = make(chan interface{})
	//устанавливаем функцию обработки подключения
	gs.handleConnection = handler
	//создаём tcp listener
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	gs.listener = l
	//стартуем сервер в отдельной горутине
	gs.wg.Add(1)
	go gs.serve()
	return gs, nil
}

// рабочий цикл сервера
func (gs *GracefulTCPServer) serve() {
	defer gs.wg.Done()
	for {
		conn, err := gs.listener.Accept()
		if err != nil {
			select {
			case <-gs.quit_channel:
				//получен сигнал остановки
				return
			default:
				//продолжаем работу
			}
		} else {
			//запускаем обработчик подключения в новой горутине
			gs.wg.Add(1)
			go func() {
				gs.handleConnection(conn)
				gs.wg.Done()
			}()
		}
	}
}

// остановка сервера
func (gs *GracefulTCPServer) Stop() {
	//закрываем канал сигнала остановки сервера
	close(gs.quit_channel)
	//закрываем listener - останавливаем обработку новых подключений
	gs.listener.Close()
	//ждём заверешения всех обработчиков уже открытых подключений
	gs.wg.Wait()
}
