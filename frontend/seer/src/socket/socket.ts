let socket: WebSocket

const reconnectionInterval = 1000

function connect() {
    socket = new WebSocket("ws://192.168.1.61:4000/ws")

    socket.onopen = () => {
        console.log("WebSocket Connected")
    }

    socket.onclose = () => {
        console.log("WebSocket closed, retry every 2 seconds")
        setTimeout(connect, reconnectionInterval)
    }

    socket.onerror = (err) => {
        console.error("WebSocket Error", err)
        socket.close()
    }
}

connect()

export {socket}